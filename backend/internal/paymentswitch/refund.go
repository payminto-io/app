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

// Refund returns captured money, in full or in part. Keyed by (merchant, idempotency key) like Create; the
// refundable balance counts pending refunds so two in flight cannot over-refund.
func (s *Service) Refund(ctx context.Context, merchantID, intentID string, cmd RefundCommand) (Refund, error) {
	if len(cmd.IdempotencyKey) > maxKeyLen {
		return Refund{}, fmt.Errorf("%w: idempotency key longer than %d", ErrInvalid, maxKeyLen)
	}
	if cmd.IdempotencyKey == "" {
		cmd.IdempotencyKey = s.newID("idem")
	}
	hashInput := canonicalRefund{IntentID: intentID, Reason: cmd.Reason}
	if cmd.Amount != nil {
		hashInput.Amount = cmd.Amount.String()
	}
	hash := hashJSON(hashInput)

	db := s.db.WithContext(ctx)
	intent, err := loadIntent(db, merchantID, intentID)
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

	row := RefundRow{
		ID:             s.newID("re"),
		IntentID:       intent.ID,
		AttemptID:      attempt.ID,
		MerchantID:     merchantID,
		ConnectorCode:  attempt.ConnectorCode,
		IdempotencyKey: cmd.IdempotencyKey,
		RequestHash:    hash,
		Status:         RefundPending,
		Asset:          attempt.Asset,
		Reason:         cmd.Reason,
		CreatedAt:      s.now(),
		UpdatedAt:      s.now(),
	}
	var replayed bool
	err = db.Transaction(func(tx *gorm.DB) error {
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
		// The sum is read under the intent lock so concurrent refunds serialise on Postgres.
		if _, err := loadIntent(lockIfPostgres(tx), merchantID, intentID); err != nil {
			return err
		}
		var committed decimal.Decimal
		if err := tx.Model(&RefundRow{}).Where("intent_id = ? AND status IN ?", intentID, []RefundStatus{RefundPending, RefundSucceeded}).
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
		return s.recordTransition(tx, "refund", row.ID, "", string(RefundPending), "refund")
	})
	if err != nil {
		return Refund{}, err
	}
	if replayed {
		return row.toRefund(), nil
	}

	resp, callErr := conn.Refund(ctx, connectors.RefundRequest{
		RefundID:               row.ID,
		AttemptID:              attempt.ID,
		ConnectorTransactionID: *attempt.ConnectorTransactionID,
		Money:                  Money{Amount: row.Amount, Asset: row.Asset},
		Reason:                 cmd.Reason,
		IdempotencyKey:         row.ID,
	})
	var update refundUpdate
	switch {
	case callErr == nil:
		status, err := MapRefundStatus(attempt.ConnectorCode, resp.RawStatus)
		if err != nil {
			return Refund{}, err
		}
		update = refundUpdate{status: status, raw: resp.RawStatus, connectorRefundID: resp.ConnectorRefundID, errorCode: resp.ErrorCode, errorMessage: resp.ErrorMessage, reason: "refund"}
	case errors.Is(callErr, connectors.ErrTimeout):
		return row.toRefund(), nil
	default:
		update = refundUpdate{status: RefundFailed, errorCode: "connector_error", errorMessage: callErr.Error(), reason: "refund error"}
	}
	return s.applyToRefund(ctx, merchantID, row.ID, update)
}

func (s *Service) applyToRefund(ctx context.Context, merchantID, refundID string, u refundUpdate) (Refund, error) {
	var out RefundRow
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var refund RefundRow
		if err := lockIfPostgres(tx).Where("id = ? AND merchant_id = ?", refundID, merchantID).First(&refund).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("%w: refund %s", ErrNotFound, refundID)
			}
			return fmt.Errorf("paymentswitch: load refund: %w", err)
		}
		intent, err := loadIntent(lockIfPostgres(tx), merchantID, refund.IntentID)
		if err != nil {
			return err
		}
		if err := s.applyRefund(tx, &intent, &refund, u); err != nil {
			return err
		}
		out = refund
		return nil
	})
	if err != nil {
		return Refund{}, err
	}
	return out.toRefund(), nil
}
