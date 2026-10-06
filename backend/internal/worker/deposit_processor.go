package worker

import (
	"context"
	"log"
	"time"

	"github.com/payminto/payminto/backend/internal/constants"
	"github.com/payminto/payminto/backend/internal/metrics"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/realtime"
	"gorm.io/gorm"
)

// DepositProcessor is a background worker that polls for Deposit rows in the
// CONFIRMING state and marks them CONFIRMED once they reach the required
// on-chain confirmation count.
type DepositProcessor struct {
	db     *gorm.DB
	broker realtime.Broker
}

// NewDepositProcessor creates a DepositProcessor backed by the given database.
// broker may be nil, in which case real-time events are simply not published.
func NewDepositProcessor(db *gorm.DB, broker realtime.Broker) *DepositProcessor {
	return &DepositProcessor{db: db, broker: broker}
}

// Name implements Worker and returns the human-readable identifier for this worker.
func (dp *DepositProcessor) Name() string { return "deposit_processor" }

// Start implements Worker. It polls for CONFIRMING deposits and marks them
// CONFIRMED once their on-chain confirmation count reaches RequiredConfirmations.
// Runs until ctx is cancelled.
func (dp *DepositProcessor) Start(ctx context.Context) error {
	ticker := time.NewTicker(constants.DepositConfirmPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			dp.processConfirmingDeposits()
		}
	}
}

func (dp *DepositProcessor) processConfirmingDeposits() {
	var deposits []models.Deposit
	dp.db.Where("status = ?", models.DepositStatusConfirming).
		Preload("BlockchainCurrency.Blockchain").
		Find(&deposits)

	for _, d := range deposits {
		if d.Confirmations >= d.RequiredConfirmations {
			dp.db.Model(&d).Updates(map[string]any{
				"status": models.DepositStatusConfirmed,
			})

			if d.PaymentRequestID != nil {
				dp.db.Model(&models.PaymentRequest{}).
					Where("id = ?", *d.PaymentRequestID).
					Update("state", models.PaymentStateFilled)
				dp.publishConfirmed(d)
				metrics.PaymentConfirmed()
			}

			log.Printf("Deposit %d confirmed (tx: %s)", d.ID, d.TxID)
		}
	}
}

// publishConfirmed emits a real-time FILLED event on the payment's topic so
// connected checkout clients update instantly. No-op when no broker is wired.
func (dp *DepositProcessor) publishConfirmed(d models.Deposit) {
	if dp.broker == nil || d.PaymentRequestID == nil {
		return
	}
	var pr models.PaymentRequest
	if err := dp.db.Select("reference_id").First(&pr, *d.PaymentRequestID).Error; err != nil {
		return
	}
	dp.broker.Publish(dp.broker.PaymentTopic(pr.ReferenceID), realtime.PaymentEvent{
		ReferenceID:   pr.ReferenceID,
		State:         string(models.PaymentStateFilled),
		Confirmations: d.Confirmations,
		Required:      d.RequiredConfirmations,
		TxID:          d.TxID,
	})
}
