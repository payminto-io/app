package paymentswitch

import (
	"context"
	"fmt"

	"github.com/payminto/payminto/backend/internal/connectors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// claimAttempt is the exclusive compare-and-set into an in-flight status (C2): the attempt row moves only if it
// still has the status and version the caller read, and the intent follows in the same transaction.
func (s *Service) claimAttempt(ctx context.Context, intent IntentRow, attempt AttemptRow, to AttemptStatus, amountToCapture *decimal.Decimal, reason string) error {
	if err := transitionAttempt(attempt.Status, to); err != nil || attempt.Status == to {
		return fmt.Errorf("%w: attempt %s is %s", ErrInvalidTransition, attempt.ID, attempt.Status)
	}
	derived := IntentStatusFor(to)
	if err := transitionIntent(intent.Status, derived); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock order: intent first, then the attempt (I10); the version check keeps the claim compare-and-set.
		current, err := s.loadIntent(lockIfPostgres(tx), intent.MerchantID, intent.ID)
		if err != nil {
			return err
		}
		if current.Version != intent.Version {
			return fmt.Errorf("%w: intent %s", ErrConcurrentUpdate, intent.ID)
		}
		now := s.now()
		lease := now.Add(s.lease)
		updates := map[string]any{
			"status": to, "error_code": "", "error_message": "", "claimed_until": lease, "next_sync_at": lease,
			"status_changed_at": now, "version": attempt.Version + 1, "updated_at": now,
		}
		if amountToCapture != nil {
			updates["amount_to_capture"] = *amountToCapture
		}
		res := tx.Model(&AttemptRow{}).Where("id = ? AND status = ? AND version = ?", attempt.ID, attempt.Status, attempt.Version).Updates(updates)
		if res.Error != nil {
			return fmt.Errorf("paymentswitch: claim attempt: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("%w: attempt %s was claimed by another request", ErrInvalidTransition, attempt.ID)
		}
		res = tx.Model(&IntentRow{}).Where("id = ? AND version = ?", intent.ID, intent.Version).
			Updates(map[string]any{"status": derived, "last_error_code": "", "last_error_message": "", "version": intent.Version + 1, "updated_at": s.now()})
		if res.Error != nil {
			return fmt.Errorf("paymentswitch: claim intent: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("%w: intent %s", ErrConcurrentUpdate, intent.ID)
		}
		if err := s.recordTransition(tx, "attempt", attempt.ID, string(attempt.Status), string(to), reason); err != nil {
			return err
		}
		return s.recordTransition(tx, "intent", intent.ID, string(intent.Status), string(derived), reason)
	})
}

// Capture settles an authorized attempt, in full or in part. A partial capture is terminal: the remainder cannot
// be captured later. A capture whose outcome is unknown is retried with the same connector key and amount.
func (s *Service) Capture(ctx context.Context, merchantID, intentID string, cmd CaptureCommand) (Intent, error) {
	if err := s.requireEnv(ctx); err != nil {
		return Intent{}, err
	}
	db := s.db.WithContext(ctx)
	intent, err := s.loadIntent(db, merchantID, intentID)
	if err != nil {
		return Intent{}, err
	}
	if intent.ActiveAttemptID == "" {
		return Intent{}, fmt.Errorf("%w: intent %s has no attempt to capture", ErrInvalidTransition, intentID)
	}
	attempt, err := loadAttempt(db, intent.ActiveAttemptID)
	if err != nil {
		return Intent{}, err
	}
	if cmd.Amount != nil {
		if err := validateAmount(*cmd.Amount); err != nil {
			return Intent{}, err
		}
	}
	conn, err := s.connector(attempt.ConnectorCode)
	if err != nil {
		return Intent{}, err
	}
	if attempt.ConnectorTransactionID == nil {
		return Intent{}, fmt.Errorf("%w: attempt %s has no connector transaction; sync first", ErrInvalid, attempt.ID)
	}

	amount := attempt.Amount
	if cmd.Amount != nil {
		amount = *cmd.Amount
	}
	switch attempt.Status {
	case AttemptCaptureInitiated:
		// A second capture while one is in flight is a conflict (R1); the reconciler resolves the first with Sync.
		return Intent{}, fmt.Errorf("%w: a capture of %s is in flight on attempt %s", ErrInvalidTransition, attempt.AmountToCapture, attempt.ID)
	case AttemptAuthorized, AttemptCaptureFailed:
		if amount.GreaterThan(attempt.Amount) {
			return Intent{}, fmt.Errorf("%w: capture %s of %s %s", ErrAmountExceeds, amount, attempt.Amount, attempt.Asset)
		}
		if amount.LessThan(attempt.Amount) && !conn.Capabilities().PartialCapture {
			return Intent{}, fmt.Errorf("%w: connector %s cannot capture partially", ErrInvalid, attempt.ConnectorCode)
		}
		if err := s.claimAttempt(ctx, intent, attempt, AttemptCaptureInitiated, &amount, "capture"); err != nil {
			return Intent{}, err
		}
	default:
		return Intent{}, fmt.Errorf("%w: attempt %s is %s, capture needs an authorization", ErrInvalidTransition, attempt.ID, attempt.Status)
	}

	resp, callErr := conn.Capture(ctx, connectors.CaptureRequest{
		AttemptID:              attempt.ID,
		ConnectorTransactionID: *attempt.ConnectorTransactionID,
		Money:                  Money{Amount: amount, Asset: attempt.Asset},
		IdempotencyKey:         attempt.ID + ".capture",
	})
	var update attemptUpdate
	switch {
	case callErr == nil:
		status, err := MapAttemptStatus(attempt.ConnectorCode, resp.RawStatus)
		if err != nil {
			return Intent{}, err
		}
		update = attemptUpdate{status: status, raw: resp.RawStatus, errorCode: resp.ErrorCode, errorMessage: resp.ErrorMessage, reason: "capture"}
		if resp.AmountCaptured.IsPositive() {
			captured := resp.AmountCaptured
			update.amountCaptured = &captured
		}
	case connectors.Definitive(callErr):
		ec, em := redact(callErr)
		update = attemptUpdate{status: AttemptCaptureFailed, errorCode: ec, errorMessage: em, reason: "capture declined: " + callErr.Error()}
	default:
		return s.markUnknown(ctx, merchantID, intentID, attempt.ID, "capture", callErr)
	}
	out, _, err := s.applyToActiveAttempt(ctx, merchantID, intentID, attempt.ID, update)
	return out, err
}

// markUnknown keeps an in-flight attempt where it is, records the redacted error and schedules the reconciler.
func (s *Service) markUnknown(ctx context.Context, merchantID, intentID, attemptID, op string, callErr error) (Intent, error) {
	ec, em := redact(callErr)
	s.logf("[paymentswitch] %s outcome unknown for attempt %s: %v", op, attemptID, callErr)
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		intent, err := s.loadIntent(lockIfPostgres(tx), merchantID, intentID)
		if err != nil {
			return err
		}
		attempt, err := loadAttempt(lockIfPostgres(tx), attemptID)
		if err != nil {
			return err
		}
		attempt.ErrorCode, attempt.ErrorMessage = ec, em
		intent.LastErrorCode, intent.LastErrorMessage = ec, em
		if err := s.saveAttempt(tx, &attempt); err != nil {
			return err
		}
		if err := s.saveIntent(tx, &intent); err != nil {
			return err
		}
		return s.recordNote(tx, "attempt", attempt.ID, string(attempt.Status), op+" outcome unknown: "+callErr.Error())
	})
	if err != nil {
		return Intent{}, err
	}
	return s.current(ctx, merchantID, intentID)
}

func (s *Service) current(ctx context.Context, merchantID, intentID string) (Intent, error) {
	row, err := s.loadIntent(s.db.WithContext(ctx), merchantID, intentID)
	if err != nil {
		return Intent{}, err
	}
	return row.toIntent(), nil
}

// Cancel voids an intent. Before any attempt it is a local transition; with an attempt it voids at the connector
// after an exclusive claim into void_initiated. A partially paid deposit may be cancelled: the received funds stay on
// the books and await refund (ticket 11).
func (s *Service) Cancel(ctx context.Context, merchantID, intentID string, cmd CancelCommand) (Intent, error) {
	if err := s.requireEnv(ctx); err != nil {
		return Intent{}, err
	}
	db := s.db.WithContext(ctx)
	intent, err := s.loadIntent(db, merchantID, intentID)
	if err != nil {
		return Intent{}, err
	}
	reason := "cancel"
	if cmd.Reason != "" {
		reason = "cancel: " + cmd.Reason
	}
	if intent.ActiveAttemptID == "" {
		return s.cancelLocally(ctx, merchantID, intentID, reason)
	}
	attempt, err := loadAttempt(db, intent.ActiveAttemptID)
	if err != nil {
		return Intent{}, err
	}
	switch attempt.Status {
	case AttemptAuthorizationFailed, AttemptFailure:
		return s.cancelLocally(ctx, merchantID, intentID, reason)
	case AttemptVoidInitiated:
		return Intent{}, fmt.Errorf("%w: a void is in flight on attempt %s; the reconciler resolves it", ErrInvalidTransition, attempt.ID)
	default:
		if attempt.Status.IsTerminal() {
			return Intent{}, fmt.Errorf("%w: attempt %s is %s", ErrInvalidTransition, attempt.ID, attempt.Status)
		}
		if !attempt.Status.CanTransitionTo(AttemptVoidInitiated) {
			return Intent{}, fmt.Errorf("%w: attempt %s is %s; its outcome must be synced before it can be cancelled", ErrInvalidTransition, attempt.ID, attempt.Status)
		}
	}
	conn, err := s.connector(attempt.ConnectorCode)
	if err != nil {
		return Intent{}, err
	}
	if !conn.Capabilities().Void {
		return Intent{}, fmt.Errorf("%w: connector %s cannot void", ErrInvalid, attempt.ConnectorCode)
	}
	if attempt.ConnectorTransactionID == nil {
		return Intent{}, fmt.Errorf("%w: attempt %s has no connector transaction yet; sync first", ErrInvalid, attempt.ID)
	}
	if attempt.Status == AttemptPartiallyPaid {
		reason += " (received funds await refund)"
	}
	if err := s.claimAttempt(ctx, intent, attempt, AttemptVoidInitiated, nil, reason); err != nil {
		return Intent{}, err
	}
	resp, callErr := conn.Void(ctx, connectors.VoidRequest{
		AttemptID:              attempt.ID,
		ConnectorTransactionID: *attempt.ConnectorTransactionID,
		Reason:                 cmd.Reason,
		IdempotencyKey:         attempt.ID + ".void",
	})
	var update attemptUpdate
	switch {
	case callErr == nil:
		status, err := MapAttemptStatus(attempt.ConnectorCode, resp.RawStatus)
		if err != nil {
			return Intent{}, err
		}
		update = attemptUpdate{status: status, raw: resp.RawStatus, amountReceived: resp.AmountReceived, receivedAsset: resp.ReceivedAsset, errorCode: resp.ErrorCode, errorMessage: resp.ErrorMessage, reason: reason}
	case connectors.Definitive(callErr):
		ec, em := redact(callErr)
		update = attemptUpdate{status: AttemptVoidFailed, errorCode: ec, errorMessage: em, reason: "void declined: " + callErr.Error()}
	default:
		return s.markUnknown(ctx, merchantID, intentID, attempt.ID, "void", callErr)
	}
	out, _, err := s.applyToActiveAttempt(ctx, merchantID, intentID, attempt.ID, update)
	return out, err
}

func (s *Service) cancelLocally(ctx context.Context, merchantID, intentID, reason string) (Intent, error) {
	var out IntentRow
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		intent, err := s.loadIntent(lockIfPostgres(tx), merchantID, intentID)
		if err != nil {
			return err
		}
		if err := transitionIntent(intent.Status, IntentCancelled); err != nil {
			return err
		}
		from := intent.Status
		intent.Status = IntentCancelled
		intent.NextAction = nil
		if err := s.saveIntent(tx, &intent); err != nil {
			return err
		}
		if err := s.recordTransition(tx, "intent", intent.ID, string(from), string(IntentCancelled), reason); err != nil {
			return err
		}
		out = intent
		return nil
	})
	if err != nil {
		return Intent{}, err
	}
	return out.toIntent(), nil
}
