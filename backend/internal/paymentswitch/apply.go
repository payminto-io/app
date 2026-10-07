package paymentswitch

import (
	"fmt"

	"github.com/payminto/payminto/backend/internal/connectors"
	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
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
// hypotheses (C1). Evidence that contradicts a terminal state is recorded as an anomaly, not applied (M10).
// A strictly older state is ignored. Money is posted the first time it is in, or by delta for deposits.
func (s *Service) applyAttempt(tx *gorm.DB, intent *IntentRow, attempt *AttemptRow, u attemptUpdate) (applyResult, error) {
	if u.status == attempt.Status {
		return s.applySameStatus(tx, intent, attempt, u)
	}
	if attempt.Status.IsTerminal() {
		detail := fmt.Sprintf("connector reported %q (%s) after terminal %s: %s", u.raw, u.status, attempt.Status, u.reason)
		if err := s.recordAnomaly(tx, "attempt", attempt.ID, intent.ID, AnomalyEvidenceAfterTerminal, detail); err != nil {
			return applyResult{}, err
		}
		return applyResult{ignored: "evidence after terminal state recorded as anomaly"}, nil
	}
	if err := transitionAttempt(attempt.Status, u.status); err != nil {
		return applyResult{ignored: err.Error()}, nil
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
	if u.raw != "" {
		attempt.RawStatus = u.raw
	}
	if u.connectorTransactionID != "" {
		id := u.connectorTransactionID
		attempt.ConnectorTransactionID = &id
	}
	if u.amountReceived != nil {
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
	if u.amountReceived != nil && attempt.Status.MoneyIn() {
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

// settleAmount decides what is captured and posts the journal. Card path: the connector's amount, else the amount
// claimed at capture time, else refuse (I6). Deposit path: the received token amount, posted by delta in the
// chain-qualified asset against custody (I7, I9); the USD figure is the finalizer's own 1:1 for stablecoins.
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
		received := *u.amountReceived
		already := decimal.Zero
		if attempt.AmountReceived != nil {
			already = *attempt.AmountReceived
		}
		delta := received.Sub(already)
		if delta.IsPositive() {
			if err := s.postDeposit(tx, intent, attempt, delta, received, u.receivedAsset); err != nil {
				return decimal.Zero, "", err
			}
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

// postPayment debits the connector's asset account and credits the merchant's liability, once per attempt.
func (s *Service) postPayment(tx *gorm.DB, intent *IntentRow, attempt *AttemptRow, captured decimal.Decimal) error {
	j := ledger.Journal{
		Kind:           ledger.KindPayment,
		Reference:      ledger.Reference{Type: "payment_attempt", ID: attempt.ID},
		IdempotencyKey: "switch.payment." + attempt.ID,
		PostedAt:       s.now(),
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
		PostedAt:       s.now(),
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
		PostedAt:       s.now(),
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
