package cre

import (
	"context"
	"errors"
	"fmt"
	"math/big"
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

// Binding is the (id, owner, name) the consumer contract holds for one kind; every report must carry it.
type Binding struct {
	ID    [32]byte
	Owner [20]byte
	Name  [10]byte
}

// Verifier applies SPEC section 6 to a raw attestation before anything is stored. It mirrors the contract:
// replay is the report hash, there is no ordering between distinct reports, and finality is the reader's bound.
type Verifier struct {
	Provider  string
	GatewayID [32]byte
	Consumer  [20]byte
	Bindings  map[Kind]Binding
	// MaxClockSkew bounds how far ahead of the gateway clock a DON timestamp may be (the contract allows 5 min).
	MaxClockSkew time.Duration
	// MockSigner is the only accepted signer of signature evidence; set for the mock provider.
	MockSigner common.Address
	Subjects   SubjectIndex
	// Seen reports whether a payload hash is already recorded for this provider.
	Seen func(ctx context.Context, payloadHash []byte) (bool, error)
	Now  func() time.Time
	// Chain is stamped on the rows.
	Chain string
	// SimulatedForwarder is true when CRE_FORWARDER_ADDRESS is a mock or simulation forwarder: every record is simulated.
	SimulatedForwarder bool
	// Live refuses simulated identities outright instead of recording them as simulated.
	Live bool
}

// Verify returns one row per item. A whole-report failure is an error; a per-item finding is a row with
// StatusMismatch, StatusIgnored or StatusFailed and a Reason, so nothing unverified is ever shown as attested.
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
	want, ok := v.Bindings[report.Kind]
	if !ok || want.ID == ([32]byte{}) || meta.WorkflowID != want.ID {
		return nil, fmt.Errorf("%w: %x", ErrWrongWorkflow, meta.WorkflowID)
	}
	if meta.Owner != want.Owner {
		return nil, fmt.Errorf("%w: %s", ErrWrongOwner, common.BytesToAddress(meta.Owner[:]))
	}
	if meta.WorkflowName != want.Name {
		return nil, fmt.Errorf("%w: %x", ErrWrongName, meta.WorkflowName)
	}
	if report.GatewayID != v.GatewayID {
		return nil, fmt.Errorf("%w: %x", ErrWrongGateway, report.GatewayID)
	}
	simulated := raw.Simulated || v.SimulatedForwarder || IsSimulatorIdentity(meta)
	if simulated && v.Live {
		return nil, fmt.Errorf("%w: workflow %x owner %s", ErrSimulated, meta.WorkflowID, common.BytesToAddress(meta.Owner[:]))
	}
	raw.Simulated = simulated
	if err := v.checkEvidence(raw); err != nil {
		return nil, err
	}
	now := v.now()
	if report.ObservedAt.Sub(now) > v.skew() {
		return nil, fmt.Errorf("%w: observed_at %s, gateway clock %s", ErrClockAhead, report.ObservedAt.Format(time.RFC3339), now.Format(time.RFC3339))
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

// checkEvidence is checks 1, 2 and 6 for chainlink (own-RPC log, emitter, finality, calldata hash) and the
// dev-key signature for mock. A log that is not yet final is ErrUnconfirmed: retried, never refused.
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
			return fmt.Errorf("%w: calldata report does not hash to the contract's reportHash", ErrForged)
		}
		if !ev.Final {
			return fmt.Errorf("%w: block %d is above the finality bound %d", ErrUnconfirmed, ev.BlockNumber, ev.HeadBlock)
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
	onChain, err := v.solvencyOutcomes(report, raw)
	if err != nil {
		return nil, err
	}
	var out []Attestation
	// check returns the fact comparison for a found subject; attested means every served fact matches.
	add := func(key [32]byte, item any, check func(Subject) (Status, string)) error {
		row := base
		row.ID = uuid.NewString()
		row.Item = item
		row.ItemIndex = len(out)
		row.OnChain = OnChainEmitted
		if onChain != nil {
			row.OnChain = onChain[row.ItemIndex]
		}
		subject, found, err := v.Subjects.LookupSubject(ctx, report.Kind, key)
		if err != nil {
			return err
		}
		if found {
			row.SubjectID = subject.ID
			row.FactCheck, row.Reason = check(subject)
		} else {
			row.SubjectID = "0x" + common.Bytes2Hex(key[:])
			row.FactCheck, row.Reason = StatusFailed, ErrUnknownSubject.Error()
		}
		row.Status = row.FactCheck
		if row.OnChain == OnChainIgnored {
			row.Status = StatusIgnored
			reason := "superseded on chain: the contract kept another snapshot for this asset (SolvencyIgnored)"
			if row.FactCheck != StatusAttested {
				reason += "; fact check " + string(row.FactCheck) + ": " + row.Reason
			}
			row.Reason = reason
		}
		out = append(out, row)
		return nil
	}
	switch items := report.Items.(type) {
	case []SolvencyItem:
		for _, it := range items {
			it := it
			if err := add(it.CheckpointHash, it, func(s Subject) (Status, string) { return checkSolvency(s, it) }); err != nil {
				return nil, err
			}
		}
	case []DepositItem:
		for _, it := range items {
			it := it
			if err := add(it.DepositID, it, func(s Subject) (Status, string) { return checkDeposit(s, it) }); err != nil {
				return nil, err
			}
		}
	case []ConversionItem:
		for _, it := range items {
			it := it
			if err := add(it.ConversionID, it, func(s Subject) (Status, string) { return checkConversion(s, it) }); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

// solvencyOutcomes is the contract's verdict per solvency item, nil for other kinds. It comes from the contract's
// own events (raw.Outcomes), never from the gateway's fact check: the contract stores the first occurrence of an
// asset when it is newer than its stored snapshot and ignores later duplicates, whatever their figures.
func (v *Verifier) solvencyOutcomes(report Report, raw RawAttestation) ([]OnChain, error) {
	items, ok := report.Items.([]SolvencyItem)
	if !ok {
		return nil, nil
	}
	outcomes := raw.Outcomes
	if outcomes == nil && v.Provider == ProviderMock {
		// No contract behind a pushed mock report: apply the contract's in-batch rule.
		outcomes = SimulateSolvencyOutcomes(items, nil)
	}
	if len(outcomes) != len(items) {
		return nil, fmt.Errorf("%w: %d contract item events for %d report items", ErrForged, len(outcomes), len(items))
	}
	out := make([]OnChain, len(items))
	for i, it := range items {
		if outcomes[i].Key != it.Asset {
			return nil, fmt.Errorf("%w: contract item event %d names asset %s, report item is %s", ErrForged, i, LabelFromKey(outcomes[i].Key), LabelFromKey(it.Asset))
		}
		out[i] = OnChainIgnored
		if outcomes[i].Stored {
			out[i] = OnChainStored
		}
	}
	return out, nil
}

// SimulateSolvencyOutcomes applies GatewayAttestations._recordSolvency to one batch: an item is stored when its
// asset has no snapshot at or after this batch, which also makes every later duplicate in the batch ignored.
// stale reports the assets whose stored snapshot is already as new as this batch.
func SimulateSolvencyOutcomes(items []SolvencyItem, stale func(asset [32]byte) bool) []ItemOutcome {
	written := map[[32]byte]bool{}
	out := make([]ItemOutcome, len(items))
	for i, it := range items {
		stored := !written[it.Asset] && (stale == nil || !stale(it.Asset))
		written[it.Asset] = true
		out[i] = ItemOutcome{Key: it.Asset, Stored: stored}
	}
	return out
}

// checkSolvency: the attested liabilities and decimals must equal what the checkpoint served for that asset;
// reserves are the DON's own observation and are not compared.
func checkSolvency(s Subject, it SolvencyItem) (Status, string) {
	cp := checkpointFromSubject(s)
	for _, a := range cp.Assets {
		if LabelKey(a.Asset) != it.Asset {
			continue
		}
		if it.Liabilities == nil || a.Liabilities.Cmp(it.Liabilities) != 0 || a.Decimals != it.Decimals {
			return StatusMismatch, fmt.Sprintf("%s: attested liabilities %s with %d decimals, checkpoint has %s with %d", ErrSubjectMismatch, bigString(it.Liabilities), it.Decimals, a.Liabilities, a.Decimals)
		}
		return StatusAttested, ""
	}
	return StatusMismatch, fmt.Sprintf("%s: asset %s is not in checkpoint %s", ErrSubjectMismatch, LabelFromKey(it.Asset), s.ID)
}

// checkDeposit: the attested (token, amount, destination) must equal what the gateway credited.
func checkDeposit(s Subject, it DepositItem) (Status, string) {
	expected, err := DepositItemFromSubject(s)
	if err != nil {
		return StatusFailed, err.Error()
	}
	if it.Verdict != VerdictConfirmed {
		return StatusMismatch, fmt.Sprintf("%s: verdict %d", ErrSubjectMismatch, it.Verdict)
	}
	if it.Token != expected.Token || it.Destination != expected.Destination || it.Amount == nil || expected.Amount == nil || it.Amount.Cmp(expected.Amount) != 0 {
		return StatusMismatch, fmt.Sprintf("%s: token, amount or destination differ", ErrSubjectMismatch)
	}
	return StatusAttested, ""
}

// checkConversion: the attested pair must be the conversion's base/quote.
func checkConversion(s Subject, it ConversionItem) (Status, string) {
	base, _ := s.Facts["base"].(string)
	quote, _ := s.Facts["quote"].(string)
	if base == "" || quote == "" {
		return StatusFailed, fmt.Sprintf("%s: conversion %s has no pair", ErrInvalidReport, s.ID)
	}
	if LabelKey(base+"/"+quote) != it.Pair {
		return StatusMismatch, fmt.Sprintf("%s: attested pair %s, conversion is %s/%s", ErrSubjectMismatch, LabelFromKey(it.Pair), base, quote)
	}
	return StatusAttested, ""
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

// IsRejection says whether an error is a definitive verdict on the report (the cursor may pass it).
func IsRejection(err error) bool {
	for _, target := range []error{ErrInvalidReport, ErrForged, ErrReplayed, ErrWrongWorkflow, ErrWrongOwner, ErrWrongName, ErrWrongGateway, ErrWrongEmitter, ErrSimulated, ErrDisabled, ErrUnsupported} {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

// IsConfigurationMismatch says whether a refusal is the gateway's own configuration or clock disagreeing with a
// report the contract already accepted under the owner's binding; such a refusal is retryable, not terminal.
func IsConfigurationMismatch(err error) bool {
	for _, target := range []error{ErrWrongWorkflow, ErrWrongOwner, ErrWrongName, ErrClockAhead} {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

// IsRetryable says whether time or infrastructure may cure the error; the cursor must not pass such a block.
func IsRetryable(err error) bool {
	return err != nil && !IsRejection(err)
}

func bigString(v *big.Int) string {
	if v == nil {
		return "nil"
	}
	return v.String()
}
