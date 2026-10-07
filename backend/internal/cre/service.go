package cre

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/payminto/payminto/backend/internal/observability"
)

// Config is the provider-independent configuration of the service; the wiring derives it from config.CREConfig.
type Config struct {
	Provider         string
	Chain            string
	PublicBaseURL    string
	GatewayID        [32]byte
	ConsumerAddress  common.Address
	ForwarderAddress common.Address
	// Bindings are the per-kind (id, owner, name) the consumer contract enforces; the verifier checks them too.
	Bindings map[Kind]Binding
	// TriggerSigner is a key reference; the service never sees a key.
	TriggerSigner string

	SolvencyInterval      time.Duration
	FinalityBatchInterval time.Duration
	ConversionInterval    time.Duration
	PollInterval          time.Duration
	Confirmations         uint64

	PublicVerifyEnabled bool
	// ReadTokens are the per-workflow bearer credentials; the service keeps only their hashes.
	ReadTokens map[Kind]string

	DegradedFrom string
	MissingKeys  []string
}

// Service is the provider-independent core: it builds workflow inputs from gateway facts, verifies
// and stores what comes back, and answers the dashboard. It never moves money.
type Service struct {
	cfg         Config
	attester    Attester
	store       Store
	verifier    *Verifier
	liabilities LiabilitySource
	reserves    ReserveSource
	deposits    DepositSource
	conversions ConversionSource
	guard       environment.Guard
	events      EventSink
	decimals    Decimals
	tokens      map[Kind][32]byte
	now         func() time.Time
}

// Option configures optional sources; missing ones degrade to honest empties, never to invented facts.
type Option func(*Service)

func WithLiabilities(s LiabilitySource) Option  { return func(x *Service) { x.liabilities = s } }
func WithReserves(s ReserveSource) Option       { return func(x *Service) { x.reserves = s } }
func WithDeposits(s DepositSource) Option       { return func(x *Service) { x.deposits = s } }
func WithConversions(s ConversionSource) Option { return func(x *Service) { x.conversions = s } }

// WithGuard supplies the process environment (ticket 13); the status report names it.
func WithGuard(g environment.Guard) Option  { return func(x *Service) { x.guard = g } }
func WithEvents(s EventSink) Option         { return func(x *Service) { x.events = s } }
func WithDecimals(d Decimals) Option        { return func(x *Service) { x.decimals = d } }
func WithClock(now func() time.Time) Option { return func(x *Service) { x.now = now } }

// NewService wires the core around a provider, a store and a verifier.
func NewService(cfg Config, attester Attester, store Store, verifier *Verifier, opts ...Option) *Service {
	s := &Service{
		cfg: cfg, attester: attester, store: store, verifier: verifier,
		liabilities: NoLiabilities{}, reserves: NoReserves{}, deposits: NoDeposits{}, conversions: NoConversions{},
		events: NoopEvents{}, decimals: defaultDecimals, tokens: map[Kind][32]byte{},
		now: func() time.Time { return time.Now().UTC() },
	}
	for _, o := range opts {
		o(s)
	}
	if s.cfg.ConversionInterval <= 0 {
		s.cfg.ConversionInterval = 15 * time.Minute
	}
	for kind, raw := range cfg.ReadTokens {
		if raw = strings.TrimSpace(raw); raw != "" {
			s.tokens[kind] = sha256.Sum256([]byte(raw))
		}
	}
	if verifier != nil {
		if verifier.Seen == nil {
			verifier.Seen = func(ctx context.Context, hash []byte) (bool, error) { return store.Seen(ctx, cfg.Provider, hash) }
		}
		if verifier.Subjects == nil {
			verifier.Subjects = store
		}
		if verifier.Now == nil {
			verifier.Now = s.now
		}
	}
	return s
}

func (s *Service) Enabled() bool    { return s.cfg.Provider != ProviderNone && s.attester != nil }
func (s *Service) Now() time.Time   { return s.now() }
func (s *Service) Provider() string { return s.cfg.Provider }
func (s *Service) Config() Config   { return s.cfg }

// Authorize checks a per-workflow bearer credential in constant time. No credential configured means refuse.
func (s *Service) Authorize(kind Kind, bearer string) bool {
	want, ok := s.tokens[kind]
	if !ok {
		return false
	}
	got := sha256.Sum256([]byte(strings.TrimSpace(bearer)))
	return subtle.ConstantTimeCompare(want[:], got[:]) == 1
}

func (s *Service) CredentialConfigured(kind Kind) bool {
	_, ok := s.tokens[kind]
	return ok
}

// --- Inputs served to workflows. Each call remembers the subjects it served (SPEC section 6, check 5). ---

// Liabilities serves the latest published checkpoint. Only an authenticated workflow read may publish a
// fresh one (SPEC section 7: the public route never writes); a public read with nothing published is ErrNotFound.
func (s *Service) Liabilities(ctx context.Context, authenticated bool) (Checkpoint, error) {
	if !s.Enabled() {
		return Checkpoint{}, ErrDisabled
	}
	latest, ok, err := s.store.LatestSubject(ctx, KindSolvency)
	if err != nil {
		return Checkpoint{}, err
	}
	if ok && (!authenticated || s.now().Sub(latest.AskedAt) < s.cfg.SolvencyInterval) {
		return checkpointFromSubject(latest), nil
	}
	if !authenticated {
		return Checkpoint{}, fmt.Errorf("%w: no checkpoint published yet", ErrNotFound)
	}
	return s.PublishCheckpoint(ctx)
}

// PublishCheckpoint snapshots the ledger's liabilities now and records the checkpoint as a solvency subject.
func (s *Service) PublishCheckpoint(ctx context.Context) (Checkpoint, error) {
	cp, anomalies, err := BuildCheckpoint(ctx, s.liabilities, s.decimals, s.now())
	if err != nil {
		return Checkpoint{}, err
	}
	for _, a := range anomalies {
		observability.Logger().Error("cre: asset omitted from the liabilities checkpoint", "asset", a.Asset, "reason", a.Reason)
	}
	if err := s.store.RememberSubjects(ctx, []Subject{CheckpointSubject(cp)}); err != nil {
		return Checkpoint{}, err
	}
	return cp, nil
}

func checkpointFromSubject(sub Subject) Checkpoint {
	cp := Checkpoint{ID: sub.ID, Hash: sub.Key, TakenAt: sub.AskedAt}
	// Facts may hold Go values (memory store) or JSON values (Postgres); one JSON round trip normalises them.
	raw, err := json.Marshal(sub.Facts)
	if err != nil {
		return cp
	}
	var facts struct {
		MaxJournalID uint64 `json:"max_journal_id"`
		Assets       []struct {
			Asset       string `json:"asset"`
			Liabilities string `json:"liabilities_minor"`
			Decimals    uint8  `json:"decimals"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(raw, &facts); err != nil {
		return cp
	}
	cp.MaxJournalID = facts.MaxJournalID
	for _, a := range facts.Assets {
		total, ok := parseBig(a.Liabilities)
		if !ok {
			continue
		}
		cp.Assets = append(cp.Assets, AssetTotal{Asset: a.Asset, Liabilities: total, Decimals: a.Decimals})
	}
	return cp
}

// PendingDeposits is the deposit-finality batch; the limit keeps a run under the DON's HTTP quota.
func (s *Service) PendingDeposits(ctx context.Context, limit int) ([]PendingDeposit, error) {
	if !s.Enabled() {
		return nil, ErrDisabled
	}
	if limit <= 0 || limit > 12 {
		limit = 12
	}
	rows, err := s.deposits.PendingDeposits(ctx, limit)
	if err != nil {
		return nil, err
	}
	now := s.now()
	subjects := make([]Subject, 0, len(rows))
	for _, d := range rows {
		subjects = append(subjects, DepositSubject(d, now))
	}
	if err := s.store.RememberSubjects(ctx, subjects); err != nil {
		return nil, err
	}
	return rows, nil
}

// Conversions is the conversion-reference batch.
func (s *Service) Conversions(ctx context.Context, since time.Time, limit int) ([]Conversion, error) {
	if !s.Enabled() {
		return nil, ErrDisabled
	}
	if limit <= 0 || limit > 10 {
		limit = 10
	}
	rows, err := s.conversions.Conversions(ctx, since, limit)
	if err != nil {
		return nil, err
	}
	now := s.now()
	subjects := make([]Subject, 0, len(rows))
	for _, c := range rows {
		subjects = append(subjects, ConversionSubject(c, now))
	}
	if err := s.store.RememberSubjects(ctx, subjects); err != nil {
		return nil, err
	}
	return rows, nil
}

// --- Reports coming back. ---

// Submit verifies a raw attestation and stores its rows; a rejection is returned and nothing is stored.
// Only rows the store actually inserted are returned and announced, so a concurrent duplicate is a replay.
func (s *Service) Submit(ctx context.Context, raw RawAttestation) ([]Attestation, error) {
	if !s.Enabled() {
		return nil, ErrDisabled
	}
	rows, err := s.verifier.Verify(ctx, raw)
	if err != nil {
		recordVerifyFailure(err)
		observability.Logger().Warn("cre: report refused", "kind", raw.Kind, "execution_id", raw.ExecutionID, "reason", Sanitize(err.Error()))
		return nil, err
	}
	inserted, err := s.store.SaveAttestations(ctx, rows)
	if err != nil {
		return nil, err
	}
	if len(inserted) == 0 {
		return nil, ErrReplayed
	}
	for _, r := range inserted {
		s.announce(ctx, r)
	}
	observability.Logger().Info("cre: report recorded", "kind", raw.Kind, "execution_id", raw.ExecutionID, "rows", len(inserted), "tx", common.Bytes2Hex(raw.Evidence.TxHash))
	return inserted, nil
}

func (s *Service) announce(ctx context.Context, r Attestation) {
	payload := map[string]any{
		"attestation_id": r.ID, "kind": string(r.Kind), "subject_type": r.SubjectType, "subject_id": r.SubjectID,
		"status": string(r.Status), "provider": r.Provider, "simulated": r.Simulated, "observed_at": r.ObservedAt.Format(time.RFC3339),
	}
	typ := EventAttestationRecorded
	if r.Status != StatusAttested {
		typ = EventAttestationFailed
		payload["reason"] = r.Reason
		if r.Status == StatusMismatch {
			observability.Logger().Error("cre: attested figures differ from what the gateway served", "kind", r.Kind, "subject_id", r.SubjectID, "reason", r.Reason)
		}
	}
	if err := s.events.Emit(ctx, Event{Type: typ, Payload: payload}); err != nil {
		observability.Logger().Error("cre: emit event", "type", typ, "err", Sanitize(err.Error()))
	}
	recordAttestation(r)
}

// Poll pulls what the provider has seen since the cursor and submits it, in block order. The cursor only
// advances past blocks whose reports were recorded or definitively refused; a report that is not yet final,
// or any store or RPC error, stops the cursor at that block so the next poll sees it again (nothing is lost).
func (s *Service) Poll(ctx context.Context, kind Kind) (int, error) {
	if !s.Enabled() {
		return 0, ErrDisabled
	}
	cursor, err := s.store.GetCursor(ctx, kind)
	if err != nil {
		return 0, err
	}
	raws, next, err := s.attester.Poll(ctx, kind, cursor)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", ErrNotFinal, Sanitize(err.Error()))
	}
	sort.SliceStable(raws, func(i, j int) bool {
		if raws[i].Evidence.BlockNumber != raws[j].Evidence.BlockNumber {
			return raws[i].Evidence.BlockNumber < raws[j].Evidence.BlockNumber
		}
		return raws[i].Evidence.LogIndex < raws[j].Evidence.LogIndex
	})
	recorded := 0
	for _, raw := range raws {
		if raw.Kind == "" {
			raw.Kind = kind
		}
		_, err := s.Submit(ctx, raw)
		switch {
		case err == nil:
			recorded++
		case errors.Is(err, ErrReplayed):
			// Already recorded (a re-read after a partial pass); nothing to do.
		case IsRejection(err):
			if s.cfg.Provider == ProviderChainlink {
				s.recordRefusal(ctx, raw, err)
			}
		default:
			// Not final yet, or infrastructure: stop here and come back.
			if raw.Evidence.BlockNumber > 0 && raw.Evidence.BlockNumber < next.Block {
				next.Block = raw.Evidence.BlockNumber
			}
			if raw.Evidence.BlockNumber == 0 {
				next = cursor
			}
			if err := s.store.SetCursor(ctx, kind, next); err != nil {
				return recorded, err
			}
			return recorded, err
		}
	}
	if next != cursor {
		if err := s.store.SetCursor(ctx, kind, next); err != nil {
			return recorded, err
		}
	}
	return recorded, nil
}

// recordRefusal keeps a definitively refused on-chain report visible to operators as a failed row.
func (s *Service) recordRefusal(ctx context.Context, raw RawAttestation, cause error) {
	meta, _ := DecodeMetadata(raw.Metadata)
	row := Attestation{
		ID: uuid.NewString(), Kind: raw.Kind, SubjectType: "report", SubjectID: "0x" + common.Bytes2Hex(raw.Evidence.TxHash),
		PayloadHash: PayloadHash(raw.Report), Payload: raw.Report, Chain: s.cfg.Chain, TxHash: raw.Evidence.TxHash, BlockNumber: raw.Evidence.BlockNumber,
		WorkflowID: meta.WorkflowID, WorkflowOwner: meta.Owner, ReportID: meta.ReportID, RecordedAt: s.now(), Status: StatusFailed, Provider: s.cfg.Provider,
		Reason: Sanitize(cause.Error()), Item: map[string]any{"refused": true},
	}
	if _, err := s.store.SaveAttestations(ctx, []Attestation{row}); err != nil {
		observability.Logger().Error("cre: record refusal", "err", Sanitize(err.Error()))
	}
}

// Run triggers one workflow with the batch input the gateway would serve it over HTTP.
func (s *Service) Run(ctx context.Context, kind Kind) (Run, error) {
	if !s.Enabled() {
		return Run{}, ErrDisabled
	}
	input, err := s.input(ctx, kind)
	if err != nil {
		return Run{}, err
	}
	started := s.now()
	execID, err := s.attester.Trigger(ctx, kind, input)
	run := Run{Kind: kind, Provider: s.cfg.Provider, ExecutionID: execID, Status: RunAccepted, StartedAt: started}
	if err != nil {
		run.Status = RunFailed
		run.Detail = err.Error()
	}
	recordRun(run, s.now().Sub(started))
	if rerr := s.store.RecordRun(ctx, run); rerr != nil {
		return run, rerr
	}
	return run, err
}

// input is the JSON body a workflow fetches from the pull routes; the same bytes go to Trigger.
func (s *Service) input(ctx context.Context, kind Kind) ([]byte, error) {
	switch kind {
	case KindSolvency:
		cp, err := s.PublishCheckpoint(ctx)
		if err != nil {
			return nil, err
		}
		return json.Marshal(CheckpointJSON(cp, true))
	case KindDepositFinality:
		rows, err := s.PendingDeposits(ctx, 12)
		if err != nil {
			return nil, err
		}
		return json.Marshal(map[string]any{"deposits": DepositsJSON(rows)})
	case KindConversionReference:
		rows, err := s.Conversions(ctx, s.now().Add(-s.cfg.ConversionInterval), 10)
		if err != nil {
			return nil, err
		}
		return json.Marshal(map[string]any{"conversions": ConversionsJSON(rows)})
	}
	return nil, fmt.Errorf("%w: unknown kind %q", ErrInvalidReport, kind)
}

// --- JSON shapes shared by the routes and the trigger input (snake_case). ---

// CheckpointJSON is the liabilities body; the journal counter is served only to the authenticated workflow.
func CheckpointJSON(cp Checkpoint, authenticated bool) map[string]any {
	assets := make([]map[string]any, 0, len(cp.Assets))
	for _, a := range cp.Assets {
		assets = append(assets, map[string]any{"asset": a.Asset, "liabilities_minor": a.Liabilities.String(), "decimals": a.Decimals})
	}
	out := map[string]any{
		"checkpoint_id": cp.ID, "checkpoint_hash": "0x" + common.Bytes2Hex(cp.Hash[:]), "taken_at": cp.TakenAt.UTC().Format(time.RFC3339), "assets": assets,
	}
	if authenticated {
		out["max_journal_id"] = cp.MaxJournalID
	}
	return out
}

func DepositsJSON(rows []PendingDeposit) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, d := range rows {
		amount := "0"
		if d.ExpectedAmountMinor != nil {
			amount = d.ExpectedAmountMinor.String()
		}
		out = append(out, map[string]any{
			"deposit_id": d.DepositID, "chain": d.Chain, "tx": d.Tx, "log_index_or_signature": d.LogIndexOrSig,
			"token": d.Token, "expected_amount_minor": amount, "destination": d.Destination,
		})
	}
	return out
}

func ConversionsJSON(rows []Conversion) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, c := range rows {
		amount := "0"
		if c.AmountMinor != nil {
			amount = c.AmountMinor.String()
		}
		out = append(out, map[string]any{
			"conversion_id": c.ConversionID, "executed_at": c.ExecutedAt.UTC().Format(time.RFC3339), "base": c.Base, "quote": c.Quote,
			"executed_rate_decimal": c.ExecutedRate, "amount_minor": amount,
		})
	}
	return out
}

// --- Dashboard reads. ---

func (s *Service) Attestations(ctx context.Context, kind Kind, limit int) ([]Attestation, error) {
	if !s.Enabled() {
		return nil, ErrDisabled
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.store.ListAttestations(ctx, s.cfg.Provider, kind, limit)
}

func (s *Service) Attestation(ctx context.Context, id string) (Attestation, error) {
	if !s.Enabled() {
		return Attestation{}, ErrDisabled
	}
	a, ok, err := s.store.GetAttestation(ctx, id)
	if err != nil {
		return Attestation{}, err
	}
	if !ok {
		return Attestation{}, ErrNotFound
	}
	return a, nil
}

// WorkflowStatus is one row of the status report.
type WorkflowStatus struct {
	Kind                 Kind
	WorkflowID           string
	WorkflowName         string
	Interval             time.Duration
	CredentialConfigured bool
	LastRun              *Run
	// LastAttestation is the newest row of any status; LastVerified the newest attested one.
	LastAttestation *Attestation
	LastVerified    *Attestation
	// State is never, fresh or stale: stale after two intervals without an attested record (SPEC section 5.1).
	State string
}

type StatusReport struct {
	Enabled              bool
	Provider             string
	DegradedFrom         string
	MissingKeys          []string
	Environment          environment.Environment
	Chain                string
	ConsumerAddress      string
	ForwarderAddress     string
	WorkflowOwner        string
	TriggerSigner        string
	TriggerSignerAddress string
	GatewayID            string
	PublicBaseURL        string
	PublicVerifyEnabled  bool
	Health               Health
	Workflows            []WorkflowStatus
}

func (s *Service) Status(ctx context.Context) (StatusReport, error) {
	rep := StatusReport{
		Enabled: s.Enabled(), Provider: s.cfg.Provider, DegradedFrom: s.cfg.DegradedFrom, MissingKeys: s.cfg.MissingKeys,
		Chain: s.cfg.Chain, TriggerSigner: s.cfg.TriggerSigner, PublicBaseURL: s.cfg.PublicBaseURL,
		PublicVerifyEnabled: s.cfg.PublicVerifyEnabled, Health: Health{Status: HealthOff},
	}
	if s.guard != nil {
		rep.Environment = s.guard.Current()
	}
	if !rep.Enabled {
		return rep, nil
	}
	rep.GatewayID = "0x" + common.Bytes2Hex(s.cfg.GatewayID[:])
	if s.cfg.ConsumerAddress != (common.Address{}) {
		rep.ConsumerAddress = s.cfg.ConsumerAddress.Hex()
	}
	if s.cfg.ForwarderAddress != (common.Address{}) {
		rep.ForwarderAddress = s.cfg.ForwarderAddress.Hex()
	}
	if b, ok := s.cfg.Bindings[KindSolvency]; ok && b.Owner != ([20]byte{}) {
		rep.WorkflowOwner = common.BytesToAddress(b.Owner[:]).Hex()
	}
	rep.Health = s.attester.Health(ctx)
	rep.Health.Message = Sanitize(rep.Health.Message)
	rep.TriggerSignerAddress = rep.Health.SignerAddress
	now := s.now()
	for _, kind := range Kinds {
		ws := WorkflowStatus{Kind: kind, Interval: s.interval(kind), CredentialConfigured: s.CredentialConfigured(kind), State: "never"}
		if b, ok := s.cfg.Bindings[kind]; ok && b.ID != ([32]byte{}) {
			ws.WorkflowID = "0x" + common.Bytes2Hex(b.ID[:])
			ws.WorkflowName = string(b.Name[:])
		}
		if run, ok, err := s.store.LatestRun(ctx, kind); err != nil {
			return rep, err
		} else if ok {
			ws.LastRun = &run
		}
		if att, ok, err := s.store.LatestAttestation(ctx, s.cfg.Provider, kind, ""); err != nil {
			return rep, err
		} else if ok {
			ws.LastAttestation = &att
		}
		// Freshness derives from the latest verified record only; a failed or mismatched row never reads as fresh.
		if att, ok, err := s.store.LatestAttestation(ctx, s.cfg.Provider, kind, StatusAttested); err != nil {
			return rep, err
		} else if ok {
			ws.LastVerified = &att
			ws.State = "fresh"
			if now.Sub(att.RecordedAt) > 2*ws.Interval {
				ws.State = "stale"
			}
		}
		rep.Workflows = append(rep.Workflows, ws)
	}
	return rep, nil
}

func (s *Service) interval(kind Kind) time.Duration {
	switch kind {
	case KindSolvency:
		return s.cfg.SolvencyInterval
	case KindDepositFinality:
		return s.cfg.FinalityBatchInterval
	default:
		return s.cfg.ConversionInterval
	}
}

// SweepStale emits cre.workflow.stale.v1 for workflows whose newest record is older than two intervals.
func (s *Service) SweepStale(ctx context.Context) error {
	if !s.Enabled() {
		return nil
	}
	rep, err := s.Status(ctx)
	if err != nil {
		return err
	}
	for _, w := range rep.Workflows {
		if w.State != "stale" {
			continue
		}
		payload := map[string]any{"kind": string(w.Kind), "provider": s.cfg.Provider, "last_verified_at": w.LastVerified.RecordedAt.Format(time.RFC3339), "interval_seconds": int64(w.Interval.Seconds())}
		if err := s.events.Emit(ctx, Event{Type: EventWorkflowStale, Payload: payload}); err != nil {
			return err
		}
	}
	return nil
}

// Gate is the settlement gate; the no-op until ticket 21b, whatever the provider.
func (s *Service) Gate() SettlementGate { return NoopGate{} }

// PolicyMayRequireAttestation says whether a settlement policy may set require_attestation_above (SPEC section 2).
func (s *Service) PolicyMayRequireAttestation() bool { return s.Enabled() }

// IsDisabled reports ErrDisabled.
func IsDisabled(err error) bool { return errors.Is(err, ErrDisabled) }

// --- Default sources: honest empties. ---

type NoLiabilities struct{}

func (NoLiabilities) LiabilityTotals(context.Context) ([]LedgerTotal, uint64, error) {
	return nil, 0, nil
}

type NoReserves struct{}

func (NoReserves) Reserves(context.Context) ([]Reserve, error) { return nil, nil }

type NoDeposits struct{}

func (NoDeposits) PendingDeposits(context.Context, int) ([]PendingDeposit, error) { return nil, nil }
func (NoDeposits) Deposit(context.Context, string) (PendingDeposit, bool, error) {
	return PendingDeposit{}, false, nil
}

type NoConversions struct{}

func (NoConversions) Conversions(context.Context, time.Time, int) ([]Conversion, error) {
	return nil, nil
}
func (NoConversions) ConversionExists(context.Context, string) (bool, error) { return false, nil }
