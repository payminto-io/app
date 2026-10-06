package repository

import (
	"fmt"
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupWalletExtrasDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	err = db.AutoMigrate(
		&models.Member{},
		&models.BlockchainFamily{},
		&models.Blockchain{},
		&models.Currency{},
		&models.BlockchainCurrency{},
		&models.Wallet{},
		&models.WalletXpub{},
		&models.WalletSCW{},
		&models.WalletFunction{},
		&models.AddressPool{},
		&models.AccountAddress{},
		&models.DepositAddress{},
		&models.Deposit{},
	)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestWalletXpubRepo_CreateAndGet(t *testing.T) {
	repo := NewWalletXpubRepository(setupWalletExtrasDB(t))
	x := &models.WalletXpub{
		WalletID:   1,
		Xpub:       "xpub6CUGRUonZSQ...",
		Path:       "m/44'/60'/0'",
		AccountIdx: 0,
		NextIdx:    0,
	}
	if err := repo.Create(x); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetByWalletAndAccount(1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != "m/44'/60'/0'" {
		t.Errorf("wrong path: %s", got.Path)
	}
}

func TestWalletXpubRepo_IncrementNextIdx(t *testing.T) {
	repo := NewWalletXpubRepository(setupWalletExtrasDB(t))
	x := &models.WalletXpub{WalletID: 1, Xpub: "xpub", Path: "m", AccountIdx: 0, NextIdx: 5}
	_ = repo.Create(x)

	if err := repo.IncrementNextIdx(x.ID); err != nil {
		t.Fatal(err)
	}

	got, _ := repo.GetByID(x.ID)
	if got.NextIdx != 6 {
		t.Errorf("expected next_idx=6, got %d", got.NextIdx)
	}
}

func TestWalletSCWRepo_PendingAndDeployed(t *testing.T) {
	repo := NewWalletSCWRepository(setupWalletExtrasDB(t))
	_ = repo.Create(&models.WalletSCW{
		WalletID:       1,
		BlockchainID:   1,
		OwnerAddress:   "0xowner",
		FactoryAddress: "0xfactory",
		Salt:           "salt1",
		Status:         "pending",
	})

	pending, err := repo.ListPending(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 {
		t.Errorf("expected 1 pending scw, got %d", len(pending))
	}

	if err := repo.MarkDeployed(pending[0].ID, "0xcontract", "0xtxhash"); err != nil {
		t.Fatal(err)
	}

	pending2, _ := repo.ListPending(1)
	if len(pending2) != 0 {
		t.Errorf("expected 0 pending after mark deployed, got %d", len(pending2))
	}
}

func TestWalletFunctionRepo_CreateAndList(t *testing.T) {
	repo := NewWalletFunctionRepository(setupWalletExtrasDB(t))
	mid := uint(7)
	_ = repo.Create(&models.WalletFunction{
		WalletID:   1,
		MemberID:   &mid,
		Action:     "generate_address",
		Successful: true,
	})
	_ = repo.Create(&models.WalletFunction{
		WalletID:   1,
		MemberID:   &mid,
		Action:     "sign_transaction",
		Successful: true,
	})

	list, err := repo.ListByWallet(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2, got %d", len(list))
	}
}

func TestAddressRepo_Balances_Empty(t *testing.T) {
	repo := NewAddressRepository(setupWalletExtrasDB(t))
	balances, err := repo.Balances(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(balances) != 0 {
		t.Errorf("expected empty balances, got %d", len(balances))
	}
}

func TestAddressRepo_Balances_Aggregates(t *testing.T) {
	db := setupWalletExtrasDB(t)

	bc := &models.BlockchainCurrency{
		BlockchainID:   1,
		CurrencyID:     1,
		BlockchainCode: "ETH",
		CurrencyCode:   "ETH",
	}
	db.Create(bc)

	amount := decimal.NewFromFloat(1.5)
	db.Create(&models.AccountAddress{
		Address:              "0xa",
		Balance:              amount,
		BlockchainCurrencyID: bc.ID,
		MemberID:             1,
	})
	db.Create(&models.AccountAddress{
		Address:              "0xb",
		Balance:              decimal.NewFromFloat(0.5),
		BlockchainCurrencyID: bc.ID,
		MemberID:             1,
	})

	repo := NewAddressRepository(db)
	balances, err := repo.Balances(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(balances) != 1 {
		t.Fatalf("expected 1 balance row, got %d", len(balances))
	}
	want := decimal.NewFromFloat(2.0)
	if !balances[0].Balance.Equal(want) {
		t.Errorf("expected 2.0, got %s", balances[0].Balance.String())
	}
}

func TestAddressRepo_EligibleToSweep(t *testing.T) {
	db := setupWalletExtrasDB(t)

	bc := &models.BlockchainCurrency{BlockchainID: 1, CurrencyID: 1, BlockchainCode: "ETH", CurrencyCode: "ETH"}
	db.Create(bc)

	// Used pool entry with enough balance
	db.Create(&models.AddressPool{Address: "0xfull", WalletID: 1, BlockchainFamilyID: 1, Status: "used"})
	db.Create(&models.AccountAddress{
		Address:              "0xfull",
		Balance:              decimal.NewFromFloat(1.0),
		BlockchainCurrencyID: bc.ID,
		MemberID:             1,
	})

	// Used pool entry below threshold
	db.Create(&models.AddressPool{Address: "0xlow", WalletID: 1, BlockchainFamilyID: 1, Status: "used"})
	db.Create(&models.AccountAddress{
		Address:              "0xlow",
		Balance:              decimal.NewFromFloat(0.0005),
		BlockchainCurrencyID: bc.ID,
		MemberID:             1,
	})

	repo := NewAddressRepository(db)
	eligible, err := repo.GetEligibleAddressesToSweep(bc.ID, decimal.NewFromFloat(0.001))
	if err != nil {
		t.Fatal(err)
	}
	if len(eligible) != 1 {
		t.Errorf("expected 1 eligible, got %d", len(eligible))
	}
	if eligible[0].Address != "0xfull" {
		t.Errorf("expected 0xfull, got %s", eligible[0].Address)
	}
}

func TestAddressRepo_ProcessBalance_UsesCallback(t *testing.T) {
	db := setupWalletExtrasDB(t)
	bc := &models.BlockchainCurrency{BlockchainID: 1, CurrencyID: 1, BlockchainCode: "ETH", CurrencyCode: "ETH"}
	db.Create(bc)

	db.Create(&models.AccountAddress{
		Address:              "0xa",
		Balance:              decimal.NewFromFloat(0.5),
		BlockchainCurrencyID: bc.ID,
		MemberID:             1,
	})

	repo := NewAddressRepository(db)
	updated, err := repo.ProcessAddressBalanceAndUpdateStatus(1, func(addr string, bcID uint) (decimal.Decimal, error) {
		return decimal.NewFromFloat(2.0), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated != 1 {
		t.Errorf("expected 1 updated, got %d", updated)
	}

	var aa models.AccountAddress
	db.First(&aa, "address = ?", "0xa")
	if !aa.Balance.Equal(decimal.NewFromFloat(2.0)) {
		t.Errorf("expected balance 2.0 after update, got %s", aa.Balance.String())
	}
}

func TestAddressRepo_MarkLockedUnlocked(t *testing.T) {
	db := setupWalletExtrasDB(t)
	db.Create(&models.AddressPool{Address: "0xabc", WalletID: 1, BlockchainFamilyID: 1, Status: "available"})

	var pool models.AddressPool
	db.First(&pool, "address = ?", "0xabc")

	repo := NewAddressRepository(db)
	if err := repo.MarkLocked(pool.ID, 300); err != nil {
		t.Fatal(err)
	}

	var after models.AddressPool
	db.First(&after, pool.ID)
	if after.Status != "locked" {
		t.Errorf("expected status locked, got %s", after.Status)
	}

	if err := repo.MarkUnlocked(pool.ID); err != nil {
		t.Fatal(err)
	}
	var after2 models.AddressPool
	db.First(&after2, pool.ID)
	if after2.Status != "available" {
		t.Errorf("expected status available, got %s", after2.Status)
	}
}

func TestAddressPoolRepo_ClaimNextAvailable_Atomic(t *testing.T) {
	db := setupWalletExtrasDB(t)
	repo := NewAddressPoolRepository(db)

	// Seed 5 available addresses
	for i := uint(0); i < 5; i++ {
		if err := repo.Create(&models.AddressPool{
			Address:            fmt.Sprintf("0x%d", i),
			WalletID:           1,
			BlockchainFamilyID: 1,
			PathIndex:          i,
			Status:             "available",
		}); err != nil {
			t.Fatal(err)
		}
	}

	// Sequential claims should return distinct rows in path_index order
	seen := map[string]bool{}
	for range 5 {
		a, err := repo.ClaimNextAvailable(1)
		if err != nil {
			t.Fatal(err)
		}
		if seen[a.Address] {
			t.Fatalf("duplicate claim: %s", a.Address)
		}
		seen[a.Address] = true
		if a.Status != "used" {
			t.Errorf("expected used, got %s", a.Status)
		}
	}

	// Sixth claim should fail (pool exhausted)
	if _, err := repo.ClaimNextAvailable(1); err == nil {
		t.Error("expected error when pool exhausted")
	}
}

func TestWalletXpubRepo_ClaimNextIdx_Sequential(t *testing.T) {
	db := setupWalletExtrasDB(t)
	repo := NewWalletXpubRepository(db)

	x := &models.WalletXpub{WalletID: 1, Xpub: "xpub", Path: "m", AccountIdx: 0, NextIdx: 0}
	if err := repo.Create(x); err != nil {
		t.Fatal(err)
	}

	var claimed []uint
	for range 5 {
		idx, err := repo.ClaimNextIdx(x.ID)
		if err != nil {
			t.Fatal(err)
		}
		claimed = append(claimed, idx)
	}

	// Should be 0, 1, 2, 3, 4
	for i, v := range claimed {
		if v != uint(i) {
			t.Errorf("claim %d: got %d, want %d", i, v, i)
		}
	}

	// next_idx should now be 5
	got, err := repo.GetByID(x.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.NextIdx != 5 {
		t.Errorf("expected next_idx=5 after 5 claims, got %d", got.NextIdx)
	}
}
