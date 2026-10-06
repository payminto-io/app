package service

import (
	"context"
	"math/big"
	"testing"

	"github.com/payminto/payminto/backend/internal/blockchain"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAddressServiceDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.Member{},
		&models.BlockchainFamily{},
		&models.Blockchain{},
		&models.Currency{},
		&models.BlockchainCurrency{},
		&models.Wallet{},
		&models.AddressPool{},
		&models.AccountAddress{},
	); err != nil {
		t.Fatal(err)
	}
	return db
}

func newAddressServiceForTest(t *testing.T, db *gorm.DB) *AddressService {
	t.Helper()
	return NewAddressService(
		repository.NewAddressRepository(db),
		repository.NewBlockchainCurrencyRepository(db),
	)
}

// fakeAdapter is a ChainAdapter stub that returns a canned balance.
type fakeAdapter struct {
	balance *big.Int
	calls   int
}

func (f *fakeAdapter) Name() string                           { return "fake" }
func (f *fakeAdapter) Code() string                           { return "FAKE" }
func (f *fakeAdapter) IsMainnet() bool                        { return false }
func (f *fakeAdapter) GenerateAddress(uint32) (string, error) { return "", nil }
func (f *fakeAdapter) GetBalance(_ context.Context, _, _ string) (*big.Int, error) {
	f.calls++
	if f.balance == nil {
		return big.NewInt(0), nil
	}
	return new(big.Int).Set(f.balance), nil
}
func (f *fakeAdapter) GetConfirmations(context.Context, string) (uint64, error) { return 0, nil }
func (f *fakeAdapter) MonitorBlocks(context.Context, uint64) (<-chan blockchain.Transaction, error) {
	ch := make(chan blockchain.Transaction)
	close(ch)
	return ch, nil
}
func (f *fakeAdapter) BroadcastTransaction(context.Context, []byte) (string, error) {
	return "", nil
}
func (f *fakeAdapter) EstimateGas(context.Context, blockchain.TxParams) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (f *fakeAdapter) ParseBlock(_ context.Context, _ uint64, _ map[string]blockchain.WatchedAddressInfo) ([]blockchain.Transaction, error) {
	return []blockchain.Transaction{}, nil
}
func (f *fakeAdapter) LatestBlock(_ context.Context) (uint64, error) {
	return 0, nil
}

var _ blockchain.ChainAdapter = (*fakeAdapter)(nil)

func TestAddressService_Balances_Empty(t *testing.T) {
	db := setupAddressServiceDB(t)
	svc := newAddressServiceForTest(t, db)

	balances, err := svc.Balances(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(balances) != 0 {
		t.Errorf("expected empty, got %d", len(balances))
	}
}

func TestAddressService_Balances_Aggregated(t *testing.T) {
	db := setupAddressServiceDB(t)
	svc := newAddressServiceForTest(t, db)

	bc := &models.BlockchainCurrency{BlockchainID: 1, CurrencyID: 1, BlockchainCode: "ETH", CurrencyCode: "ETH"}
	db.Create(bc)
	db.Create(&models.AccountAddress{Address: "0xa", Balance: decimal.NewFromFloat(1.2), BlockchainCurrencyID: bc.ID, MemberID: 1})
	db.Create(&models.AccountAddress{Address: "0xb", Balance: decimal.NewFromFloat(0.8), BlockchainCurrencyID: bc.ID, MemberID: 1})

	balances, err := svc.Balances(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(balances) != 1 {
		t.Fatalf("expected 1 row, got %d", len(balances))
	}
	if !balances[0].Balance.Equal(decimal.NewFromFloat(2.0)) {
		t.Errorf("expected 2.0, got %s", balances[0].Balance)
	}
}

func TestAddressService_GetEligibleAddressesToSweep_NegativeThreshold(t *testing.T) {
	db := setupAddressServiceDB(t)
	svc := newAddressServiceForTest(t, db)

	_, err := svc.GetEligibleAddressesToSweep(1, decimal.NewFromFloat(-1))
	if err == nil {
		t.Error("expected error for negative threshold")
	}
}

func TestAddressService_GetEligibleAddressesToSweep_Filters(t *testing.T) {
	db := setupAddressServiceDB(t)
	svc := newAddressServiceForTest(t, db)

	bc := &models.BlockchainCurrency{BlockchainID: 1, CurrencyID: 1, BlockchainCode: "ETH", CurrencyCode: "ETH"}
	db.Create(bc)

	db.Create(&models.AddressPool{Address: "0xhot", WalletID: 1, BlockchainFamilyID: 1, Status: "used"})
	db.Create(&models.AccountAddress{Address: "0xhot", Balance: decimal.NewFromFloat(0.5), BlockchainCurrencyID: bc.ID, MemberID: 1})

	db.Create(&models.AddressPool{Address: "0xcold", WalletID: 1, BlockchainFamilyID: 1, Status: "used"})
	db.Create(&models.AccountAddress{Address: "0xcold", Balance: decimal.NewFromFloat(0.001), BlockchainCurrencyID: bc.ID, MemberID: 1})

	eligible, err := svc.GetEligibleAddressesToSweep(bc.ID, decimal.NewFromFloat(0.1))
	if err != nil {
		t.Fatal(err)
	}
	if len(eligible) != 1 {
		t.Errorf("expected 1 eligible, got %d", len(eligible))
	}
}

func TestAddressService_RefreshBalancesForChain(t *testing.T) {
	db := setupAddressServiceDB(t)
	svc := newAddressServiceForTest(t, db)

	bcf := &models.BlockchainFamily{Code: "evm", Name: "EVM"}
	db.Create(bcf)
	chain := &models.Blockchain{Code: "ETH", Name: "Ethereum", BlockchainFamilyID: bcf.ID, Status: "active"}
	db.Create(chain)
	currency := &models.Currency{Code: "ETH", Name: "Ethereum", Type: "native"}
	db.Create(currency)
	bc := &models.BlockchainCurrency{
		BlockchainID:    chain.ID,
		CurrencyID:      currency.ID,
		BlockchainCode:  "ETH",
		CurrencyCode:    "ETH",
		WalletPrecision: 18,
	}
	db.Create(bc)

	db.Create(&models.AccountAddress{
		Address:              "0xabc",
		Balance:              decimal.NewFromFloat(0.1),
		BlockchainCurrencyID: bc.ID,
		MemberID:             1,
	})

	// 2 ETH in wei = 2 * 1e18
	fake := &fakeAdapter{balance: new(big.Int).Mul(big.NewInt(2), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))}

	updated, err := svc.RefreshBalancesForChain(t.Context(), chain.ID, fake)
	if err != nil {
		t.Fatal(err)
	}
	if updated != 1 {
		t.Errorf("expected 1 updated, got %d", updated)
	}
	if fake.calls != 1 {
		t.Errorf("expected adapter called once, got %d", fake.calls)
	}

	var aa models.AccountAddress
	db.First(&aa, "address = ?", "0xabc")
	if !aa.Balance.Equal(decimal.NewFromFloat(2.0)) {
		t.Errorf("expected 2.0 after refresh, got %s", aa.Balance.String())
	}
}

func TestAddressService_RefreshBalancesForChain_NilAdapter(t *testing.T) {
	db := setupAddressServiceDB(t)
	svc := newAddressServiceForTest(t, db)
	if _, err := svc.RefreshBalancesForChain(t.Context(), 1, nil); err == nil {
		t.Error("expected error for nil adapter")
	}
}

func TestAddressService_MarkLocked_InvalidDuration(t *testing.T) {
	db := setupAddressServiceDB(t)
	svc := newAddressServiceForTest(t, db)
	if err := svc.MarkLocked(1, 0); err == nil {
		t.Error("expected error for zero duration")
	}
}

func TestAddressService_MarkLockedUnlocked(t *testing.T) {
	db := setupAddressServiceDB(t)
	svc := newAddressServiceForTest(t, db)

	db.Create(&models.AddressPool{Address: "0x1", WalletID: 1, BlockchainFamilyID: 1, Status: "available"})
	var pool models.AddressPool
	db.First(&pool, "address = ?", "0x1")

	if err := svc.MarkLocked(pool.ID, 300); err != nil {
		t.Fatal(err)
	}

	var locked models.AddressPool
	db.First(&locked, pool.ID)
	if locked.Status != "locked" {
		t.Errorf("expected locked, got %s", locked.Status)
	}

	if err := svc.MarkUnlocked(pool.ID); err != nil {
		t.Fatal(err)
	}
	var unlocked models.AddressPool
	db.First(&unlocked, pool.ID)
	if unlocked.Status != "available" {
		t.Errorf("expected available, got %s", unlocked.Status)
	}
}
