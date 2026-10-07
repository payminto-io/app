package chaindeposit_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/payminto/payminto/backend/internal/connectors"
	"github.com/payminto/payminto/backend/internal/connectors/chaindeposit"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// dbOpener does what PaymentService.CreatePayment does for the fields the backend relies on: one insert that
// carries the invoice id, a Payminto-generated reference, and a deposit address on the blockchain currency.
type dbOpener struct {
	db  *gorm.DB
	seq int
}

func (o *dbOpener) open(amount decimal.Decimal, invoiceID, _, _ string, memberID, platformID uint) (chaindeposit.Opened, error) {
	o.seq++
	expires := time.Now().Add(30 * time.Minute)
	p := &models.PaymentRequest{ReferenceID: uuid.NewString(), AmountInUSD: amount, State: models.PaymentStateOpen, InvoiceID: &invoiceID, ExpiresAt: &expires, MemberID: memberID, ExternalPlatformID: platformID}
	if err := o.db.Create(p).Error; err != nil {
		return chaindeposit.Opened{}, err
	}
	addr := &models.DepositAddress{Address: "0xaddr" + p.ReferenceID[:6], BlockchainCurrencyID: 1, MemberID: memberID, PaymentRequestID: &p.ID}
	if err := o.db.Create(addr).Error; err != nil {
		return chaindeposit.Opened{}, err
	}
	return chaindeposit.Opened{Reference: p.ReferenceID, Address: addr.Address, ExpiresAt: p.ExpiresAt}, nil
}

func paymintoFixture(t *testing.T) (*gorm.DB, *chaindeposit.PaymintoBackend) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.PaymentRequest{}, &models.Deposit{}, &models.DepositAddress{}, &models.BlockchainCurrency{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.BlockchainCurrency{CurrencyCode: "USDC", BlockchainCode: "ETH"}).Error; err != nil {
		t.Fatal(err)
	}
	return db, chaindeposit.NewPaymintoBackend((&dbOpener{db: db}).open, repository.NewPaymentRepository(db), repository.NewDepositRepository(db), db)
}

// confirmDeposit books a confirmed deposit and runs Payminto's own finalizer, exactly as the block processor path does.
func confirmDeposit(t *testing.T, db *gorm.DB, reference string, amount decimal.Decimal) {
	t.Helper()
	var p models.PaymentRequest
	if err := db.Where("reference_id = ?", reference).First(&p).Error; err != nil {
		t.Fatal(err)
	}
	d := models.Deposit{TxID: "tx-" + decimal.NewFromInt(time.Now().UnixNano()).String(), Amount: amount, Status: models.DepositStatusConfirmed, ToAddress: "0x", BlockchainCurrencyID: 1, MemberID: p.MemberID, PaymentRequestID: &p.ID}
	if err := db.Create(&d).Error; err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := repository.NewPaymentRepository(db).FinalizeFromConfirmedDeposits(p.ID); err != nil {
		t.Fatal(err)
	}
}

func TestPaymintoBackendSuite(t *testing.T) {
	var db *gorm.DB
	chaindeposit.RunBackendSuite(t, chaindeposit.BackendHarness{
		New: func(t *testing.T) chaindeposit.Backend {
			var b *chaindeposit.PaymintoBackend
			db, b = paymintoFixture(t)
			return b
		},
		Confirm: func(t *testing.T, _ chaindeposit.Backend, ref string, amount decimal.Decimal) {
			confirmDeposit(t, db, ref, amount)
		},
	})
}

func TestPaymintoBackend_StatusSumsOnlyConfirmedDeposits(t *testing.T) {
	db, b := paymintoFixture(t)
	ctx := context.Background()
	res, err := b.OpenPayment(ctx, chaindeposit.OpenRequest{MerchantMemberID: 7, PlatformID: 3, AmountInUSD: decimal.NewFromInt(100), ChainCode: "ETH", CurrencyCode: "USDC", AttemptID: "pa_1"})
	if err != nil {
		t.Fatal(err)
	}
	var p models.PaymentRequest
	db.Where("reference_id = ?", res.Reference).First(&p)
	for _, d := range []models.Deposit{
		{TxID: "a", Amount: decimal.NewFromInt(30), Status: models.DepositStatusConfirmed, ToAddress: "0x", BlockchainCurrencyID: 1, MemberID: 7, PaymentRequestID: &p.ID},
		{TxID: "b", Amount: decimal.NewFromInt(50), Status: models.DepositStatusPending, ToAddress: "0x", BlockchainCurrencyID: 1, MemberID: 7, PaymentRequestID: &p.ID},
		{TxID: "c", Amount: decimal.NewFromInt(20), Status: models.DepositStatusSwept, ToAddress: "0x", BlockchainCurrencyID: 1, MemberID: 7, PaymentRequestID: &p.ID},
	} {
		if err := db.Create(&d).Error; err != nil {
			t.Fatal(err)
		}
	}
	st, err := b.PaymentStatus(ctx, res.Reference)
	if err != nil || st.State != "OPEN" || !st.Received.Equal(decimal.NewFromInt(50)) || st.CurrencyCode != "USDC" || st.ChainCode != "ETH" {
		t.Fatalf("status = %+v, %v", st, err)
	}
}

func TestPaymintoBackend_OpenRequiresADepositAddressAndTheConnectorUsesIt(t *testing.T) {
	_, b := paymintoFixture(t)
	noAddr := chaindeposit.NewPaymintoBackend(func(decimal.Decimal, string, string, string, uint, uint) (chaindeposit.Opened, error) {
		return chaindeposit.Opened{Reference: "ref-2"}, nil
	}, nil, nil, nil)
	if _, err := noAddr.OpenPayment(context.Background(), chaindeposit.OpenRequest{AttemptID: "pa"}); err == nil {
		t.Fatal("a payment without a deposit address cannot be paid")
	}
	c := chaindeposit.New(b)
	resp, err := c.Authorize(context.Background(), connectors.AuthorizeRequest{AttemptID: "pa_9", MerchantID: "7", PlatformID: "3", Money: connectors.Money{Amount: decimal.NewFromInt(100), Asset: "USD"}, CaptureMethod: connectors.CaptureAutomatic, PaymentMethod: usdc()})
	if err != nil || resp.ConnectorTransactionID == "" || resp.ConnectorTransactionID == "pa_9" || resp.NextAction.Address == "" {
		t.Fatalf("authorize through payminto = %+v, %v", resp, err)
	}
	sync, err := c.Sync(context.Background(), connectors.SyncRequest{AttemptID: "pa_9"})
	if err != nil || sync.ConnectorTransactionID != resp.ConnectorTransactionID {
		t.Fatalf("sync by attempt through payminto = %+v, %v", sync, err)
	}
}
