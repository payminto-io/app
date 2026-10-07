package paymentswitch

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/payminto/payminto/backend/internal/connectors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const defaultSyncBase = 30 * time.Second

func (s *Service) syncBase() time.Duration { return defaultSyncBase }

// Sync asks the connector for the active attempt's current state and for every refund still open on the intent.
// It is how an unknown outcome resolves. Evidence that cannot be applied is recorded, never swallowed silently.
func (s *Service) Sync(ctx context.Context, merchantID, intentID string) (Intent, error) {
	db := s.db.WithContext(ctx)
	intent, err := loadIntent(db, merchantID, intentID)
	if err != nil {
		return Intent{}, err
	}
	if intent.ActiveAttemptID != "" {
		attempt, err := loadAttempt(db, intent.ActiveAttemptID)
		if err != nil {
			return Intent{}, err
		}
		if !attempt.Status.IsTerminal() {
			if _, err := s.syncAttempt(ctx, attempt); err != nil {
				return Intent{}, err
			}
		}
	}
	var open []RefundRow
	if err := db.Where("intent_id = ? AND status IN ?", intentID, []RefundStatus{RefundInitiated, RefundPending}).Order("id").Find(&open).Error; err != nil {
		return Intent{}, fmt.Errorf("paymentswitch: load open refunds: %w", err)
	}
	for _, r := range open {
		if err := s.syncRefund(ctx, r); err != nil {
			return Intent{}, err
		}
	}
	return s.current(ctx, merchantID, intentID)
}

// syncAttempt is one reconciliation step; it never invents a status: ErrNotFound only moves the sync schedule.
func (s *Service) syncAttempt(ctx context.Context, attempt AttemptRow) (applyResult, error) {
	conn, err := s.connector(attempt.ConnectorCode)
	if err != nil {
		return applyResult{}, err
	}
	if !conn.Capabilities().Sync {
		return applyResult{}, nil
	}
	if err := s.touchAttemptSync(ctx, attempt.ID); err != nil {
		return applyResult{}, err
	}
	req := connectors.SyncRequest{AttemptID: attempt.ID}
	if attempt.ConnectorTransactionID != nil {
		req.ConnectorTransactionID = *attempt.ConnectorTransactionID
	}
	resp, err := conn.Sync(ctx, req)
	if err != nil {
		if errors.Is(err, connectors.ErrNotFound) {
			s.logf("[paymentswitch] sync: connector %s does not know attempt %s yet", attempt.ConnectorCode, attempt.ID)
			return applyResult{}, nil
		}
		return applyResult{}, fmt.Errorf("paymentswitch: sync at %s: %w", attempt.ConnectorCode, err)
	}
	status, err := MapAttemptStatus(attempt.ConnectorCode, resp.RawStatus)
	if err != nil {
		return applyResult{}, s.unmapped(ctx, attempt, resp.RawStatus, err)
	}
	update := attemptUpdate{
		status: status, raw: resp.RawStatus, connectorTransactionID: resp.ConnectorTransactionID,
		amountCaptured: resp.AmountCaptured, amountReceived: resp.AmountReceived, receivedAsset: resp.ReceivedAsset, nextAction: resp.NextAction,
		errorCode: resp.ErrorCode, errorMessage: resp.ErrorMessage, reason: "sync",
	}
	_, result, err := s.applyToActiveAttempt(ctx, attempt.MerchantID, attempt.IntentID, attempt.ID, update)
	if err != nil {
		if errors.Is(err, ErrAmountUnknown) {
			return applyResult{ignored: err.Error()}, nil
		}
		return applyResult{}, err
	}
	if result.ignored != "" {
		s.logf("[paymentswitch] sync: attempt %s reported %q, not applied: %s", attempt.ID, resp.RawStatus, result.ignored)
	}
	return result, nil
}

func (s *Service) unmapped(ctx context.Context, attempt AttemptRow, raw connectors.RawStatus, err error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.recordAnomaly(tx, "attempt", attempt.ID, attempt.IntentID, AnomalyUnmappedStatus, fmt.Sprintf("connector %s reported %q: %v", attempt.ConnectorCode, raw, err))
	})
}

// touchAttemptSync advances the schedule before the call so a crash mid-sync still backs off.
func (s *Service) touchAttemptSync(ctx context.Context, attemptID string) error {
	var row AttemptRow
	if err := s.db.WithContext(ctx).Select("sync_count").Where("id = ?", attemptID).First(&row).Error; err != nil {
		return fmt.Errorf("paymentswitch: touch sync: %w", err)
	}
	now := s.now()
	next := now.Add(backoff(row.SyncCount+1, s.syncBase()))
	return s.db.WithContext(ctx).Model(&AttemptRow{}).Where("id = ?", attemptID).
		Updates(map[string]any{"sync_count": row.SyncCount + 1, "next_sync_at": next, "last_synced_at": now}).Error
}

func (s *Service) touchRefundSync(ctx context.Context, refundID string) error {
	var row RefundRow
	if err := s.db.WithContext(ctx).Select("sync_count").Where("id = ?", refundID).First(&row).Error; err != nil {
		return fmt.Errorf("paymentswitch: touch refund sync: %w", err)
	}
	now := s.now()
	next := now.Add(backoff(row.SyncCount+1, s.syncBase()))
	return s.db.WithContext(ctx).Model(&RefundRow{}).Where("id = ?", refundID).
		Updates(map[string]any{"sync_count": row.SyncCount + 1, "next_sync_at": next, "last_synced_at": now}).Error
}

// syncRefund resolves an initiated or pending refund through SyncRefund (I4).
func (s *Service) syncRefund(ctx context.Context, refund RefundRow) error {
	conn, err := s.connector(refund.ConnectorCode)
	if err != nil {
		return err
	}
	if !conn.Capabilities().RefundSync {
		return nil
	}
	if err := s.touchRefundSync(ctx, refund.ID); err != nil {
		return err
	}
	attempt, err := loadAttempt(s.db.WithContext(ctx), refund.AttemptID)
	if err != nil {
		return err
	}
	req := connectors.SyncRefundRequest{RefundID: refund.ID, AttemptID: refund.AttemptID}
	if attempt.ConnectorTransactionID != nil {
		req.ConnectorTransactionID = *attempt.ConnectorTransactionID
	}
	if refund.ConnectorRefundID != nil {
		req.ConnectorRefundID = *refund.ConnectorRefundID
	}
	resp, err := conn.SyncRefund(ctx, req)
	if err != nil {
		if errors.Is(err, connectors.ErrNotFound) {
			s.logf("[paymentswitch] sync: connector %s does not know refund %s yet", refund.ConnectorCode, refund.ID)
			return nil
		}
		return fmt.Errorf("paymentswitch: sync refund at %s: %w", refund.ConnectorCode, err)
	}
	status, err := MapRefundStatus(refund.ConnectorCode, resp.RawStatus)
	if err != nil {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return s.recordAnomaly(tx, "refund", refund.ID, refund.IntentID, AnomalyUnmappedStatus, fmt.Sprintf("connector %s reported %q: %v", refund.ConnectorCode, resp.RawStatus, err))
		})
	}
	_, result, err := s.applyToRefund(ctx, refund.MerchantID, refund.ID, refundUpdate{status: status, raw: resp.RawStatus, connectorRefundID: resp.ConnectorRefundID, errorCode: resp.ErrorCode, errorMessage: resp.ErrorMessage, reason: "sync"})
	if err != nil {
		return err
	}
	if result.ignored != "" {
		s.logf("[paymentswitch] sync: refund %s reported %q, not applied: %s", refund.ID, resp.RawStatus, result.ignored)
	}
	return nil
}

// WebhookResult says what a delivery did; Ignored means verified and recorded but not applied, with the reason.
type WebhookResult struct {
	EventID   string
	Kind      connectors.WebhookKind
	IntentID  string
	Ignored   bool
	IgnoreWhy string
}

// HandleWebhook verifies a delivery with the connector, records its event id and applies it to the attempt or
// refund it names, all in one transaction. A replay or an out-of-order event is answered as ignored (I3);
// evidence against a terminal state is recorded as an anomaly (M10).
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
	if ev.Kind != connectors.WebhookPayment && ev.Kind != connectors.WebhookRefund {
		return WebhookResult{}, fmt.Errorf("%w: unknown webhook kind %q", connectors.ErrWebhookMalformed, ev.Kind)
	}
	result := WebhookResult{EventID: ev.EventID, Kind: ev.Kind}
	var events []Event
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "connector_code"}, {Name: "event_id"}}, DoNothing: true}).
			Create(&WebhookEventRow{ConnectorCode: code, EventID: ev.EventID, ReceivedAt: s.now()})
		if res.Error != nil {
			return fmt.Errorf("paymentswitch: record webhook: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			result.Ignored, result.IgnoreWhy = true, "replay"
			return nil
		}
		var applied applyResult
		var err error
		if ev.Kind == connectors.WebhookRefund {
			applied, err = s.applyRefundWebhook(tx, code, ev, &result)
		} else {
			applied, err = s.applyPaymentWebhook(tx, code, ev, &result)
		}
		if err != nil {
			return err
		}
		events = applied.events
		if applied.ignored != "" {
			result.Ignored, result.IgnoreWhy = true, applied.ignored
		}
		return nil
	})
	if err != nil {
		return WebhookResult{}, err
	}
	s.emit(ctx, events)
	return result, nil
}

// applyPaymentWebhook finds the attempt without a lock, then locks intent then attempt (one order everywhere, I10).
func (s *Service) applyPaymentWebhook(tx *gorm.DB, code connectors.Code, ev connectors.WebhookEvent, result *WebhookResult) (applyResult, error) {
	var peek AttemptRow
	err := tx.Where("connector_code = ? AND connector_transaction_id = ?", code, ev.ConnectorTransactionID).First(&peek).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return applyResult{}, fmt.Errorf("%w: no attempt for %s transaction %q", ErrNotFound, code, ev.ConnectorTransactionID)
	}
	if err != nil {
		return applyResult{}, fmt.Errorf("paymentswitch: load attempt by connector transaction: %w", err)
	}
	intent, err := loadIntent(lockIfPostgres(tx), peek.MerchantID, peek.IntentID)
	if err != nil {
		return applyResult{}, err
	}
	attempt, err := loadAttempt(lockIfPostgres(tx), peek.ID)
	if err != nil {
		return applyResult{}, err
	}
	result.IntentID = intent.ID
	status, err := MapAttemptStatus(code, ev.RawStatus)
	if err != nil {
		if aerr := s.recordAnomaly(tx, "attempt", attempt.ID, intent.ID, AnomalyUnmappedStatus, fmt.Sprintf("webhook %s reported %q: %v", ev.EventID, ev.RawStatus, err)); aerr != nil {
			return applyResult{}, aerr
		}
		return applyResult{ignored: "connector status unmapped; recorded as anomaly"}, nil
	}
	if intent.ActiveAttemptID != attempt.ID {
		if status.MoneyIn() {
			if err := s.recordAnomaly(tx, "attempt", attempt.ID, intent.ID, AnomalyEvidenceAfterTerminal, fmt.Sprintf("webhook %s reports money in on an attempt that is no longer active", ev.EventID)); err != nil {
				return applyResult{}, err
			}
		}
		return applyResult{ignored: "attempt is no longer active"}, nil
	}
	update := attemptUpdate{status: status, raw: ev.RawStatus, amountCaptured: ev.AmountCaptured, amountReceived: ev.AmountReceived, receivedAsset: ev.ReceivedAsset, reason: "webhook " + ev.EventID}
	applied, err := s.applyAttempt(tx, &intent, &attempt, update)
	if errors.Is(err, ErrAmountUnknown) {
		return applyResult{ignored: err.Error()}, nil
	}
	return applied, err
}

func (s *Service) applyRefundWebhook(tx *gorm.DB, code connectors.Code, ev connectors.WebhookEvent, result *WebhookResult) (applyResult, error) {
	var peek RefundRow
	err := tx.Where("connector_code = ? AND connector_refund_id = ?", code, ev.ConnectorRefundID).First(&peek).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return applyResult{}, fmt.Errorf("%w: no refund for %s refund %q", ErrNotFound, code, ev.ConnectorRefundID)
	}
	if err != nil {
		return applyResult{}, fmt.Errorf("paymentswitch: load refund by connector id: %w", err)
	}
	intent, err := loadIntent(lockIfPostgres(tx), peek.MerchantID, peek.IntentID)
	if err != nil {
		return applyResult{}, err
	}
	refund, err := loadRefund(lockIfPostgres(tx), peek.ID)
	if err != nil {
		return applyResult{}, err
	}
	result.IntentID = intent.ID
	status, err := MapRefundStatus(code, ev.RawStatus)
	if err != nil {
		if aerr := s.recordAnomaly(tx, "refund", refund.ID, intent.ID, AnomalyUnmappedStatus, fmt.Sprintf("webhook %s reported %q: %v", ev.EventID, ev.RawStatus, err)); aerr != nil {
			return applyResult{}, aerr
		}
		return applyResult{ignored: "connector status unmapped; recorded as anomaly"}, nil
	}
	return s.applyRefund(tx, &intent, &refund, refundUpdate{status: status, raw: ev.RawStatus, reason: "webhook " + ev.EventID})
}
