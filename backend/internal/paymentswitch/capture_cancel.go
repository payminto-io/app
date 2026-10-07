package paymentswitch

import (
	"context"
	"errors"
	"fmt"

	"github.com/payminto/payminto/backend/internal/connectors"
	"gorm.io/gorm"
)

// Capture settles an authorized attempt, in full or in part. A partial capture is terminal: the remainder
// cannot be captured later (Hyperswitch semantics).
func (s *Service) Capture(ctx context.Context, merchantID, intentID string, cmd CaptureCommand) (Intent, error) {
	db := s.db.WithContext(ctx)
	intent, err := loadIntent(db, merchantID, intentID)
	if err != nil {
		return Intent{}, err
	}
	if intent.Status != IntentRequiresCapture || intent.ActiveAttemptID == "" {
		return Intent{}, fmt.Errorf("%w: intent %s is %s, capture needs requires_capture", ErrInvalidTransition, intentID, intent.Status)
	}
	attempt, err := loadAttempt(db, intent.ActiveAttemptID)
	if err != nil {
		return Intent{}, err
	}
	amount := attempt.Amount
	if cmd.Amount != nil {
		amount = *cmd.Amount
	}
	if !amount.IsPositive() || amount.GreaterThan(attempt.Amount) {
		return Intent{}, fmt.Errorf("%w: capture %s of %s %s", ErrAmountExceeds, amount, attempt.Amount, attempt.Asset)
	}
	conn, err := s.connector(attempt.ConnectorCode)
	if err != nil {
		return Intent{}, err
	}
	if amount.LessThan(attempt.Amount) && !conn.Capabilities().PartialCapture {
		return Intent{}, fmt.Errorf("%w: connector %s cannot capture partially", ErrInvalid, attempt.ConnectorCode)
	}
	if attempt.ConnectorTransactionID == nil {
		return Intent{}, fmt.Errorf("%w: attempt %s has no connector transaction", ErrInvalid, attempt.ID)
	}

	if _, err := s.applyToActiveAttempt(ctx, merchantID, intentID, attempt.ID, attemptUpdate{status: AttemptCaptureInitiated, reason: "capture"}); err != nil {
		return Intent{}, err
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
		captured := resp.AmountCaptured
		if captured.IsZero() {
			captured = amount
		}
		update = attemptUpdate{status: status, raw: resp.RawStatus, amountCaptured: &captured, errorCode: resp.ErrorCode, errorMessage: resp.ErrorMessage, reason: "capture"}
	case errors.Is(callErr, connectors.ErrTimeout):
		return s.current(ctx, merchantID, intentID)
	default:
		update = attemptUpdate{status: AttemptCaptureFailed, errorCode: "connector_error", errorMessage: callErr.Error(), reason: "capture error"}
	}
	return s.applyToActiveAttempt(ctx, merchantID, intentID, attempt.ID, update)
}

// current re-reads the intent after a timed-out connector call; Sync resolves it later.
func (s *Service) current(ctx context.Context, merchantID, intentID string) (Intent, error) {
	row, err := loadIntent(s.db.WithContext(ctx), merchantID, intentID)
	if err != nil {
		return Intent{}, err
	}
	return row.toIntent(), nil
}

// Cancel voids an intent. Before any attempt it is a local transition; with an attempt holding an
// authorization or still pending it voids at the connector.
func (s *Service) Cancel(ctx context.Context, merchantID, intentID string, cmd CancelCommand) (Intent, error) {
	db := s.db.WithContext(ctx)
	intent, err := loadIntent(db, merchantID, intentID)
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
	if attempt.Status.IsTerminal() && attempt.Status != AttemptAuthorizationFailed && attempt.Status != AttemptFailure {
		return Intent{}, fmt.Errorf("%w: attempt %s is %s", ErrInvalidTransition, attempt.ID, attempt.Status)
	}
	if attempt.Status == AttemptAuthorizationFailed || attempt.Status == AttemptFailure {
		return s.cancelLocally(ctx, merchantID, intentID, reason)
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
	claimed := attempt.Status.CanTransitionTo(AttemptVoidInitiated)
	if claimed {
		if _, err := s.applyToActiveAttempt(ctx, merchantID, intentID, attempt.ID, attemptUpdate{status: AttemptVoidInitiated, reason: reason}); err != nil {
			return Intent{}, err
		}
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
		update = attemptUpdate{status: status, raw: resp.RawStatus, errorCode: resp.ErrorCode, errorMessage: resp.ErrorMessage, reason: reason}
	case errors.Is(callErr, connectors.ErrTimeout):
		return s.current(ctx, merchantID, intentID)
	case claimed:
		update = attemptUpdate{status: AttemptVoidFailed, errorCode: "connector_error", errorMessage: callErr.Error(), reason: "void error"}
	default:
		return Intent{}, fmt.Errorf("paymentswitch: void at %s: %w", attempt.ConnectorCode, callErr)
	}
	return s.applyToActiveAttempt(ctx, merchantID, intentID, attempt.ID, update)
}

func (s *Service) cancelLocally(ctx context.Context, merchantID, intentID, reason string) (Intent, error) {
	var out IntentRow
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		intent, err := loadIntent(lockIfPostgres(tx), merchantID, intentID)
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
