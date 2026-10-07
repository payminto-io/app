package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/cre"
	"github.com/payminto/payminto/backend/internal/cre/chainlink"
	"github.com/payminto/payminto/backend/internal/cre/mock"
	"github.com/payminto/payminto/backend/internal/cre/none"
	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/payminto/payminto/backend/internal/ledger"
)

// CREModule is the wired attestation module; config keys are documented in internal/cre/README.md.
// With provider none, Service is a no-op core: no worker, no routes, no table reads.
type CREModule struct {
	Service  *cre.Service
	Provider string
	// Mock is set only for the mock provider so tests and the compose profile can script it.
	Mock *mock.Provider
	// Gate is the settlement gate; the no-op until ticket 21b.
	Gate cre.SettlementGate
}

// Enabled is false for provider none: the gateway is then byte-for-byte what it is without the module.
func (m *CREModule) Enabled() bool { return m != nil && m.Service != nil && m.Service.Enabled() }

// CREOptions carries the sources other modules provide; nil fields fall back to honest empties.
type CREOptions struct {
	// Liabilities overrides the ledger read (tests); nil uses deps.Ledger.
	Liabilities cre.LiabilitySource
	Reserves    cre.ReserveSource
	Deposits    cre.DepositSource
	Conversions cre.ConversionSource
	Signer      chainlink.Signer
	// Reader overrides the RPC log reader (tests); nil dials CRE_CHAIN_RPC_URL for the chainlink provider.
	Reader chainlink.LogReader
	Store  cre.Store
}

// WireCRE builds the module for the configured provider (docs/cre/SPEC.md sections 2 and 3).
func WireCRE(deps Deps, opts CREOptions) (*CREModule, error) {
	if deps.Config == nil {
		return nil, fmt.Errorf("modules: cre needs Config")
	}
	cfg := deps.Config.CRE
	provider := cfg.Provider
	if !cfg.Enabled {
		provider = config.CREProviderNone
	}
	if deps.Environment != nil && deps.Environment.Guard != nil {
		if err := deps.Environment.Guard.RequireProvider("cre", provider); err != nil {
			return nil, err
		}
	}
	if provider == config.CREProviderNone {
		svc := cre.NewService(cre.Config{Provider: cre.ProviderNone}, none.New(), cre.NewMemoryStore(), nil, guardOption(deps))
		return &CREModule{Service: svc, Provider: provider, Gate: cre.NoopGate{}}, nil
	}
	if deps.DB == nil && opts.Store == nil {
		return nil, fmt.Errorf("modules: cre provider %s needs DB", provider)
	}
	decimals, err := cre.ParseDecimals(cfg.AssetDecimals)
	if err != nil {
		return nil, err
	}
	publicBase := cfg.PublicBaseURL
	if publicBase == "" {
		publicBase = fmt.Sprintf("http://localhost:%d", deps.Config.Server.Port)
	}
	sc := cre.Config{
		Provider: provider, Chain: cfg.Chain, PublicBaseURL: publicBase, GatewayID: cre.GatewayID(publicBase),
		TriggerSigner: cfg.TriggerSigner, SolvencyInterval: cfg.SolvencyInterval, FinalityBatchInterval: cfg.FinalityBatchInterval,
		PollInterval: cfg.PollInterval, Confirmations: cfg.VerifyConfirmations,
		PublicVerifyEnabled: cfg.PublicVerifyEnabled, DegradedFrom: cfg.DegradedFrom, MissingKeys: cfg.MissingKeys,
		ReadTokens: map[cre.Kind]string{
			cre.KindSolvency: cfg.ReadTokenSolvency, cre.KindDepositFinality: cfg.ReadTokenDepositFinality, cre.KindConversionReference: cfg.ReadTokenConversionReference,
		},
		Bindings: map[cre.Kind]cre.Binding{},
	}
	names := map[cre.Kind]string{cre.KindSolvency: cfg.WorkflowNameSolvency, cre.KindDepositFinality: cfg.WorkflowNameDepositFinality, cre.KindConversionReference: cfg.WorkflowNameConversionReference}
	store := opts.Store
	if store == nil {
		store = cre.NewPostgresStore(deps.DB)
	}
	live := deps.Environment != nil && deps.Environment.Environment == environment.Live
	verifier := &cre.Verifier{Provider: provider, GatewayID: sc.GatewayID, Chain: sc.Chain, Bindings: sc.Bindings, SimulatedForwarder: cfg.ForwarderSimulated, Live: live}
	sc.ForwarderSimulated = cfg.ForwarderSimulated

	var attester cre.Attester
	var mockProvider *mock.Provider
	switch provider {
	case config.CREProviderMock:
		mockOpts := []mock.Option{mock.WithWorkflowNames(names)}
		if opts.Reserves != nil {
			// Without a reserve source the mock refuses solvency runs rather than attesting an invented zero.
			mockOpts = append(mockOpts, mock.WithReserves(opts.Reserves))
		}
		m, err := mock.New(sc.GatewayID, mockOpts...)
		if err != nil {
			return nil, err
		}
		for _, k := range cre.Kinds {
			sc.Bindings[k] = m.Binding(k)
		}
		verifier.MockSigner = m.Owner()
		attester, mockProvider = m, m
	case config.CREProviderChainlink:
		sc.ConsumerAddress = common.HexToAddress(cfg.ConsumerAddress)
		sc.ForwarderAddress = common.HexToAddress(cfg.ForwarderAddress)
		ids := map[cre.Kind]string{cre.KindSolvency: cfg.WorkflowIDSolvency, cre.KindDepositFinality: cfg.WorkflowIDDepositFinality, cre.KindConversionReference: cfg.WorkflowIDConversionReference}
		owners := map[cre.Kind]string{cre.KindSolvency: cfg.WorkflowOwnerSolvency, cre.KindDepositFinality: cfg.WorkflowOwnerDepositFinality, cre.KindConversionReference: cfg.WorkflowOwnerConversionReference}
		for _, k := range cre.Kinds {
			owner := owners[k]
			if owner == "" {
				owner = cfg.WorkflowOwner
			}
			sc.Bindings[k] = cre.Binding{ID: common.HexToHash(strings.TrimPrefix(ids[k], "0x")), Owner: common.HexToAddress(owner), Name: cre.KeystoneName(names[k])}
		}
		verifier.Consumer = sc.ConsumerAddress
		for k, b := range sc.Bindings {
			if live && cre.IsSimulatorBinding(b) {
				return nil, fmt.Errorf("modules: cre binding for %s is the simulator's identity; refused in live", k)
			}
		}
		reader := opts.Reader
		if reader == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			r, err := chainlink.DialReader(ctx, cfg.ChainRPCURL, cfg.VerifyConfirmations)
			if err != nil {
				return nil, err
			}
			reader = r
		}
		workflowIDs := map[cre.Kind][32]byte{}
		for k, b := range sc.Bindings {
			workflowIDs[k] = b.ID
		}
		attester = chainlink.New(chainlink.Config{
			GatewayURL: cfg.GatewayURL, WorkflowIDs: workflowIDs, KeyRef: cfg.TriggerSigner, Consumer: sc.ConsumerAddress, GatewayID: sc.GatewayID, StartBlock: cfg.StartBlock,
		}, opts.Signer, reader)
	default:
		return nil, fmt.Errorf("modules: unknown CRE provider %q", provider)
	}

	serviceOpts := []cre.Option{cre.WithDecimals(decimals), guardOption(deps)}
	switch {
	case opts.Liabilities != nil:
		serviceOpts = append(serviceOpts, cre.WithLiabilities(opts.Liabilities))
	case deps.Ledger != nil:
		serviceOpts = append(serviceOpts, cre.WithLiabilities(ledgerLiabilities{deps.Ledger}))
	}
	if opts.Reserves != nil {
		serviceOpts = append(serviceOpts, cre.WithReserves(opts.Reserves))
	}
	if opts.Deposits != nil {
		serviceOpts = append(serviceOpts, cre.WithDeposits(opts.Deposits))
	}
	if opts.Conversions != nil {
		serviceOpts = append(serviceOpts, cre.WithConversions(opts.Conversions))
	}
	if deps.EmitEvent != nil {
		serviceOpts = append(serviceOpts, cre.WithEvents(eventSink(deps.EmitEvent)))
	}
	svc := cre.NewService(sc, attester, store, verifier, serviceOpts...)
	return &CREModule{Service: svc, Provider: provider, Mock: mockProvider, Gate: svc.Gate()}, nil
}

func guardOption(deps Deps) cre.Option {
	if deps.Environment != nil && deps.Environment.Guard != nil {
		return cre.WithGuard(deps.Environment.Guard)
	}
	if deps.Config != nil {
		if env, err := environment.Parse(deps.Config.Gateway.Environment); err == nil {
			if g, err := environment.NewGuard(env); err == nil {
				return cre.WithGuard(g)
			}
		}
	}
	return func(*cre.Service) {}
}

// ledgerLiabilities adapts the ledger's read to the module's port.
type ledgerLiabilities struct{ ledger *ledger.Service }

func (l ledgerLiabilities) LiabilityTotals(ctx context.Context) (cre.LedgerSnapshot, error) {
	snap, err := l.ledger.LiabilityTotals(ctx)
	if err != nil {
		return cre.LedgerSnapshot{}, err
	}
	out := cre.LedgerSnapshot{Head: snap.Head, TakenAt: snap.TakenAt, Totals: make([]cre.LedgerTotal, 0, len(snap.Totals))}
	for _, t := range snap.Totals {
		out.Totals = append(out.Totals, cre.LedgerTotal{Asset: t.Asset, Total: t.Total.String()})
	}
	return out, nil
}

type eventSink func(eventType string, payload any) error

func (f eventSink) Emit(_ context.Context, e cre.Event) error { return f(e.Type, e.Payload) }
