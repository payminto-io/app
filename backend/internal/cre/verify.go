package cre

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
)

// Subject is something the gateway asked a workflow about; attestations are matched back to it by key.
type Subject struct {
	Kind    Kind
	Key     [32]byte
	ID      string
	Facts   map[string]any
	AskedAt time.Time
}

// SubjectIndex remembers subjects served to workflows (SPEC section 6, check 5).
type SubjectIndex interface {
	RememberSubjects(ctx context.Context, subjects []Subject) error
	LookupSubject(ctx context.Context, kind Kind, key [32]byte) (Subject, bool, error)
}

// Verifier applies the six checks of SPEC section 6 to a raw attestation before anything is stored.
type Verifier struct {
	Provider      string
	GatewayID     [32]byte
	Consumer      [20]byte
	Owner         [20]byte
	WorkflowIDs   map[Kind][32]byte
	Confirmations uint64
	MaxReportAge  time.Duration
	// MaxClockSkew bounds how far ahead of the gateway clock a DON timestamp may be.
	MaxClockSkew time.Duration
	// MockSigner is the only accepted signer of signature evidence; set for the mock provider.
	MockSigner common.Address
	Subjects   SubjectIndex
	// Seen reports whether a payload hash is already recorded.
	Seen func(ctx context.Context, payloadHash []byte) (bool, error)
	// LatestObservedAt is the newest accepted observation per kind; the contract requires strictly newer.
	LatestObservedAt func(ctx context.Context, kind Kind) (time.Time, bool, error)
	Now              func() time.Time
	// Chain is stamped on the rows.
	Chain string
}

// Verify returns one row per item. A whole-report failure is an error; a per-subject failure is a
// row with StatusFailed and a Reason, so the mismatch is kept and raised rather than dropped.
func (v *Verifier) Verify(ctx context.Context, raw RawAttestation) ([]Attestation, error) {
	if v.Provider == ProviderNone || v.Provider == "" {
		return nil, ErrDisabled
	}
	meta, err := DecodeMetadata(raw.Metadata)
	if err != nil {
		return nil, err
	}
	report, err := DecodeReport(raw.Report)
	if err != nil {
		return nil, err
	}
	if raw.Kind != "" && raw.Kind != report.Kind {
		return nil, fmt.Errorf("%w: payload kind %s under %s", ErrWrongWorkflow, report.Kind, raw.Kind)
	}
	want, ok := v.WorkflowIDs[report.Kind]
	if !ok || want == ([32]byte{}) || meta.WorkflowID != want {
		return nil, fmt.Errorf("%w: %x", ErrWrongWorkflow, meta.WorkflowID)
	}
	if meta.Owner != v.Owner {
		return nil, fmt.Errorf("%w: %s", ErrWrongOwner, common.BytesToAddress(meta.Owner[:]))
	}
	if report.GatewayID != v.GatewayID {
		return nil, fmt.Errorf("%w: %x", ErrWrongGateway, report.GatewayID)
	}
	if err := v.checkEvidence(raw); err != nil {
		return nil, err
	}
	now := v.now()
	if now.Sub(report.ObservedAt) > v.MaxReportAge {
		return nil, fmt.Errorf("%w: observed %s", ErrStale, report.ObservedAt.Format(time.RFC3339))
	}
	if report.ObservedAt.Sub(now) > v.skew() {
		return nil, fmt.Errorf("%w: observed_at %s is in the future", ErrInvalidReport, report.ObservedAt.Format(time.RFC3339))
	}
	hash := PayloadHash(raw.Report)
	if v.Seen != nil {
		seen, err := v.Seen(ctx, hash)
		if err != nil {
			return nil, err
		}
		if seen {
			return nil, ErrReplayed
		}
	}
	if v.LatestObservedAt != nil {
		latest, ok, err := v.LatestObservedAt(ctx, report.Kind)
		if err != nil {
			return nil, err
		}
		if ok && !report.ObservedAt.After(latest) {
			return nil, fmt.Errorf("%w: observed %s is not newer than %s", ErrReplayed, report.ObservedAt.Format(time.RFC3339), latest.Format(time.RFC3339))
		}
	}
	rows, err := v.rows(ctx, report, meta, raw, hash, now)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("%w: report carries no items", ErrInvalidReport)
	}
	return rows, nil
}

func (v *Verifier) now() time.Time {
	if v.Now != nil {
		return v.Now().UTC()
	}
	return time.Now().UTC()
}

func (v *Verifier) skew() time.Duration {
	if v.MaxClockSkew > 0 {
		return v.MaxClockSkew
	}
	return 5 * time.Minute
}

// checkEvidence is check 1, 2 and 6 for chainlink (own-RPC log, emitter, confirmations) and the dev-key signature for mock.
func (v *Verifier) checkEvidence(raw RawAttestation) error {
	switch v.Provider {
	case ProviderChainlink:
		ev := raw.Evidence
		if len(ev.TxHash) == 0 {
			return fmt.Errorf("%w: no transaction behind the report", ErrForged)
		}
		if ev.Emitter != v.Consumer {
			return fmt.Errorf("%w: %s", ErrWrongEmitter, common.BytesToAddress(ev.Emitter[:]))
		}
		if ev.ReportHash == ([32]byte{}) || ev.ReportHash != [32]byte(PayloadHash(raw.Report)) {
			return fmt.Errorf("%w: rebuilt report does not hash to the contract's reportHash", ErrForged)
		}
		if ev.HeadBlock < ev.BlockNumber || ev.HeadBlock-ev.BlockNumber < v.Confirmations {
			return fmt.Errorf("%w: block %d, head %d, need %d confirmations", ErrUnconfirmed, ev.BlockNumber, ev.HeadBlock, v.Confirmations)
		}
		return nil
	case ProviderMock:
		if v.MockSigner == (common.Address{}) {
			return fmt.Errorf("%w: no mock signer configured", ErrForged)
		}
		signer, err := RecoverSigner(raw.Metadata, raw.Report, raw.Evidence.Signature)
		if err != nil || signer != v.MockSigner {
			return fmt.Errorf("%w: signature does not recover to the mock signer", ErrForged)
		}
		return nil
	}
	return fmt.Errorf("%w: provider %q", ErrUnsupported, v.Provider)
}

// SigningHash is what signature evidence signs: keccak256(metadata || report).
func SigningHash(metadata, report []byte) []byte {
	return crypto.Keccak256(metadata, report)
}

// RecoverSigner recovers the address behind a 65-byte signature over SigningHash.
func RecoverSigner(metadata, report, sig []byte) (common.Address, error) {
	if len(sig) != 65 {
		return common.Address{}, fmt.Errorf("%w: signature is %d bytes", ErrForged, len(sig))
	}
	s := make([]byte, 65)
	copy(s, sig)
	if s[64] >= 27 {
		s[64] -= 27
	}
	pub, err := crypto.SigToPub(SigningHash(metadata, report), s)
	if err != nil {
		return common.Address{}, fmt.Errorf("%w: %v", ErrForged, err)
	}
	return crypto.PubkeyToAddress(*pub), nil
}

func (v *Verifier) rows(ctx context.Context, report Report, meta Metadata, raw RawAttestation, hash []byte, now time.Time) ([]Attestation, error) {
	base := Attestation{
		Kind: report.Kind, SubjectType: report.Kind.SubjectType(), PayloadHash: hash, Payload: raw.Report,
		Chain: v.Chain, TxHash: raw.Evidence.TxHash, BlockNumber: raw.Evidence.BlockNumber,
		WorkflowID: meta.WorkflowID, WorkflowOwner: meta.Owner, ReportID: meta.ReportID,
		ObservedAt: report.ObservedAt, RecordedAt: now, Provider: v.Provider, Simulated: raw.Simulated,
	}
	var out []Attestation
	add := func(key [32]byte, item any, check func(Subject) error) error {
		row := base
		row.ID = uuid.NewString()
		row.Item = item
		row.ItemIndex = len(out)
		subject, found, err := v.Subjects.LookupSubject(ctx, report.Kind, key)
		if err != nil {
			return err
		}
		switch {
		case !found:
			row.SubjectID = "0x" + common.Bytes2Hex(key[:])
			row.Status = StatusFailed
			row.Reason = ErrUnknownSubject.Error()
		default:
			row.SubjectID = subject.ID
			if err := check(subject); err != nil {
				row.Status = StatusFailed
				row.Reason = err.Error()
			} else {
				row.Status = StatusAttested
			}
		}
		out = append(out, row)
		return nil
	}
	switch items := report.Items.(type) {
	case []SolvencyItem:
		for _, it := range items {
			if err := add(it.CheckpointHash, it, func(Subject) error { return nil }); err != nil {
				return nil, err
			}
		}
	case []DepositItem:
		for _, it := range items {
			it := it
			if err := add(it.DepositID, it, func(s Subject) error { return checkDeposit(s, it) }); err != nil {
				return nil, err
			}
		}
	case []ConversionItem:
		for _, it := range items {
			if err := add(it.ConversionID, it, func(Subject) error { return nil }); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

// checkDeposit is check 5 for deposits: the attested (token, amount, destination) must equal what the gateway credited.
func checkDeposit(s Subject, it DepositItem) error {
	expected, err := DepositItemFromSubject(s)
	if err != nil {
		return err
	}
	if it.Verdict != VerdictConfirmed {
		return fmt.Errorf("%w: verdict %d", ErrSubjectMismatch, it.Verdict)
	}
	if it.Token != expected.Token || it.Destination != expected.Destination || it.Amount == nil || expected.Amount == nil || it.Amount.Cmp(expected.Amount) != 0 {
		return fmt.Errorf("%w: token, amount or destination differ", ErrSubjectMismatch)
	}
	return nil
}

// DepositSubject turns a pending deposit into the subject the verifier will match against.
func DepositSubject(d PendingDeposit, at time.Time) Subject {
	amount := "0"
	if d.ExpectedAmountMinor != nil {
		amount = d.ExpectedAmountMinor.String()
	}
	return Subject{
		Kind: KindDepositFinality, Key: SubjectKey(d.DepositID), ID: d.DepositID, AskedAt: at,
		Facts: map[string]any{"chain": d.Chain, "tx": d.Tx, "token": d.Token, "expected_amount_minor": amount, "destination": d.Destination},
	}
}

// DepositItemFromSubject builds the item a confirmed attestation of this subject must equal (verdict and slot aside).
func DepositItemFromSubject(s Subject) (DepositItem, error) {
	str := func(k string) string {
		v, _ := s.Facts[k].(string)
		return v
	}
	amount, ok := parseBig(str("expected_amount_minor"))
	if !ok {
		return DepositItem{}, fmt.Errorf("%w: subject %s has no amount", ErrInvalidReport, s.ID)
	}
	return DepositItem{
		DepositID: s.Key, ChainID: LabelKey(str("chain")), TxRef: SubjectKey(str("tx")), Token: LabelKey(str("token")),
		Amount: amount, Destination: SubjectKey(str("destination")),
	}, nil
}

// ConversionSubject and CheckpointSubject are the other two subject shapes.
func ConversionSubject(c Conversion, at time.Time) Subject {
	amount := "0"
	if c.AmountMinor != nil {
		amount = c.AmountMinor.String()
	}
	return Subject{
		Kind: KindConversionReference, Key: SubjectKey(c.ConversionID), ID: c.ConversionID, AskedAt: at,
		Facts: map[string]any{"executed_at": c.ExecutedAt.UTC().Format(time.RFC3339), "base": c.Base, "quote": c.Quote, "executed_rate_decimal": c.ExecutedRate, "amount_minor": amount},
	}
}

func CheckpointSubject(cp Checkpoint) Subject {
	assets := make([]map[string]any, 0, len(cp.Assets))
	for _, a := range cp.Assets {
		assets = append(assets, map[string]any{"asset": a.Asset, "liabilities_minor": a.Liabilities.String(), "decimals": a.Decimals})
	}
	return Subject{
		Kind: KindSolvency, Key: cp.Hash, ID: cp.ID, AskedAt: cp.TakenAt,
		Facts: map[string]any{"checkpoint_hash": "0x" + common.Bytes2Hex(cp.Hash[:]), "max_journal_id": cp.MaxJournalID, "assets": assets},
	}
}

// IsRejection says whether an error is a verification verdict rather than an infrastructure failure.
func IsRejection(err error) bool {
	for _, target := range []error{ErrInvalidReport, ErrForged, ErrReplayed, ErrWrongWorkflow, ErrWrongOwner, ErrWrongGateway, ErrWrongEmitter, ErrUnconfirmed, ErrStale, ErrDisabled, ErrUnsupported} {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}
