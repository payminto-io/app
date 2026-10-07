// Package mock is the in-process provider: it plays the DON, signs with a dev key, and is scriptable.
// Records it produces carry provider=mock and are labelled as such everywhere (SPEC section 10).
package mock

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/payminto/payminto/backend/internal/cre"
)

// ErrNoReserveSource: the mock has nothing to observe reserves with, so a solvency run cannot be honest.
var ErrNoReserveSource = errors.New("mock: no reserve source; custody must supply one before solvency runs")

type queued struct {
	seq   uint64
	ready time.Time
	raw   cre.RawAttestation
}

// Provider is safe for concurrent use.
type Provider struct {
	key       *cre.DevKey
	gatewayID [32]byte
	reserves  cre.ReserveSource
	names     map[cre.Kind]string
	now       func() time.Time

	mu       sync.Mutex
	seq      uint64
	reportID uint16
	queue    map[cre.Kind][]queued
	delay    time.Duration
	failures map[cre.Kind]error
	verdicts map[string]uint8
	health   cre.Health
}

var _ cre.Attester = (*Provider)(nil)

type Option func(*Provider)

func WithReserves(r cre.ReserveSource) Option { return func(p *Provider) { p.reserves = r } }

// WithWorkflowNames sets the workflow.yaml names per kind (defaults: solvency, deposit-finality, conversion-reference).
func WithWorkflowNames(names map[cre.Kind]string) Option {
	return func(p *Provider) {
		for k, v := range names {
			if v != "" {
				p.names[k] = v
			}
		}
	}
}
func WithClock(now func() time.Time) Option { return func(p *Provider) { p.now = now } }
func WithDevKey(k *cre.DevKey) Option       { return func(p *Provider) { p.key = k } }

func New(gatewayID [32]byte, opts ...Option) (*Provider, error) {
	p := &Provider{
		gatewayID: gatewayID, reserves: nil, now: func() time.Time { return time.Now().UTC() },
		names: map[cre.Kind]string{cre.KindSolvency: "solvency", cre.KindDepositFinality: "deposit-finality", cre.KindConversionReference: "conversion-reference"},
		queue: map[cre.Kind][]queued{}, failures: map[cre.Kind]error{}, verdicts: map[string]uint8{},
	}
	for _, o := range opts {
		o(p)
	}
	if p.key == nil {
		k, err := cre.NewDevKey()
		if err != nil {
			return nil, err
		}
		p.key = k
	}
	p.health = cre.Health{Status: cre.HealthOK, Message: "mock provider; attestations are signed by an in-process dev key", SignerAddress: p.key.Address().Hex()}
	return p, nil
}

func (p *Provider) Name() string { return cre.ProviderMock }

// WorkflowID is deterministic per kind so a verifier can be configured before any run.
func (p *Provider) WorkflowID(kind cre.Kind) [32]byte { return cre.SubjectKey("mock:" + string(kind)) }

// WorkflowName is the Keystone name the mock writes into every report's metadata (cre.KeystoneName).
func (p *Provider) WorkflowName(kind cre.Kind) [10]byte { return cre.KeystoneName(p.names[kind]) }

// Binding is what a verifier must be configured with for this mock.
func (p *Provider) Binding(kind cre.Kind) cre.Binding {
	return cre.Binding{ID: p.WorkflowID(kind), Owner: p.key.Address(), Name: p.WorkflowName(kind)}
}

// Owner doubles as the signer: the dev key plays both the workflow owner and the DON.
func (p *Provider) Owner() common.Address { return p.key.Address() }

// ScriptDelay holds produced attestations back from Poll for d after Trigger.
func (p *Provider) ScriptDelay(d time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.delay = d
}

// ScriptFailure makes Trigger of kind fail with err until cleared with nil.
func (p *Provider) ScriptFailure(kind cre.Kind, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err == nil {
		delete(p.failures, kind)
		return
	}
	p.failures[kind] = err
}

// ScriptVerdict sets the deposit-finality verdict for one deposit id; the default is confirmed.
func (p *Provider) ScriptVerdict(depositID string, verdict uint8) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.verdicts[depositID] = verdict
}

// ScriptHealth overrides what Health reports.
func (p *Provider) ScriptHealth(h cre.Health) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if h.SignerAddress == "" {
		h.SignerAddress = p.key.Address().Hex()
	}
	p.health = h
}

func (p *Provider) Health(context.Context) cre.Health {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.health
}

// Trigger produces the report the real workflow would, from the same JSON input, and queues it.
func (p *Provider) Trigger(ctx context.Context, kind cre.Kind, input []byte) (string, error) {
	p.mu.Lock()
	if err, failing := p.failures[kind]; failing {
		p.mu.Unlock()
		return "", err
	}
	p.mu.Unlock()
	report, err := p.produce(ctx, kind, input)
	if err != nil {
		return "", err
	}
	payload, err := cre.EncodeReport(report)
	if err != nil {
		return "", err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.seq++
	p.reportID++
	meta := cre.Metadata{WorkflowID: p.WorkflowID(kind), WorkflowName: p.WorkflowName(kind), Owner: p.key.Address(), ReportID: [2]byte{byte(p.reportID >> 8), byte(p.reportID)}}
	metadata := meta.Encode()
	sig, err := p.key.Sign(metadata, payload)
	if err != nil {
		return "", err
	}
	execID := fmt.Sprintf("mock-%s-%d", kind, p.seq)
	p.queue[kind] = append(p.queue[kind], queued{seq: p.seq, ready: p.now().Add(p.delay), raw: cre.RawAttestation{
		Kind: kind, Metadata: metadata, Report: payload, Evidence: cre.Evidence{Signature: sig}, ExecutionID: execID,
	}})
	return execID, nil
}

// Poll returns queued attestations past the cursor whose delay has elapsed.
func (p *Provider) Poll(_ context.Context, kind cre.Kind, cursor cre.Cursor) ([]cre.RawAttestation, cre.Cursor, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := p.now()
	next := cursor
	var out []cre.RawAttestation
	for _, q := range p.queue[kind] {
		if q.seq <= cursor.Seq {
			continue
		}
		if q.ready.After(now) {
			break
		}
		out = append(out, q.raw)
		next.Seq = q.seq
	}
	return out, next, nil
}

func (p *Provider) produce(ctx context.Context, kind cre.Kind, input []byte) (cre.Report, error) {
	r := cre.Report{Kind: kind, GatewayID: p.gatewayID, ObservedAt: p.now().Truncate(time.Second)}
	switch kind {
	case cre.KindSolvency:
		var in struct {
			CheckpointHash string `json:"checkpoint_hash"`
			Assets         []struct {
				Asset       string `json:"asset"`
				Liabilities string `json:"liabilities_minor"`
				Decimals    uint8  `json:"decimals"`
			} `json:"assets"`
		}
		if err := json.Unmarshal(input, &in); err != nil {
			return r, fmt.Errorf("%w: solvency input: %v", cre.ErrInvalidReport, err)
		}
		hash := common.HexToHash(in.CheckpointHash)
		// No reserve source means the reserves are unknown; the mock refuses to invent a figure for them.
		if p.reserves == nil {
			return r, ErrNoReserveSource
		}
		reserves, err := p.reserves.Reserves(ctx)
		if err != nil {
			return r, err
		}
		byAsset := map[string]*big.Int{}
		for _, res := range reserves {
			if res.Amount == nil {
				continue
			}
			if byAsset[res.Asset] == nil {
				byAsset[res.Asset] = new(big.Int)
			}
			byAsset[res.Asset].Add(byAsset[res.Asset], res.Amount)
		}
		items := make([]cre.SolvencyItem, 0, len(in.Assets))
		for _, a := range in.Assets {
			liab, ok := new(big.Int).SetString(a.Liabilities, 10)
			if !ok {
				return r, fmt.Errorf("%w: liabilities %q", cre.ErrInvalidReport, a.Liabilities)
			}
			reserve, known := byAsset[a.Asset]
			if !known {
				// An asset with no observed reserve is left out rather than attested with an invented zero.
				continue
			}
			items = append(items, cre.SolvencyItem{CheckpointHash: hash, Asset: cre.LabelKey(a.Asset), Liabilities: liab, Reserves: reserve, Decimals: a.Decimals})
		}
		if len(items) == 0 {
			return r, fmt.Errorf("%w: no asset in the checkpoint has an observed reserve", ErrNoReserveSource)
		}
		r.Items = items
	case cre.KindDepositFinality:
		var in struct {
			Deposits []struct {
				DepositID   string `json:"deposit_id"`
				Chain       string `json:"chain"`
				Tx          string `json:"tx"`
				Token       string `json:"token"`
				Amount      string `json:"expected_amount_minor"`
				Destination string `json:"destination"`
			} `json:"deposits"`
		}
		if err := json.Unmarshal(input, &in); err != nil {
			return r, fmt.Errorf("%w: deposit input: %v", cre.ErrInvalidReport, err)
		}
		p.mu.Lock()
		verdicts := make(map[string]uint8, len(p.verdicts))
		for k, v := range p.verdicts {
			verdicts[k] = v
		}
		p.mu.Unlock()
		items := make([]cre.DepositItem, 0, len(in.Deposits))
		for i, d := range in.Deposits {
			amount, ok := new(big.Int).SetString(d.Amount, 10)
			if !ok {
				return r, fmt.Errorf("%w: amount %q", cre.ErrInvalidReport, d.Amount)
			}
			verdict, scripted := verdicts[d.DepositID]
			if !scripted {
				verdict = cre.VerdictConfirmed
			}
			items = append(items, cre.DepositItem{
				DepositID: cre.SubjectKey(d.DepositID), ChainID: cre.LabelKey(d.Chain), TxRef: cre.SubjectKey(d.Tx), Token: cre.LabelKey(d.Token),
				Amount: amount, Destination: cre.SubjectKey(d.Destination), SlotOrBlock: uint64(1000 + i), Verdict: verdict,
			})
		}
		r.Items = items
	case cre.KindConversionReference:
		var in struct {
			Conversions []struct {
				ConversionID string `json:"conversion_id"`
				Base         string `json:"base"`
				Quote        string `json:"quote"`
				Rate         string `json:"executed_rate_decimal"`
			} `json:"conversions"`
		}
		if err := json.Unmarshal(input, &in); err != nil {
			return r, fmt.Errorf("%w: conversion input: %v", cre.ErrInvalidReport, err)
		}
		items := make([]cre.ConversionItem, 0, len(in.Conversions))
		for i, c := range in.Conversions {
			rate, ok := scaleDecimal(c.Rate, 8)
			if !ok {
				return r, fmt.Errorf("%w: rate %q", cre.ErrInvalidReport, c.Rate)
			}
			// The mock has no feed: it echoes the executed rate as the reference with zero deviation and a zero feed address.
			items = append(items, cre.ConversionItem{
				ConversionID: cre.SubjectKey(c.ConversionID), Pair: cre.LabelKey(c.Base + "/" + c.Quote), ReferenceRate: rate, ReferenceDecimals: 8,
				DeviationBps: new(big.Int), Feed: common.Address{}, RoundID: big.NewInt(int64(i + 1)),
			})
		}
		r.Items = items
	default:
		return r, fmt.Errorf("%w: kind %q", cre.ErrInvalidReport, kind)
	}
	return r, nil
}

// scaleDecimal turns a decimal string into an integer at the given scale, refusing anything finer.
func scaleDecimal(s string, decimals int) (*big.Int, bool) {
	rat, ok := new(big.Rat).SetString(s)
	if !ok {
		return nil, false
	}
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	rat.Mul(rat, new(big.Rat).SetInt(scale))
	if !rat.IsInt() {
		return nil, false
	}
	return new(big.Int).Set(rat.Num()), true
}
