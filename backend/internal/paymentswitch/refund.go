package paymentswitch

import (
	"context"
	"errors"
	"fmt"

	"github.com/payminto/payminto/backend/internal/connectors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type canonicalRefund struct {
	IntentID string `json:"intent_id"`
	Amount   string `json:"amount,omitempty"`
	Reason   string `json:"reason"`
}

// Refund returns captured money, in full or in part. Keyed by (merchant, idempotency key) like Create; the row
// is the claim (initiated), the refundable balance counts initiated and pending refunds, and a replay of a refund
// whose outcome is unknown re-sends with the same connector key (I2).
func (s *Service) Refund(ctx context.Context, merchantID, intentID string, cmd RefundCommand) (Refund, error) {
	if len(cmd.IdempotencyKey) > maxKeyLen {
		return Refund{}, fmt.Errorf("%w: idempotency key longer than %d", ErrInvalid, maxKeyLen)
	}
	if cmd.IdempotencyKey == "" {
		cmd.IdempotencyKey = s.newID("idem")
	}
	if cmd.Amount != nil {
		if err := validateAmount(*cmd.Amount); err != nil {
			return Refund{}, err
		}
	}
	hashInput := canonicalRefund{IntentID: intentID, Reason: cmd.Reason}
	if cmd.Amount != nil {
		hashInput.Amount = cmd.Amount.String()
	}
	hash := hashJSON(hashInput)
	if err := s.requireEnv(ctx); err != nil {
		return Refund{}, err
	}

	db := s.db.WithContext(ctx)
	intent, err := s.loadIntent(db, merchantID, intentID)
	if err != nil {
		return Refund{}, err
	}
	if intent.Status != IntentSucceeded && intent.Status != IntentPartiallyCaptured {
		return Refund{}, fmt.Errorf("%w: intent %s is %s, refund needs captured funds", ErrInvalidTransition, intentID, intent.Status)
	}
	attempt, err := loadAttempt(db, intent.ActiveAttemptID)
	if err != nil {
		return Refund{}, err
	}
	conn, err := s.connector(attempt.ConnectorCode)
	if err != nil {
		return Refund{}, err
	}
	caps := conn.Capabilities()
	if !caps.Refund {
		return Refund{}, fmt.Errorf("%w: connector %s cannot refund", ErrInvalid, attempt.ConnectorCode)
	}
	if attempt.ConnectorTransactionID == nil {
		return Refund{}, fmt.Errorf("%w: attempt %s has no connector transaction", ErrInvalid, attempt.ID)
	}

	now := s.now()
	lease := now.Add(s.lease)
	row := RefundRow{
		ClaimedUntil:    &lease,
		NextSyncAt:      &lease,
		StatusChangedAt: now,
		ID:              s.newID("re"),
		IntentID:        intent.ID,
		AttemptID:       attempt.ID,
		MerchantID:      merchantID,
		Environment:     s.env(),
		ConnectorCode:   attempt.ConnectorCode,
		IdempotencyKey:  cmd.IdempotencyKey,
		RequestHash:     hash,
		Status:          RefundInitiated,
		Asset:           attempt.Asset,
		Reason:          cmd.Reason,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	var replayed bool
	err = db.Transaction(func(tx *gorm.DB) error {
		// Lock order: intent first, so concurrent refunds serialise and the sum below is read under the lock.
		if _, err := s.loadIntent(lockIfPostgres(tx), merchantID, intentID); err != nil {
			return err
		}
		var existing RefundRow
		err := tx.Where("merchant_id = ? AND idempotency_key = ?", merchantID, cmd.IdempotencyKey).First(&existing).Error
		if err == nil {
			if existing.RequestHash != hash {
				return fmt.Errorf("%w: key %q", ErrIdempotencyConflict, cmd.IdempotencyKey)
			}
			row, replayed = existing, true
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("paymentswitch: load refund for key: %w", err)
		}
		var committed decimal.Decimal
		if err := tx.Model(&RefundRow{}).Where("intent_id = ? AND status IN ?", intentID, []RefundStatus{RefundInitiated, RefundPending, RefundSucceeded}).
			Select("COALESCE(SUM(amount), 0)").Scan(&committed).Error; err != nil {
			return fmt.Errorf("paymentswitch: sum refunds: %w", err)
		}
		refundable := intent.AmountCaptured.Sub(committed)
		amount := refundable
		if cmd.Amount != nil {
			amount = *cmd.Amount
		}
		if !amount.IsPositive() || amount.GreaterThan(refundable) {
			return fmt.Errorf("%w: refund %s, refundable %s %s", ErrAmountExceeds, amount, refundable, attempt.Asset)
		}
		if amount.LessThan(intent.AmountCaptured) && !caps.PartialRefund {
			return fmt.Errorf("%w: connector %s cannot refund partially", ErrInvalid, attempt.ConnectorCode)
		}
		row.Amount = amount
		res := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "merchant_id"}, {Name: "idempotency_key"}}, DoNothing: true}).Create(&row)
		if res.Error != nil {
			return fmt.Errorf("paymentswitch: insert refund: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("%w: refund key %q raced", ErrConcurrentUpdate, cmd.IdempotencyKey)
		}
		return s.recordTransition(tx, "refund", row.ID, "", string(RefundInitiated), "refund")
	})
	if err != nil {
		return Refund{}, err
	}
	if replayed {
		// An initiated refund is resolved by the reconciler with SyncRefund; a replay reports it as it is.
		return row.toRefund(), nil
	}
	return s.sendRefund(ctx, conn, row, attempt)
}

// sendRefund calls the connector with the refund's own id as the key and applies the outcome; an unknown outcome
// leaves the row initiated for the reconciler.
func (s *Service) sendRefund(ctx context.Context, conn connectors.Connector, row RefundRow, attempt AttemptRow) (Refund, error) {
	callCtx, cancelCall := s.callContext(ctx)
	resp, callErr := conn.Refund(callCtx, connectors.RefundRequest{
		RefundID:               row.ID,
		AttemptID:              attempt.ID,
		ConnectorTransactionID: *attempt.ConnectorTransactionID,
		Money:                  Money{Amount: row.Amount, Asset: row.Asset},
		Reason:                 row.Reason,
		IdempotencyKey:         row.ID,
	})
	cancelCall()
	var update refundUpdate
	switch {
	case callErr == nil:
		status, err := MapRefundStatus(attempt.ConnectorCode, resp.RawStatus)
		if err != nil {
			return Refund{}, err
		}
		update = refundUpdate{status: status, raw: resp.RawStatus, connectorRefundID: resp.ConnectorRefundID, errorCode: resp.ErrorCode, errorMessage: resp.ErrorMessage, reason: "refund"}
	case connectors.Definitive(callErr):
		ec, em := redact(callErr)
		update = refundUpdate{status: RefundFailed, errorCode: ec, errorMessage: em, reason: "refund declined: " + callErr.Error()}
	default:
		return s.markRefundUnknown(ctx, row.MerchantID, row.ID, callErr)
	}
	out, _, err := s.applyToRefund(ctx, row.MerchantID, row.ID, update)
	return out, err
}

func (s *Service) markRefundUnknown(ctx context.Context, merchantID, refundID string, callErr error) (Refund, error) {
	ec, em := redact(callErr)
	s.logf("[paymentswitch] refund outcome unknown for %s: %v", refundID, callErr)
	var out RefundRow
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		refund, err := loadRefund(lockIfPostgres(tx), refundID)
		if err != nil {
			return err
		}
		refund.ErrorCode, refund.ErrorMessage = ec, em
		if err := s.saveRefund(tx, &refund); err != nil {
			return err
		}
		out = refund
		return s.recordNote(tx, "refund", refund.ID, string(refund.Status), "refund outcome unknown: "+callErr.Error())
	})
	if err != nil {
		return Refund{}, err
	}
	return out.toRefund(), nil
}

// applyToRefund locks in order (intent, then refund) and applies one update in its own transaction.
func (s *Service) applyToRefund(ctx context.Context, merchantID, refundID string, u refundUpdate) (Refund, applyResult, error) {
	var out RefundRow
	var result applyResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		peek, err := loadRefund(tx, refundID)
		if err != nil {
			return err
		}
		if peek.MerchantID != merchantID {
			return fmt.Errorf("%w: refund %s", ErrNotFound, refundID)
		}
		intent, err := s.loadIntent(lockIfPostgres(tx), merchantID, peek.IntentID)
		if err != nil {
			return err
		}
		refund, err := loadRefund(lockIfPostgres(tx), refundID)
		if err != nil {
			return err
		}
		result, err = s.applyRefund(tx, &intent, &refund, u)
		if err != nil {
			return err
		}
		out = refund
		return nil
	})
	if err != nil {
		return Refund{}, applyResult{}, err
	}
	s.emit(ctx, result.events)
	return out.toRefund(), result, nil
}
