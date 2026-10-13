package service

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/payminto/payminto/backend/internal/blockchain/tempo"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeTempoReader struct {
	head      uint64
	transfers []tempo.MemoTransfer
	status    map[common.Hash]uint64
	statusErr error
	chainID   *big.Int
	ranges    [][2]uint64
}

func (f *fakeTempoReader) Head(context.Context) (uint64, error) { return f.head, nil }

func (f *fakeTempoReader) ChainID(context.Context) (*big.Int, error) {
	if f.chainID != nil {
		return f.chainID, nil
	}
	return big.NewInt(tempo.ModeratoChainID), nil
}

func (f *fakeTempoReader) MemoTransfersTo(_ context.Context, _ []common.Address, to common.Address, from, until uint64) ([]tempo.MemoTransfer, error) {
	f.ranges = append(f.ranges, [2]uint64{from, until})
	var out []tempo.MemoTransfer
	for _, t := range f.transfers {
		if t.To == to && t.BlockNumber >= from && t.BlockNumber <= until {
			out = append(out, t)
		}
	}
	return out, nil
}

func (f *fakeTempoReader) ReceiptStatus(_ context.Context, h common.Hash) (uint64, error) {
	if f.statusErr != nil {
		return 0, f.statusErr
	}
	if s, ok := f.status[h]; ok {
		return s, nil
	}
	return 1, nil
}

var (
	tempoMerchant = common.HexToAddress("0x82c70b91731914e7b678a232b366336Af1436455")
	tempoPayer    = common.HexToAddress("0x3AB22D906d5Ee0e17C44b60F45DCb56aC6371993")
)

func newTempoFixture(t *testing.T, reader *fakeTempoReader) (*TempoMemoService, *gorm.DB, *models.PaymentRequest) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Blockchain{}, &models.BlockchainCurrency{}, &models.Currency{}, &models.PaymentRequest{}, &models.DepositAddress{}, &models.Deposit{}, &models.TempoMemoCredit{}); err != nil {
		t.Fatal(err)
	}
	chainID := int64(tempo.ModeratoChainID)
	chain := &models.Blockchain{Code: tempo.ChainCode, Name: "Tempo Moderato", MinConfirmations: 1, ChainID: &chainID, Status: "active"}
	if err := db.Create(chain).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Currency{Code: "ALPHAUSD", Name: "Alpha USD", Type: "token", DepositEnabled: true, WalletPrecision: 6}).Error; err != nil {
		t.Fatal(err)
	}
	bc := models.BlockchainCurrency{Address: tempo.AlphaUSD.Hex(), Standard: "TIP20", CurrencyCode: "ALPHAUSD", BlockchainCode: tempo.ChainCode, DepositEnabled: true, WalletPrecision: 6, BlockchainID: chain.ID, CurrencyID: 1}
	if err := db.Create(&bc).Error; err != nil {
		t.Fatal(err)
	}
	p := &models.PaymentRequest{ReferenceID: "6f1c2a3b-4d5e-4f60-8a7b-9c0d1e2f3a4b", AmountInUSD: decimal.NewFromInt(2), State: models.PaymentStateOpen, MemberID: 5, ExternalPlatformID: 1}
	if err := db.Create(p).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.DepositAddress{Address: tempoMerchant.Hex(), BlockchainCurrencyID: bc.ID, MemberID: p.MemberID, PaymentRequestID: &p.ID}).Error; err != nil {
		t.Fatal(err)
	}
	depSvc := NewDepositService(repository.NewDepositRepository(db), repository.NewDepositAddressRepository(db), repository.NewPaymentRepository(db),
		repository.NewBlockchainCurrencyRepository(db), repository.NewBlockchainRepository(db))
	svc := NewTempoMemoService(db, reader, chain, TempoTokensFromCurrencies([]models.BlockchainCurrency{bc}), depSvc, nil, nil,
		TempoMemoConfig{Merchant: tempoMerchant, StartBlock: 100, MaxRange: 50})
	return svc, db, p
}

func memoTransfer(ref string, token common.Address, to common.Address, amount int64, tx string, idx uint, block uint64) tempo.MemoTransfer {
	m, _ := tempo.EncodeMemo(ref)
	return tempo.MemoTransfer{Token: token, From: tempoPayer, To: to, Amount: big.NewInt(amount), Memo: m,
		TxHash: common.HexToHash(tx), LogIndex: idx, BlockNumber: block}
}

func TestTempoPollCreditsMatchingPayment(t *testing.T) {
	reader := &fakeTempoReader{head: 180}
	svc, db, p := newTempoFixture(t, reader)
	reader.transfers = []tempo.MemoTransfer{
		memoTransfer(p.ReferenceID, tempo.AlphaUSD, tempoMerchant, 2_000_000, "0xa1", 3, 120),
		memoTransfer("not-a-payment", tempo.AlphaUSD, tempoMerchant, 1_000_000, "0xa2", 0, 130),
		memoTransfer(p.ReferenceID, tempo.PathUSD, tempoMerchant, 2_000_000, "0xa3", 0, 140), // token not accepted
	}
	reader.status = map[common.Hash]uint64{}
	n, err := svc.PollOnce(context.Background())
	if err != nil || n != 1 {
		t.Fatalf("PollOnce = %d, %v", n, err)
	}
	var deps []models.Deposit
	db.Find(&deps)
	if len(deps) != 1 || deps[0].Status != models.DepositStatusConfirmed || !deps[0].Amount.Equal(decimal.NewFromInt(2)) ||
		deps[0].PaymentRequestID == nil || *deps[0].PaymentRequestID != p.ID || deps[0].MemberID != 5 {
		t.Fatalf("deposits = %+v", deps)
	}
	var credits []models.TempoMemoCredit
	db.Order("block_number").Find(&credits)
	if len(credits) != 3 || credits[0].Outcome != TempoOutcomeCredited || credits[1].Outcome != TempoOutcomeUnknownReference || credits[2].Outcome != string(tempo.RejectUnknownToken) {
		t.Fatalf("credits = %+v", credits)
	}
	var height int64
	db.Model(&models.Blockchain{}).Where("id = ?", svc.chain.ID).Pluck("height", &height)
	if height != 180 {
		t.Fatalf("cursor = %d", height)
	}
	// Ranges are capped at MaxRange and start at StartBlock.
	if reader.ranges[0] != [2]uint64{100, 149} || reader.ranges[1] != [2]uint64{150, 180} {
		t.Fatalf("ranges = %v", reader.ranges)
	}
}

func TestTempoPollIsIdempotentOnRescan(t *testing.T) {
	reader := &fakeTempoReader{head: 120}
	svc, db, p := newTempoFixture(t, reader)
	reader.transfers = []tempo.MemoTransfer{memoTransfer(p.ReferenceID, tempo.AlphaUSD, tempoMerchant, 2_000_000, "0xb1", 1, 110)}
	if n, err := svc.PollOnce(context.Background()); err != nil || n != 1 {
		t.Fatalf("first poll = %d, %v", n, err)
	}
	// Force a rescan of the same range: the (tx hash, log index) key must stop a second credit.
	db.Model(&models.Blockchain{}).Where("id = ?", svc.chain.ID).Update("height", 0)
	if n, err := svc.PollOnce(context.Background()); err != nil || n != 0 {
		t.Fatalf("rescan = %d, %v", n, err)
	}
	var count int64
	db.Model(&models.Deposit{}).Count(&count)
	if count != 1 {
		t.Fatalf("deposits = %d", count)
	}
	// A second log in the same transaction is a separate credit.
	reader.transfers = append(reader.transfers, memoTransfer(p.ReferenceID, tempo.AlphaUSD, tempoMerchant, 1_000_000, "0xb1", 2, 110))
	db.Model(&models.Blockchain{}).Where("id = ?", svc.chain.ID).Update("height", 0)
	if n, err := svc.PollOnce(context.Background()); err != nil || n != 1 {
		t.Fatalf("second log = %d, %v", n, err)
	}
}

func TestTempoPollRejectsRevertedAndHoldsCursorOnRPCError(t *testing.T) {
	reader := &fakeTempoReader{head: 120}
	svc, db, p := newTempoFixture(t, reader)
	tr := memoTransfer(p.ReferenceID, tempo.AlphaUSD, tempoMerchant, 2_000_000, "0xc1", 0, 110)
	reader.transfers = []tempo.MemoTransfer{tr}

	reader.statusErr = errors.New("rpc down")
	if _, err := svc.PollOnce(context.Background()); err == nil {
		t.Fatal("expected error")
	}
	var height int64
	db.Model(&models.Blockchain{}).Where("id = ?", svc.chain.ID).Pluck("height", &height)
	if height >= 110 {
		t.Fatalf("cursor advanced past failing log: %d", height)
	}

	reader.statusErr = nil
	reader.status = map[common.Hash]uint64{tr.TxHash: 0}
	if n, err := svc.PollOnce(context.Background()); err != nil || n != 0 {
		t.Fatalf("reverted poll = %d, %v", n, err)
	}
	var c models.TempoMemoCredit
	db.First(&c)
	if c.Outcome != string(tempo.RejectTxFailed) || c.DepositID != nil {
		t.Fatalf("credit = %+v", c)
	}
}
