package paymentswitch

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/payminto/payminto/backend/internal/connectors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Sync asks the connector for the active attempt's current state; it is how a timeout resolves.
func (s *Service) Sync(ctx context.Context, merchantID, intentID string) (Intent, error) {
	db := s.db.WithContext(ctx)
	intent, err := loadIntent(db, merchantID, intentID)
	if err != nil {
		return Intent{}, err
	}
	if intent.ActiveAttemptID == "" {
		return intent.toIntent(), nil
	}
	attempt, err := loadAttempt(db, intent.ActiveAttemptID)
	if err != nil {
		return Intent{}, err
	}
	if attempt.Status.IsTerminal() {
		return intent.toIntent(), nil
	}
	conn, err := s.connector(attempt.ConnectorCode)
	if err != nil {
		return Intent{}, err
	}
	if !conn.Capabilities().Sync {
		return intent.toIntent(), nil
	}
	req := connectors.SyncRequest{AttemptID: attempt.ID}
	if attempt.ConnectorTransactionID != nil {
		req.ConnectorTransactionID = *attempt.ConnectorTransactionID
	}
	resp, err := conn.Sync(ctx, req)
	if errors.Is(err, connectors.ErrNotFound) && attempt.ConnectorTransactionID == nil {
		// The connector never saw the authorization: the timed-out call did not land.
		return s.applyToActiveAttempt(ctx, merchantID, intentID, attempt.ID, attemptUpdate{status: AttemptFailure, errorCode: "not_found_at_connector", errorMessage: "authorization never reached the connector", reason: "sync"})
	}
	if err != nil {
		return Intent{}, fmt.Errorf("paymentswitch: sync at %s: %w", attempt.ConnectorCode, err)
	}
	status, err := MapAttemptStatus(attempt.ConnectorCode, resp.RawStatus)
	if err != nil {
		return Intent{}, err
	}
	update := attemptUpdate{
		status: status, raw: resp.RawStatus, connectorTransactionID: resp.ConnectorTransactionID,
		amountCaptured: resp.AmountCaptured, amountReceived: resp.AmountReceived, nextAction: resp.NextAction,
		errorCode: resp.ErrorCode, errorMessage: resp.ErrorMessage, reason: "sync",
	}
	out, err := s.applyToActiveAttempt(ctx, merchantID, intentID, attempt.ID, update)
	if errors.Is(err, ErrInvalidTransition) {
		// The connector reports a state we are already past; keep ours, it is the later fact.
		return intent.toIntent(), nil
	}
	return out, err
}

// WebhookResult says what a delivery did; Ignored means verified and recorded but out of order.
type WebhookResult struct {
	EventID   string
	Kind      connectors.WebhookKind
	IntentID  string
	Ignored   bool
	IgnoreWhy string
}

// HandleWebhook verifies a delivery with the connector, records its event id (a repeat is ErrWebhookReplay)
// and applies it to the attempt or refund it names, all in one transaction.
func (s *Service) HandleWebhook(ctx context.Context, code connectors.Code, headers http.Header, body []byte) (WebhookResult, error) {
	conn, err := s.connector(code)
	if err != nil {
		return WebhookResult{}, err
	}
	if !conn.Capabilities().Webhooks {
		return WebhookResult{}, fmt.Errorf("%w: connector %s has no webhooks", connectors.ErrUnsupported, code)
	}
	ev, err := conn.VerifyWebhook(ctx, headers, body)
	if err != nil {
		return WebhookResult{}, err
	}
	if ev.EventID == "" {
		return WebhookResult{}, fmt.Errorf("%w: connector %s returned an empty event id", connectors.ErrWebhookMalformed, code)
	}
	result := WebhookResult{EventID: ev.EventID, Kind: ev.Kind}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "connector_code"}, {Name: "event_id"}}, DoNothing: true}).
			Create(&WebhookEventRow{ConnectorCode: code, EventID: ev.EventID, ReceivedAt: s.now()})
		if res.Error != nil {
			return fmt.Errorf("paymentswitch: record webhook: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("%w: %s event %s", ErrWebhookReplay, code, ev.EventID)
		}
		switch ev.Kind {
		case connectors.WebhookRefund:
			return s.applyRefundWebhook(tx, code, ev, &result)
		default:
			return s.applyPaymentWebhook(tx, code, ev, &result)
		}
	})
	if err != nil {
		return WebhookResult{}, err
	}
	return result, nil
}

func (s *Service) applyPaymentWebhook(tx *gorm.DB, code connectors.Code, ev connectors.WebhookEvent, result *WebhookResult) error {
	var attempt AttemptRow
	err := lockIfPostgres(tx).Where("connector_code = ? AND connector_transaction_id = ?", code, ev.ConnectorTransactionID).First(&attempt).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("%w: no attempt for %s transaction %q", ErrNotFound, code, ev.ConnectorTransactionID)
	}
	if err != nil {
		return fmt.Errorf("paymentswitch: load attempt by connector transaction: %w", err)
	}
	intent, err := loadIntent(lockIfPostgres(tx), attempt.MerchantID, attempt.IntentID)
	if err != nil {
		return err
	}
	result.IntentID = intent.ID
	status, err := MapAttemptStatus(code, ev.RawStatus)
	if err != nil {
		return err
	}
	update := attemptUpdate{status: status, raw: ev.RawStatus, amountCaptured: ev.AmountCaptured, amountReceived: ev.AmountReceived, reason: "webhook " + ev.EventID}
	if intent.ActiveAttemptID != attempt.ID {
		result.Ignored, result.IgnoreWhy = true, "attempt is no longer active"
		return nil
	}
	if err := s.applyAttempt(tx, &intent, &attempt, update); err != nil {
		if errors.Is(err, ErrInvalidTransition) {
			result.Ignored, result.IgnoreWhy = true, err.Error()
			return nil
		}
		return err
	}
	return nil
}

func (s *Service) applyRefundWebhook(tx *gorm.DB, code connectors.Code, ev connectors.WebhookEvent, result *WebhookResult) error {
	var refund RefundRow
	err := lockIfPostgres(tx).Where("connector_code = ? AND connector_refund_id = ?", code, ev.ConnectorRefundID).First(&refund).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("%w: no refund for %s refund %q", ErrNotFound, code, ev.ConnectorRefundID)
	}
	if err != nil {
		return fmt.Errorf("paymentswitch: load refund by connector id: %w", err)
	}
	intent, err := loadIntent(lockIfPostgres(tx), refund.MerchantID, refund.IntentID)
	if err != nil {
		return err
	}
	result.IntentID = intent.ID
	status, err := MapRefundStatus(code, ev.RawStatus)
	if err != nil {
		return err
	}
	if err := s.applyRefund(tx, &intent, &refund, refundUpdate{status: status, raw: ev.RawStatus, reason: "webhook " + ev.EventID}); err != nil {
		if errors.Is(err, ErrInvalidTransition) {
			result.Ignored, result.IgnoreWhy = true, err.Error()
			return nil
		}
		return err
	}
	return nil
}
