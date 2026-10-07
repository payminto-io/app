package paymentswitch

import (
	"errors"
	"fmt"

	"github.com/payminto/payminto/backend/internal/connectors"
	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// updateSource says who brings a fact; only the reconciler may apply a rollback edge, and only after the lease (R1).
type updateSource string

const (
	sourceCall       updateSource = "call"
	sourceWebhook    updateSource = "webhook"
	sourceManualSync updateSource = "sync"
	sourceReconciler updateSource = "reconciler"
)

// attemptUpdate is one connector-reported fact about an attempt, already mapped into our vocabulary.
type attemptUpdate struct {
	status                 AttemptStatus
	raw                    connectors.RawStatus
	connectorTransactionID string
	amountCaptured         *decimal.Decimal
	amountReceived         *decimal.Decimal
	receivedAsset          string
	nextAction             *connectors.NextAction
	errorCode              string
	errorMessage           string
	reason                 string
	source                 updateSource
}

// applyResult says what applyAttempt did so callers can answer honestly.
type applyResult struct {
	changed bool
	ignored string // non-empty when the update was recorded but not applied
	events  []Event
}

// keepsNextAction lists the statuses where the customer may still act, so Sync and webhooks keep the instruction (I5).
func keepsNextAction(s AttemptStatus) bool {
	return s == AttemptPending || s == AttemptAuthenticationPending || s == AttemptPartiallyPaid
}

// applyAttempt moves the attempt and its intent together inside tx. Connector evidence wins over our own
// hypotheses (C1), except a rollback edge while the claim lease is live (R1). Evidence that contradicts a
// terminal state is recorded as an anomaly, not applied (M10); money that arrives after a terminal state is
// booked to unallocated receipts with a late_receipt anomaly (R3). A strictly older state is ignored.
func (s *Service) applyAttempt(tx *gorm.DB, intent *IntentRow, attempt *AttemptRow, u attemptUpdate) (applyResult, error) {
	if u.status == attempt.Status {
		return s.applySameStatus(tx, intent, attempt, u)
	}
	if attempt.Status.IsTerminal() {
		late, err := s.lateReceipt(tx, intent, attempt, u)
		if err != nil {
			return applyResult{}, err
		}
		detail := fmt.Sprintf("connector reported %q (%s) after terminal %s: %s", u.raw, u.status, attempt.Status, u.reason)
		if err := s.recordAnomaly(tx, "attempt", attempt.ID, intent.ID, AnomalyEvidenceAfterTerminal, detail); err != nil {
			return applyResult{}, err
		}
		return applyResult{changed: late, ignored: "evidence after terminal state recorded as anomaly"}, nil
	}
	if err := transitionAttempt(attempt.Status, u.status); err != nil {
		return applyResult{ignored: err.Error()}, nil
	}
	if IsRollback(attempt.Status, u.status) {
		if u.source != sourceReconciler {
			return applyResult{ignored: "rollback evidence is applied only by the reconciler"}, nil
		}
		if attempt.ClaimedUntil != nil && s.now().Before(*attempt.ClaimedUntil) {
			return applyResult{ignored: "claim lease is live; the operation may still be in flight"}, nil
		}
	}
	derived := IntentStatusFor(u.status)
	if err := transitionIntent(intent.Status, derived); err != nil {
		return applyResult{}, err
	}
	fromAttempt, fromIntent := attempt.Status, intent.Status

	moneyIn := u.status.MoneyIn()
	captured, asset, err := s.settleAmount(tx, intent, attempt, u, moneyIn)
	if err != nil {
		return applyResult{}, err
	}

	attempt.Status = u.status
	s.touchStatus(attempt, fromAttempt)
	attempt.ClaimedUntil = nil
	if u.raw != "" {
		attempt.RawStatus = u.raw
	}
	if u.connectorTransactionID != "" {
		id := u.connectorTransactionID
		attempt.ConnectorTransactionID = &id
	}
	if u.amountReceived != nil && moneyIn {
		received := *u.amountReceived
		attempt.AmountReceived = &received
		attempt.ReceivedAsset = u.receivedAsset
	}
	attempt.ErrorCode, attempt.ErrorMessage = u.errorCode, u.errorMessage
	if u.nextAction != nil || !keepsNextAction(u.status) {
		attempt.NextAction = nextActionJSON(u.nextAction)
	}
	if moneyIn {
		attempt.AmountCaptured = captured
		intent.AmountCaptured = captured
	}
	if u.status == AttemptUnderpaid {
		intent.Metadata = withString(intent.Metadata, "shortfall", attempt.Amount.Sub(captured).String())
		intent.Metadata = withString(intent.Metadata, "received_asset", asset)
	}
	if moneyIn && (u.status == AttemptCharged || u.status == AttemptPartialCharged || u.status == AttemptOverpaid) {
		if err := s.postFee(tx, intent, attempt, captured); err != nil {
			return applyResult{}, err
		}
	}
	// Money that arrived with a terminal non-money status (a fill that beat a void) is late money.
	if u.amountReceived != nil && !moneyIn && u.status.IsTerminal() {
		if _, err := s.lateReceipt(tx, intent, attempt, u); err != nil {
			return applyResult{}, err
		}
	}

	intent.Status = derived
	intent.ActiveAttemptID = attempt.ID
	intent.ConnectorCode = attempt.ConnectorCode
	intent.NextAction = attempt.NextAction
	intent.LastErrorCode, intent.LastErrorMessage = u.errorCode, u.errorMessage

	if err := s.saveAttempt(tx, attempt); err != nil {
		return applyResult{}, err
	}
	if err := s.saveIntent(tx, intent); err != nil {
		return applyResult{}, err
	}
	if err := s.recordTransition(tx, "attempt", attempt.ID, string(fromAttempt), string(attempt.Status), u.reason); err != nil {
		return applyResult{}, err
	}
	if err := s.recordTransition(tx, "intent", intent.ID, string(fromIntent), string(intent.Status), u.reason); err != nil {
		return applyResult{}, err
	}
	return applyResult{changed: true, events: intentEvents(fromIntent, intent, attempt)}, nil
}

// applySameStatus records fresh facts (connector id, received amount growth, next action) without a transition.
func (s *Service) applySameStatus(tx *gorm.DB, intent *IntentRow, attempt *AttemptRow, u attemptUpdate) (applyResult, error) {
	changed := false
	if u.connectorTransactionID != "" && (attempt.ConnectorTransactionID == nil || *attempt.ConnectorTransactionID != u.connectorTransactionID) {
		id := u.connectorTransactionID
		attempt.ConnectorTransactionID = &id
		changed = true
	}
	if u.nextAction != nil {
		attempt.NextAction = nextActionJSON(u.nextAction)
		intent.NextAction = attempt.NextAction
		changed = true
	}
	if u.amountReceived != nil {
		switch {
		case attempt.Status.IsTerminal():
			late, err := s.lateReceipt(tx, intent, attempt, u)
			if err != nil {
				return applyResult{}, err
			}
			changed = changed || late
		case attempt.Status.MoneyIn():
			captured, _, err := s.settleAmount(tx, intent, attempt, u, true)
			if err != nil {
				return applyResult{}, err
			}
			if !captured.Equal(attempt.AmountCaptured) {
				received := *u.amountReceived
				attempt.AmountReceived = &received
				attempt.ReceivedAsset = u.receivedAsset
				attempt.AmountCaptured = captured
				intent.AmountCaptured = captured
				changed = true
			}
		}
	}
	if !changed {
		return applyResult{}, nil
	}
	if err := s.saveAttempt(tx, attempt); err != nil {
		return applyResult{}, err
	}
	if err := s.saveIntent(tx, intent); err != nil {
		return applyResult{}, err
	}
	return applyResult{changed: true}, nil
}

// watermark checks a reported cumulative against what is already booked: growth returns the delta, equality
// nothing, a decrease is an anomaly and nothing changes (R4).
func (s *Service) watermark(tx *gorm.DB, intent *IntentRow, attempt *AttemptRow, received decimal.Decimal) (decimal.Decimal, bool, error) {
	already := decimal.Zero
	if attempt.AmountReceived != nil {
		already = *attempt.AmountReceived
	}
	delta := received.Sub(already)
	if delta.IsNegative() {
		detail := fmt.Sprintf("connector reported %s received, below the booked %s; the watermark is kept", received, already)
		if err := s.recordAnomaly(tx, "attempt", attempt.ID, intent.ID, AnomalyReceivedDecreased, detail); err != nil {
			return decimal.Zero, false, err
		}
		return decimal.Zero, false, nil
	}
	return delta, delta.IsPositive(), nil
}

// lateReceipt books money that arrived after the attempt closed: custody up, a platform unallocated_receipts
// liability up, and a late_receipt anomaly for the operator (refund or apply is ticket 11).
func (s *Service) lateReceipt(tx *gorm.DB, intent *IntentRow, attempt *AttemptRow, u attemptUpdate) (bool, error) {
	if u.amountReceived == nil {
		return false, nil
	}
	if u.receivedAsset == "" {
		return false, s.recordAnomaly(tx, "attempt", attempt.ID, intent.ID, AnomalyAmountUnknown, "connector reported late money without naming the asset")
	}
	delta, grew, err := s.watermark(tx, intent, attempt, *u.amountReceived)
	if err != nil || !grew {
		return false, err
	}
	received := *u.amountReceived
	j := ledger.Journal{
		Kind:           ledger.KindPayment,
		Reference:      ledger.Reference{Type: "payment_attempt", ID: attempt.ID},
		IdempotencyKey: "switch.late." + attempt.ID + "." + received.String(),
		Metadata: map[string]any{
			"intent_id": intent.ID, "connector": string(attempt.ConnectorCode), "received_asset": u.receivedAsset,
			"cumulative_received": received.String(), "attempt_status": string(attempt.Status), "late_receipt": true,
		},
		Lines: []ledger.Line{
			{Account: ledger.AccountKey{OwnerType: ledger.OwnerPlatform, OwnerID: "crypto_assets", Asset: u.receivedAsset, Kind: ledger.KindAsset}, Amount: delta},
			{Account: ledger.AccountKey{OwnerType: ledger.OwnerPlatform, OwnerID: UnallocatedReceiptsOwner, Asset: u.receivedAsset, Kind: ledger.KindLiability}, Amount: delta.Neg()},
		},
	}
	if _, err := s.ledger.PostIn(tx.Statement.Context, tx, j); err != nil {
		return false, fmt.Errorf("paymentswitch: post late receipt: %w", err)
	}
	attempt.AmountReceived = &received
	attempt.ReceivedAsset = u.receivedAsset
	detail := fmt.Sprintf("%s %s arrived after the attempt was %s; booked to %s, total received %s", delta, u.receivedAsset, attempt.Status, UnallocatedReceiptsOwner, received)
	if err := s.recordAnomaly(tx, "attempt", attempt.ID, intent.ID, AnomalyLateReceipt, detail); err != nil {
		return false, err
	}
	if err := s.saveAttempt(tx, attempt); err != nil {
		return false, err
	}
	return true, nil
}

// settleAmount decides what is captured and posts the journal. Card path: the connector's amount, else the amount
// claimed at capture time, else refuse (I6). Deposit path: the received token amount, posted by delta against the
// watermark in the chain-qualified asset (I7, I9, R4); the USD figure is the finalizer's own 1:1 for stablecoins.
func (s *Service) settleAmount(tx *gorm.DB, intent *IntentRow, attempt *AttemptRow, u attemptUpdate, moneyIn bool) (decimal.Decimal, string, error) {
	if !moneyIn {
		return attempt.AmountCaptured, "", nil
	}
	if u.amountReceived != nil {
		if u.receivedAsset == "" {
			if err := s.recordAnomaly(tx, "attempt", attempt.ID, intent.ID, AnomalyAmountUnknown, "connector reported a received amount without naming the asset"); err != nil {
				return decimal.Zero, "", err
			}
			return decimal.Zero, "", fmt.Errorf("%w: received asset missing", ErrAmountUnknown)
		}
		delta, grew, err := s.watermark(tx, intent, attempt, *u.amountReceived)
		if err != nil {
			return decimal.Zero, "", err
		}
		if !grew {
			return attempt.AmountCaptured, attempt.ReceivedAsset, nil
		}
		received := *u.amountReceived
		if err := s.postDeposit(tx, intent, attempt, delta, received, u.receivedAsset); err != nil {
			return decimal.Zero, "", err
		}
		return received, u.receivedAsset, nil
	}
	if attempt.AmountCaptured.IsPositive() {
		if attempt.ReceivedAsset != "" {
			return attempt.AmountCaptured, attempt.ReceivedAsset, nil
		}
		return attempt.AmountCaptured, attempt.Asset, nil
	}
	captured := decimal.Zero
	switch {
	case u.amountCaptured != nil && u.amountCaptured.IsPositive():
		captured = *u.amountCaptured
	case attempt.AmountToCapture.IsPositive():
		captured = attempt.AmountToCapture
	default:
		if err := s.recordAnomaly(tx, "attempt", attempt.ID, intent.ID, AnomalyAmountUnknown, "connector reported money in without an amount and no capture amount was claimed"); err != nil {
			return decimal.Zero, "", err
		}
		return decimal.Zero, "", ErrAmountUnknown
	}
	if captured.GreaterThan(attempt.Amount) {
		return decimal.Zero, "", fmt.Errorf("%w: connector reported %s captured on a %s authorization", ErrInvalid, captured, attempt.Amount)
	}
	if err := s.postPayment(tx, intent, attempt, captured); err != nil {
		return decimal.Zero, "", err
	}
	return captured, attempt.Asset, nil
}

// postFee books the fee in the same transaction as the money (fees README "Payment path"). A refusal because
// another attempt of the payment already carries the fee is an anomaly to reconcile, not a failed payment.
func (s *Service) postFee(tx *gorm.DB, intent *IntentRow, attempt *AttemptRow, captured decimal.Decimal) error {
	if s.fees == nil {
		return nil
	}
	err := s.fees.PostFee(tx.Statement.Context, tx, FeeRef{PaymentRecordID: intent.PaymentRecordID, AttemptID: attempt.ID}, captured)
	if errors.Is(err, ErrFeeAlreadyPosted) {
		return s.recordAnomaly(tx, "attempt", attempt.ID, intent.ID, AnomalyFeeAlreadyPosted, err.Error())
	}
	if err != nil {
		return fmt.Errorf("paymentswitch: post fee: %w", err)
	}
	return nil
}

// Switch journals never set PostedAt: the ledger's request hash includes it when set, and a replay of one of these
// keys must be a replay, not a conflict.

// postPayment debits the connector's asset account and credits the merchant's liability, once per attempt.
func (s *Service) postPayment(tx *gorm.DB, intent *IntentRow, attempt *AttemptRow, captured decimal.Decimal) error {
	j := ledger.Journal{
		Kind:           ledger.KindPayment,
		Reference:      ledger.Reference{Type: "payment_attempt", ID: attempt.ID},
		IdempotencyKey: "switch.payment." + attempt.ID,
		Metadata:       map[string]any{"intent_id": intent.ID, "connector": string(attempt.ConnectorCode)},
		Lines: []ledger.Line{
			{Account: ledger.AccountKey{OwnerType: ledger.OwnerConnector, OwnerID: string(attempt.ConnectorCode), Asset: attempt.Asset, Kind: ledger.KindAsset}, Amount: captured},
			{Account: ledger.AccountKey{OwnerType: ledger.OwnerMember, OwnerID: intent.MerchantID, Asset: attempt.Asset, Kind: ledger.KindLiability}, Amount: captured.Neg()},
		},
	}
	if _, err := s.ledger.PostIn(tx.Statement.Context, tx, j); err != nil {
		return fmt.Errorf("paymentswitch: post payment journal: %w", err)
	}
	return nil
}

// postDeposit books a confirmed deposit delta into custody in the token's chain-qualified asset; keyed by the
// cumulative received amount so a repeated report of the same total is a replay, not a second posting.
func (s *Service) postDeposit(tx *gorm.DB, intent *IntentRow, attempt *AttemptRow, delta, cumulative decimal.Decimal, asset string) error {
	j := ledger.Journal{
		Kind:           ledger.KindPayment,
		Reference:      ledger.Reference{Type: "payment_attempt", ID: attempt.ID},
		IdempotencyKey: "switch.payment." + attempt.ID + "." + cumulative.String(),
		Metadata: map[string]any{
			"intent_id": intent.ID, "connector": string(attempt.ConnectorCode),
			"priced_asset": attempt.Asset, "priced_amount": attempt.Amount.String(), "received_asset": asset, "cumulative_received": cumulative.String(),
		},
		Lines: []ledger.Line{
			{Account: ledger.AccountKey{OwnerType: ledger.OwnerPlatform, OwnerID: "crypto_assets", Asset: asset, Kind: ledger.KindAsset}, Amount: delta},
			{Account: ledger.AccountKey{OwnerType: ledger.OwnerMember, OwnerID: intent.MerchantID, Asset: asset, Kind: ledger.KindLiability}, Amount: delta.Neg()},
		},
	}
	if _, err := s.ledger.PostIn(tx.Statement.Context, tx, j); err != nil {
		return fmt.Errorf("paymentswitch: post deposit journal: %w", err)
	}
	return nil
}

func intentEvents(from IntentStatus, intent *IntentRow, attempt *AttemptRow) []Event {
	if from == intent.Status {
		return nil
	}
	base := Event{MerchantID: intent.MerchantID, IntentID: intent.ID, AttemptID: attempt.ID, Payload: map[string]any{
		"status": string(intent.Status), "amount": intent.Amount.String(), "asset": intent.Asset, "amount_captured": intent.AmountCaptured.String(), "connector": string(attempt.ConnectorCode),
	}}
	switch intent.Status {
	case IntentSucceeded:
		base.Type = EventPaymentSucceeded
	case IntentFailed:
		base.Type = EventPaymentFailed
		base.Payload["error_code"] = intent.LastErrorCode
	default:
		return nil
	}
	return []Event{base}
}

type refundUpdate struct {
	status            RefundStatus
	raw               connectors.RawStatus
	connectorRefundID string
	errorCode         string
	errorMessage      string
	reason            string
}

// applyRefund moves a refund and, on success, posts the reversing journal once and adds to the intent's refunded total.
func (s *Service) applyRefund(tx *gorm.DB, intent *IntentRow, refund *RefundRow, u refundUpdate) (applyResult, error) {
	if u.status == refund.Status {
		if u.connectorRefundID != "" && refund.ConnectorRefundID == nil {
			id := u.connectorRefundID
			refund.ConnectorRefundID = &id
			if err := s.saveRefund(tx, refund); err != nil {
				return applyResult{}, err
			}
			return applyResult{changed: true}, nil
		}
		return applyResult{}, nil
	}
	if refund.Status.IsTerminal() {
		detail := fmt.Sprintf("connector reported %q (%s) after terminal %s: %s", u.raw, u.status, refund.Status, u.reason)
		if err := s.recordAnomaly(tx, "refund", refund.ID, intent.ID, AnomalyEvidenceAfterTerminal, detail); err != nil {
			return applyResult{}, err
		}
		return applyResult{ignored: "evidence after terminal state recorded as anomaly"}, nil
	}
	if err := transitionRefund(refund.Status, u.status); err != nil {
		return applyResult{ignored: err.Error()}, nil
	}
	from := refund.Status
	refund.Status = u.status
	refund.StatusChangedAt = s.now()
	refund.ClaimedUntil = nil
	if u.raw != "" {
		refund.RawStatus = u.raw
	}
	if u.connectorRefundID != "" {
		id := u.connectorRefundID
		refund.ConnectorRefundID = &id
	}
	refund.ErrorCode, refund.ErrorMessage = u.errorCode, u.errorMessage

	var events []Event
	if u.status == RefundSucceeded {
		if err := s.postRefund(tx, intent, refund); err != nil {
			return applyResult{}, err
		}
		intent.AmountRefunded = intent.AmountRefunded.Add(refund.Amount)
		if err := s.saveIntent(tx, intent); err != nil {
			return applyResult{}, err
		}
		events = append(events, Event{Type: EventRefundSucceeded, MerchantID: intent.MerchantID, IntentID: intent.ID, AttemptID: refund.AttemptID, RefundID: refund.ID, Payload: map[string]any{
			"amount": refund.Amount.String(), "asset": refund.Asset, "amount_refunded": intent.AmountRefunded.String(), "connector": string(refund.ConnectorCode),
		}})
	}
	if err := s.saveRefund(tx, refund); err != nil {
		return applyResult{}, err
	}
	if err := s.recordTransition(tx, "refund", refund.ID, string(from), string(refund.Status), u.reason); err != nil {
		return applyResult{}, err
	}
	return applyResult{changed: true, events: events}, nil
}

func (s *Service) postRefund(tx *gorm.DB, intent *IntentRow, refund *RefundRow) error {
	j := ledger.Journal{
		Kind:           ledger.KindRefund,
		Reference:      ledger.Reference{Type: "refund", ID: refund.ID},
		IdempotencyKey: "switch.refund." + refund.ID,
		Metadata:       map[string]any{"intent_id": intent.ID, "attempt_id": refund.AttemptID, "connector": string(refund.ConnectorCode)},
		Lines: []ledger.Line{
			{Account: ledger.AccountKey{OwnerType: ledger.OwnerMember, OwnerID: intent.MerchantID, Asset: refund.Asset, Kind: ledger.KindLiability}, Amount: refund.Amount},
			{Account: ledger.AccountKey{OwnerType: ledger.OwnerConnector, OwnerID: string(refund.ConnectorCode), Asset: refund.Asset, Kind: ledger.KindAsset}, Amount: refund.Amount.Neg()},
		},
	}
	if _, err := s.ledger.PostIn(tx.Statement.Context, tx, j); err != nil {
		return fmt.Errorf("paymentswitch: post refund journal: %w", err)
	}
	return nil
}

func withString(m JSONMap, key, value string) JSONMap {
	if m == nil {
		m = JSONMap{}
	}
	m[key] = value
	return m
}
