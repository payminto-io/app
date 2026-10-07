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
	nextAction             *connectors.NextAction
	errorCode              string
	errorMessage           string
	reason                 string
}

// applyAttempt moves the attempt and its intent together inside tx, and posts the payment journal the first
// time money is in. Same status twice is a no-op; a backwards edge is ErrInvalidTransition.
func (s *Service) applyAttempt(tx *gorm.DB, intent *IntentRow, attempt *AttemptRow, u attemptUpdate) error {
	if err := transitionAttempt(attempt.Status, u.status); err != nil {
		return err
	}
	derived := IntentStatusFor(u.status)
	if err := transitionIntent(intent.Status, derived); err != nil {
		return err
	}
	fromAttempt, fromIntent := attempt.Status, intent.Status

	attempt.Status = u.status
	if u.raw != "" {
		attempt.RawStatus = u.raw
	}
	if u.connectorTransactionID != "" {
		id := u.connectorTransactionID
		attempt.ConnectorTransactionID = &id
	}
	if u.amountReceived != nil {
		attempt.AmountReceived = *u.amountReceived
	}
	attempt.ErrorCode, attempt.ErrorMessage = u.errorCode, u.errorMessage
	attempt.NextAction = nextActionJSON(u.nextAction)

	moneyIn := u.status.MoneyIn() && !fromAttempt.MoneyIn()
	if moneyIn {
		captured := attempt.Amount
		switch {
		case u.amountCaptured != nil:
			captured = *u.amountCaptured
		case u.amountReceived != nil:
			captured = *u.amountReceived
		}
		if !captured.IsPositive() {
			return fmt.Errorf("%w: connector reported %s as captured for attempt %s", ErrInvalid, captured, attempt.ID)
		}
		attempt.AmountCaptured = captured
		intent.AmountCaptured = captured
		if err := s.postPayment(tx, intent, attempt); err != nil {
			return err
		}
	}

	intent.Status = derived
	intent.ActiveAttemptID = attempt.ID
	intent.ConnectorCode = attempt.ConnectorCode
	intent.NextAction = attempt.NextAction
	intent.LastErrorCode, intent.LastErrorMessage = u.errorCode, u.errorMessage

	if err := s.saveAttempt(tx, attempt); err != nil {
		return err
	}
	if err := s.saveIntent(tx, intent); err != nil {
		return err
	}
	if err := s.recordTransition(tx, "attempt", attempt.ID, string(fromAttempt), string(attempt.Status), u.reason); err != nil {
		return err
	}
	return s.recordTransition(tx, "intent", intent.ID, string(fromIntent), string(intent.Status), u.reason)
}

// postPayment debits the connector's asset account and credits the merchant's liability, once per attempt.
func (s *Service) postPayment(tx *gorm.DB, intent *IntentRow, attempt *AttemptRow) error {
	j := ledger.Journal{
		Kind:           ledger.KindPayment,
		Reference:      ledger.Reference{Type: "payment_attempt", ID: attempt.ID},
		IdempotencyKey: "switch.payment." + attempt.ID,
		PostedAt:       s.now(),
		Metadata:       map[string]any{"intent_id": intent.ID, "connector": string(attempt.ConnectorCode)},
		Lines: []ledger.Line{
			{Account: ledger.AccountKey{OwnerType: ledger.OwnerConnector, OwnerID: string(attempt.ConnectorCode), Asset: attempt.Asset, Kind: ledger.KindAsset}, Amount: attempt.AmountCaptured},
			{Account: ledger.AccountKey{OwnerType: ledger.OwnerMember, OwnerID: intent.MerchantID, Asset: attempt.Asset, Kind: ledger.KindLiability}, Amount: attempt.AmountCaptured.Neg()},
		},
	}
	if _, err := s.ledger.PostIn(tx.Statement.Context, tx, j); err != nil {
		return fmt.Errorf("paymentswitch: post payment journal: %w", err)
	}
	return nil
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
func (s *Service) applyRefund(tx *gorm.DB, intent *IntentRow, refund *RefundRow, u refundUpdate) error {
	if err := transitionRefund(refund.Status, u.status); err != nil {
		return err
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

	if u.status == RefundSucceeded && from != RefundSucceeded {
		if err := s.postRefund(tx, intent, refund); err != nil {
			return err
		}
		intent.AmountRefunded = intent.AmountRefunded.Add(refund.Amount)
		if err := s.saveIntent(tx, intent); err != nil {
			return err
		}
	}
	if err := s.saveRefund(tx, refund); err != nil {
		return err
	}
	return s.recordTransition(tx, "refund", refund.ID, string(from), string(refund.Status), u.reason)
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
