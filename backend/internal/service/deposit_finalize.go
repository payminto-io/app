package service

import (
	"context"
	"fmt"

	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// finalizeDeposit marks a deposit confirmed (from pending, confirming or a revived failed) and, in the
// same transaction, posts its payment journal unless the switch owns the payment. It reports whether
// this call performed the transition; a repeat or a racing caller gets false and posts nothing.
// Shared by chain watchers that confirm deposits themselves (Solana, Tempo).
// Optional afterConfirmed callbacks complete related writes before the same transaction commits.
// Callers without a callback finalize the invoice separately and then publish.
func finalizeDeposit(ctx context.Context, db *gorm.DB, ledger *LedgerService, d *models.Deposit, afterConfirmed ...func(*gorm.DB) error) (bool, error) {
	won := false
	post := func(tx *gorm.DB) error {
		res := tx.Model(&models.Deposit{}).
			Where("id = ? AND status IN ?", d.ID, []string{models.DepositStatusPending, models.DepositStatusConfirming, models.DepositStatusFailed}).
			Updates(map[string]any{"confirmations": d.RequiredConfirmations, "status": models.DepositStatusConfirmed})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		won = true
		if ledger != nil {
			owned, err := switchOwnsPayment(tx, d.PaymentRequestID)
			if err != nil {
				return err
			}
			if !owned {
				if err := ledger.RecordDepositIn(ctx, tx, d.ID, d.BlockchainCurrencyID, d.Amount); err != nil {
					return err
				}
			}
		}
		for _, complete := range afterConfirmed {
			if err := complete(tx); err != nil {
				return err
			}
		}
		return nil
	}
	var err error
	if ledger != nil {
		err = ledger.InTransaction(ctx, post)
	} else {
		err = db.WithContext(ctx).Transaction(post)
	}
	if err != nil {
		return false, fmt.Errorf("finalize deposit %d: %w", d.ID, err)
	}
	return won, nil
}
