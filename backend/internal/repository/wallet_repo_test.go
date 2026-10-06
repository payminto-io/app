package repository

import (
	"fmt"
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newTestDB creates an in-memory SQLite database with all required tables
// auto-migrated. Using SQLite avoids any external dependencies in CI.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	err = db.AutoMigrate(
		&models.Member{},
		&models.BlockchainFamily{},
		&models.Wallet{},
		&models.AddressPool{},
	)
	if err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}
	return db
}

// seedFamily inserts a BlockchainFamily and returns its id.
func seedFamily(t *testing.T, db *gorm.DB, code string) uint {
	t.Helper()
	bf := models.BlockchainFamily{Name: code, Code: code}
	if err := db.Create(&bf).Error; err != nil {
		t.Fatalf("seed BlockchainFamily: %v", err)
	}
	return bf.ID
}

// seedMember inserts a Member and returns its id.
func seedMember(t *testing.T, db *gorm.DB, name string) uint {
	t.Helper()
	m := models.Member{Name: name, CustomerID: name + "-cid"}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("seed Member: %v", err)
	}
	return m.ID
}

// ─── WalletRepository tests ──────────────────────────────────────────────────

func TestWalletRepo_CreateAndGet(t *testing.T) {
	db := newTestDB(t)
	familyID := seedFamily(t, db, "ETH")
	memberID := seedMember(t, db, "alice")

	repo := NewWalletRepository(db)

	w := &models.Wallet{
		Name:               "Alice ETH",
		Kind:               "hd",
		Status:             "active",
		BlockchainFamilyID: familyID,
		MemberID:           memberID,
	}
	if err := repo.Create(w); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if w.ID == 0 {
		t.Fatal("expected non-zero ID after Create")
	}

	got, err := repo.GetByID(w.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != w.Name {
		t.Errorf("name mismatch: got %q want %q", got.Name, w.Name)
	}
	if got.BlockchainFamily == nil {
		t.Error("expected BlockchainFamily to be preloaded")
	}
	if got.BlockchainFamily.Code != "ETH" {
		t.Errorf("family code mismatch: got %q want ETH", got.BlockchainFamily.Code)
	}
}

func TestWalletRepo_GetByMemberAndFamily(t *testing.T) {
	db := newTestDB(t)
	familyID := seedFamily(t, db, "BTC")
	memberID := seedMember(t, db, "bob")

	repo := NewWalletRepository(db)

	w := &models.Wallet{
		Name:               "Bob BTC",
		Kind:               "hd",
		BlockchainFamilyID: familyID,
		MemberID:           memberID,
	}
	if err := repo.Create(w); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByMemberAndFamily(memberID, familyID)
	if err != nil {
		t.Fatalf("GetByMemberAndFamily: %v", err)
	}
	if got.ID != w.ID {
		t.Errorf("id mismatch: got %d want %d", got.ID, w.ID)
	}

	// Non-existent combination should return error (ErrRecordNotFound).
	_, err = repo.GetByMemberAndFamily(memberID+99, familyID)
	if err == nil {
		t.Error("expected error for non-existent member+family, got nil")
	}
}

func TestWalletRepo_ListByMember(t *testing.T) {
	db := newTestDB(t)
	familyID1 := seedFamily(t, db, "ETH2")
	familyID2 := seedFamily(t, db, "TRX2")
	memberID := seedMember(t, db, "carol")
	otherMember := seedMember(t, db, "dave")

	repo := NewWalletRepository(db)

	wallets := []models.Wallet{
		{Name: "carol-eth", Kind: "hd", BlockchainFamilyID: familyID1, MemberID: memberID},
		{Name: "carol-trx", Kind: "hd", BlockchainFamilyID: familyID2, MemberID: memberID},
		{Name: "dave-eth", Kind: "hd", BlockchainFamilyID: familyID1, MemberID: otherMember},
	}
	for i := range wallets {
		if err := repo.Create(&wallets[i]); err != nil {
			t.Fatalf("Create wallet[%d]: %v", i, err)
		}
	}

	got, err := repo.ListByMember(memberID)
	if err != nil {
		t.Fatalf("ListByMember: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 wallets for carol, got %d", len(got))
	}

	// With limit=1.
	limited, err := repo.ListByMember(memberID, WithLimit(1))
	if err != nil {
		t.Fatalf("ListByMember with limit: %v", err)
	}
	if len(limited) != 1 {
		t.Errorf("expected 1 wallet with limit=1, got %d", len(limited))
	}
}

// ─── AddressPoolRepository tests ─────────────────────────────────────────────

func TestAddressPoolRepo_BulkCreate(t *testing.T) {
	db := newTestDB(t)
	familyID := seedFamily(t, db, "POLY")
	memberID := seedMember(t, db, "bulk-user")

	walletRepo := NewWalletRepository(db)
	w := &models.Wallet{Name: "bulk-wallet", Kind: "hd", BlockchainFamilyID: familyID, MemberID: memberID}
	if err := walletRepo.Create(w); err != nil {
		t.Fatalf("Create wallet: %v", err)
	}

	repo := NewAddressPoolRepository(db)

	var addrs []models.AddressPool
	for i := range 10 {
		addrs = append(addrs, models.AddressPool{
			Address:            fmt.Sprintf("0xbulk%d", i),
			PathIndex:          uint(i),
			Status:             "available",
			WalletID:           w.ID,
			BlockchainFamilyID: familyID,
		})
	}
	if err := repo.BulkCreate(addrs); err != nil {
		t.Fatalf("BulkCreate: %v", err)
	}

	count, err := repo.CountAvailableByWallet(w.ID)
	if err != nil {
		t.Fatalf("CountAvailableByWallet: %v", err)
	}
	if count != 10 {
		t.Errorf("expected 10 available, got %d", count)
	}
}

func TestAddressPoolRepo_GetAvailableByWallet(t *testing.T) {
	db := newTestDB(t)
	familyID := seedFamily(t, db, "BASE")
	memberID := seedMember(t, db, "avail-user")

	walletRepo := NewWalletRepository(db)
	w := &models.Wallet{Name: "avail-wallet", Kind: "hd", BlockchainFamilyID: familyID, MemberID: memberID}
	if err := walletRepo.Create(w); err != nil {
		t.Fatalf("Create wallet: %v", err)
	}

	repo := NewAddressPoolRepository(db)

	// Insert 3 available + 1 used; path indices 5, 2, 8 (shuffled) + used at 0.
	entries := []models.AddressPool{
		{Address: "0xavail5", PathIndex: 5, Status: "available", WalletID: w.ID, BlockchainFamilyID: familyID},
		{Address: "0xavail2", PathIndex: 2, Status: "available", WalletID: w.ID, BlockchainFamilyID: familyID},
		{Address: "0xavail8", PathIndex: 8, Status: "available", WalletID: w.ID, BlockchainFamilyID: familyID},
		{Address: "0xused0", PathIndex: 0, Status: "used", WalletID: w.ID, BlockchainFamilyID: familyID},
	}
	for i := range entries {
		if err := repo.Create(&entries[i]); err != nil {
			t.Fatalf("Create entry[%d]: %v", i, err)
		}
	}

	got, err := repo.GetAvailableByWallet(w.ID)
	if err != nil {
		t.Fatalf("GetAvailableByWallet: %v", err)
	}
	// Should return the address with the lowest path_index among available (index=2).
	if got.PathIndex != 2 {
		t.Errorf("expected PathIndex=2, got %d", got.PathIndex)
	}
	if got.Address != "0xavail2" {
		t.Errorf("expected address 0xavail2, got %s", got.Address)
	}
}

func TestAddressPoolRepo_MarkUsed(t *testing.T) {
	db := newTestDB(t)
	familyID := seedFamily(t, db, "MARK")
	memberID := seedMember(t, db, "mark-user")

	walletRepo := NewWalletRepository(db)
	w := &models.Wallet{Name: "mark-wallet", Kind: "hd", BlockchainFamilyID: familyID, MemberID: memberID}
	if err := walletRepo.Create(w); err != nil {
		t.Fatalf("Create wallet: %v", err)
	}

	repo := NewAddressPoolRepository(db)

	// Two available addresses; we'll mark the first (lower index) as used.
	a1 := &models.AddressPool{Address: "0xmark1", PathIndex: 1, Status: "available", WalletID: w.ID, BlockchainFamilyID: familyID}
	a2 := &models.AddressPool{Address: "0xmark2", PathIndex: 2, Status: "available", WalletID: w.ID, BlockchainFamilyID: familyID}
	for _, a := range []*models.AddressPool{a1, a2} {
		if err := repo.Create(a); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	if err := repo.MarkUsed(a1.ID); err != nil {
		t.Fatalf("MarkUsed: %v", err)
	}

	// GetAvailableByWallet should now skip a1 and return a2.
	got, err := repo.GetAvailableByWallet(w.ID)
	if err != nil {
		t.Fatalf("GetAvailableByWallet after MarkUsed: %v", err)
	}
	if got.ID != a2.ID {
		t.Errorf("expected a2 (id=%d), got id=%d", a2.ID, got.ID)
	}
}

func TestAddressPoolRepo_CountAvailableByWallet(t *testing.T) {
	db := newTestDB(t)
	familyID := seedFamily(t, db, "CNT")
	memberID := seedMember(t, db, "count-user")

	walletRepo := NewWalletRepository(db)
	w := &models.Wallet{Name: "count-wallet", Kind: "hd", BlockchainFamilyID: familyID, MemberID: memberID}
	if err := walletRepo.Create(w); err != nil {
		t.Fatalf("Create wallet: %v", err)
	}

	repo := NewAddressPoolRepository(db)

	// Insert 4 available, 2 used.
	for i := range 4 {
		a := &models.AddressPool{
			Address:            fmt.Sprintf("0xcnt-avail%d", i),
			PathIndex:          uint(i),
			Status:             "available",
			WalletID:           w.ID,
			BlockchainFamilyID: familyID,
		}
		if err := repo.Create(a); err != nil {
			t.Fatalf("Create available: %v", err)
		}
	}
	for i := 4; i < 6; i++ {
		a := &models.AddressPool{
			Address:            fmt.Sprintf("0xcnt-used%d", i),
			PathIndex:          uint(i),
			Status:             "used",
			WalletID:           w.ID,
			BlockchainFamilyID: familyID,
		}
		if err := repo.Create(a); err != nil {
			t.Fatalf("Create used: %v", err)
		}
	}

	count, err := repo.CountAvailableByWallet(w.ID)
	if err != nil {
		t.Fatalf("CountAvailableByWallet: %v", err)
	}
	if count != 4 {
		t.Errorf("expected 4 available, got %d", count)
	}
}

func TestAddressPoolRepo_GetLastPathIndex(t *testing.T) {
	db := newTestDB(t)
	familyID := seedFamily(t, db, "LAST")
	memberID := seedMember(t, db, "last-user")

	walletRepo := NewWalletRepository(db)
	w := &models.Wallet{Name: "last-wallet", Kind: "hd", BlockchainFamilyID: familyID, MemberID: memberID}
	if err := walletRepo.Create(w); err != nil {
		t.Fatalf("Create wallet: %v", err)
	}

	repo := NewAddressPoolRepository(db)

	// Empty wallet should return 0.
	idx, err := repo.GetLastPathIndex(w.ID)
	if err != nil {
		t.Fatalf("GetLastPathIndex (empty): %v", err)
	}
	if idx != 0 {
		t.Errorf("expected 0 for empty wallet, got %d", idx)
	}

	// Insert addresses with path indices 3, 7, 1.
	for _, pi := range []uint{3, 7, 1} {
		a := &models.AddressPool{
			Address:            fmt.Sprintf("0xlast%d", pi),
			PathIndex:          pi,
			Status:             "available",
			WalletID:           w.ID,
			BlockchainFamilyID: familyID,
		}
		if err := repo.Create(a); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	idx, err = repo.GetLastPathIndex(w.ID)
	if err != nil {
		t.Fatalf("GetLastPathIndex: %v", err)
	}
	if idx != 7 {
		t.Errorf("expected 7 (max path_index), got %d", idx)
	}
}
