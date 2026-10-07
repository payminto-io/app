// Package cre is the optional attestation module: an operator-independent signature on numbers the
// gateway already holds. It never computes a balance, moves money or decides a settlement (docs/cre/SPEC.md).
package cre

import (
	"context"
	"errors"
	"math/big"
	"time"
)

// Kind names a workflow; the value is the wire name used in routes, rows and events.
type Kind string

const (
	KindSolvency            Kind = "solvency"
	KindDepositFinality     Kind = "deposit_finality"
	KindConversionReference Kind = "conversion_reference"
)

// Kinds is every workflow in SPEC priority order.
var Kinds = []Kind{KindSolvency, KindDepositFinality, KindConversionReference}

// Code is the uint8 that opens every report payload (SPEC section 5).
func (k Kind) Code() uint8 {
	switch k {
	case KindSolvency:
		return 1
	case KindDepositFinality:
		return 2
	case KindConversionReference:
		return 3
	}
	return 0
}

// KindFromCode is the inverse of Code; ok is false for an unknown code.
func KindFromCode(code uint8) (Kind, bool) {
	for _, k := range Kinds {
		if k.Code() == code {
			return k, true
		}
	}
	return "", false
}

// ParseKind accepts the wire name or the hyphenated workflow folder name.
func ParseKind(s string) (Kind, bool) {
	switch s {
	case "solvency":
		return KindSolvency, true
	case "deposit_finality", "deposit-finality":
		return KindDepositFinality, true
	case "conversion_reference", "conversion-reference":
		return KindConversionReference, true
	}
	return "", false
}

// SubjectType of an attestation row.
const (
	SubjectLedgerCheckpoint = "ledger_checkpoint"
	SubjectDeposit          = "deposit"
	SubjectConversion       = "conversion"
)

func (k Kind) SubjectType() string {
	switch k {
	case KindSolvency:
		return SubjectLedgerCheckpoint
	case KindDepositFinality:
		return SubjectDeposit
	default:
		return SubjectConversion
	}
}

// Status of an attestation row.
type Status string

const (
	StatusPending  Status = "pending"
	StatusAttested Status = "attested"
	StatusFailed   Status = "failed"
	StatusStale    Status = "stale"
)

// Provider names; they match config.CREProvider*.
const (
	ProviderNone      = "none"
	ProviderMock      = "mock"
	ProviderChainlink = "chainlink"
)

// Deposit verdicts (SPEC section 5.2).
const (
	VerdictConfirmed uint8 = 1
	VerdictNotFound  uint8 = 2
	VerdictMismatch  uint8 = 3
)

// Attestation is one verified (or refused) record, one subject per row.
type Attestation struct {
	ID            string
	Kind          Kind
	SubjectType   string
	SubjectID     string
	PayloadHash   []byte
	Payload       []byte
	Chain         string
	TxHash        []byte
	BlockNumber   uint64
	WorkflowID    [32]byte
	WorkflowOwner [20]byte
	ReportID      [2]byte
	ObservedAt    time.Time
	RecordedAt    time.Time
	Status        Status
	Provider      string
	Simulated     bool
	// Reason explains a failed or stale status; empty when attested.
	Reason string
	// Item is the decoded payload item for this subject.
	Item any
}

// Evidence is how the verifier knows a report is real: a log the gateway read over its own RPC,
// or (mock only) a signature by the mock provider's dev key.
type Evidence struct {
	// OnChain evidence.
	Emitter     [20]byte
	TxHash      []byte
	BlockNumber uint64
	LogIndex    uint
	// HeadBlock is the chain head the reader saw when it read the log; confirmations are judged against it.
	HeadBlock uint64
	// Signature evidence (mock): 65-byte secp256k1 signature over keccak256(metadata || report).
	Signature []byte
}

// RawAttestation is what a provider hands the verifier: exactly the bytes the consumer contract saw.
type RawAttestation struct {
	Kind     Kind
	Metadata []byte
	Report   []byte
	Evidence Evidence
	// ExecutionID is the provider's run id when known.
	ExecutionID string
	Simulated   bool
}

// Cursor is a provider-specific position; a provider returns the next one from Poll.
type Cursor struct {
	Block uint64
	Seq   uint64
}

// HealthStatus is a coarse provider state for the dashboard.
type HealthStatus string

const (
	HealthOK       HealthStatus = "ok"
	HealthDegraded HealthStatus = "degraded"
	HealthDown     HealthStatus = "down"
	HealthOff      HealthStatus = "off"
)

type Health struct {
	Status  HealthStatus
	Message string
	// SignerAddress is the trigger signer's address when the provider can resolve it.
	SignerAddress string
}

// Attester is the provider port. It never decides anything.
type Attester interface {
	Name() string
	// Trigger asks the provider to run a workflow with a batch input. Returns a provider execution id.
	Trigger(ctx context.Context, kind Kind, input []byte) (string, error)
	// Poll returns attestations the provider has seen written since the cursor.
	Poll(ctx context.Context, kind Kind, cursor Cursor) ([]RawAttestation, Cursor, error)
	Health(ctx context.Context) Health
}

var (
	ErrDisabled        = errors.New("cre: attestation module is off")
	ErrUnsupported     = errors.New("cre: provider does not support this operation")
	ErrNotFound        = errors.New("cre: not found")
	ErrUnauthorized    = errors.New("cre: unauthorized")
	ErrInvalidReport   = errors.New("cre: invalid report")
	ErrForged          = errors.New("cre: report evidence does not verify")
	ErrReplayed        = errors.New("cre: report already recorded")
	ErrWrongWorkflow   = errors.New("cre: report from an unexpected workflow")
	ErrWrongOwner      = errors.New("cre: report from an unexpected workflow owner")
	ErrWrongGateway    = errors.New("cre: report for another gateway")
	ErrWrongEmitter    = errors.New("cre: log emitted by an unexpected contract")
	ErrUnconfirmed     = errors.New("cre: log is not yet confirmed")
	ErrStale           = errors.New("cre: report is too old")
	ErrUnknownSubject  = errors.New("cre: report names a subject the gateway never asked about")
	ErrSubjectMismatch = errors.New("cre: attested facts differ from what the gateway credited")
)

// --- Facts the module reads from the rest of the gateway, each through a narrow port. ---

// AssetTotal is the ledger's liability per asset in that asset's own units.
type AssetTotal struct {
	Asset       string
	Liabilities *big.Int
	Decimals    uint8
}

// Checkpoint is a published liabilities snapshot; its hash is what a solvency report carries.
type Checkpoint struct {
	ID           string
	Hash         [32]byte
	TakenAt      time.Time
	MaxJournalID uint64
	Assets       []AssetTotal
}

// LiabilitySource reads the ledger's liabilities; the ledger module implements the read.
type LiabilitySource interface {
	// LiabilityTotals sums member liability accounts per asset at the current journal head.
	LiabilityTotals(ctx context.Context) ([]LedgerTotal, uint64, error)
}

// LedgerTotal is a per-asset decimal total as the ledger holds it (before minor-unit scaling).
type LedgerTotal struct {
	Asset string
	// Total is a decimal string in the asset's major units, as the ledger stores amounts.
	Total string
}

// Reserve is a custody balance the solvency workflow compares against; the custody module provides it.
type Reserve struct {
	Asset    string
	Address  string
	Chain    string
	Amount   *big.Int
	Decimals uint8
}

// ReserveSource is implemented by custody (ticket 08); the mock is for tests and the compose profile.
type ReserveSource interface {
	Reserves(ctx context.Context) ([]Reserve, error)
}

// PendingDeposit is a credited deposit a policy wants attested.
type PendingDeposit struct {
	DepositID           string
	Chain               string
	Tx                  string
	LogIndexOrSig       string
	Token               string
	ExpectedAmountMinor *big.Int
	Destination         string
}

// DepositSource is implemented by the switch; the default has nothing pending.
type DepositSource interface {
	PendingDeposits(ctx context.Context, limit int) ([]PendingDeposit, error)
	// Deposit returns what the gateway credited for an id; ok is false for an unknown id.
	Deposit(ctx context.Context, depositID string) (PendingDeposit, bool, error)
}

// Conversion is an executed trade the conversion-reference workflow stamps.
type Conversion struct {
	ConversionID string
	ExecutedAt   time.Time
	Base, Quote  string
	ExecutedRate string
	AmountMinor  *big.Int
}

// ConversionSource is implemented by the conversion module; the default has none.
type ConversionSource interface {
	Conversions(ctx context.Context, since time.Time, limit int) ([]Conversion, error)
	ConversionExists(ctx context.Context, conversionID string) (bool, error)
}

// Environment is live or test; ticket 13 replaces the config-backed default.
type Environment string

const (
	EnvironmentLive Environment = "live"
	EnvironmentTest Environment = "test"
)

type EnvironmentSource interface {
	Environment(ctx context.Context) Environment
}

// --- Settlement gate: ticket 21b implements the attestation-aware gate; this ticket ships the no-op. ---

// SettlementSubject is what settlement asks about.
type SettlementSubject struct {
	DepositID   string
	AmountMinor *big.Int
	Asset       string
}

type GateDecision string

const (
	GateProceed GateDecision = "proceed"
	GateWait    GateDecision = "await_attestation"
	GateFreeze  GateDecision = "freeze"
)

// SettlementGate is consulted by settlement before releasing a deposit's funds.
type SettlementGate interface {
	Decide(ctx context.Context, subject SettlementSubject) (GateDecision, error)
}

// NoopGate never waits; it is the gate for every provider until ticket 21b.
type NoopGate struct{}

func (NoopGate) Decide(context.Context, SettlementSubject) (GateDecision, error) {
	return GateProceed, nil
}

// --- Events (SPEC section 9), delivered through the gateway's emitter by the wiring. ---

const (
	EventAttestationRecorded = "cre.attestation.recorded.v1"
	EventAttestationFailed   = "cre.attestation.failed.v1"
	EventWorkflowStale       = "cre.workflow.stale.v1"
)

type Event struct {
	Type    string
	Payload map[string]any
}

type EventSink interface {
	Emit(ctx context.Context, e Event) error
}

// NoopEvents drops events; the wiring supplies the real emitter.
type NoopEvents struct{}

func (NoopEvents) Emit(context.Context, Event) error { return nil }
