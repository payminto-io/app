package repository

import (
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupBlockchainDB creates an in-memory SQLite DB with blockchain/currency tables auto-migrated.
// Each call uses a unique DSN so tests are fully isolated from one another.
func setupBlockchainDB(t *testing.T) *gorm.DB {
	t.Helper()
	// Use a unique file URI per test to guarantee isolation.
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared&_fk=1"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	err = db.AutoMigrate(
		&models.BlockchainFamily{},
		&models.Blockchain{},
		&models.Currency{},
		&models.BlockchainCurrency{},
	)
	if err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}
	return db
}

// ─── BlockchainFamily tests ──────────────────────────────────────────────────

func TestBlockchainFamilyRepo_CreateAndGetByCode(t *testing.T) {
	db := setupBlockchainDB(t)
	repo := NewBlockchainFamilyRepository(db)

	family := &models.BlockchainFamily{
		Name: "Ethereum",
		Code: "EVM",
	}
	if err := repo.Create(family); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if family.ID == 0 {
		t.Fatal("expected non-zero ID after Create")
	}

	got, err := repo.GetByCode("EVM")
	if err != nil {
		t.Fatalf("GetByCode: %v", err)
	}
	if got.Name != "Ethereum" {
		t.Errorf("name mismatch: got %q want %q", got.Name, "Ethereum")
	}
	if got.Code != "EVM" {
		t.Errorf("code mismatch: got %q want %q", got.Code, "EVM")
	}

	// GetByID
	byID, err := repo.GetByID(family.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if byID.Code != "EVM" {
		t.Errorf("GetByID code mismatch: got %q", byID.Code)
	}

	// GetByName
	byName, err := repo.GetByName("Ethereum")
	if err != nil {
		t.Fatalf("GetByName: %v", err)
	}
	if byName.ID != family.ID {
		t.Errorf("GetByName ID mismatch: got %d want %d", byName.ID, family.ID)
	}

	// List
	all, err := repo.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("expected 1 family, got %d", len(all))
	}

	// Non-existent code should return error.
	_, err = repo.GetByCode("NONEXISTENT")
	if err == nil {
		t.Error("expected error for non-existent code, got nil")
	}
}

// ─── Blockchain tests ─────────────────────────────────────────────────────────

func TestBlockchainRepo_CreateAndGetByCode_PreloadsFamily(t *testing.T) {
	db := setupBlockchainDB(t)
	familyRepo := NewBlockchainFamilyRepository(db)
	repo := NewBlockchainRepository(db)

	family := &models.BlockchainFamily{Name: "Ethereum", Code: "EVM"}
	if err := familyRepo.Create(family); err != nil {
		t.Fatalf("Create family: %v", err)
	}

	bc := &models.Blockchain{
		Code:               "ETH",
		Name:               "Ethereum Mainnet",
		BlockchainFamilyID: family.ID,
		Status:             "active",
		MinConfirmations:   12,
	}
	if err := repo.Create(bc); err != nil {
		t.Fatalf("Create blockchain: %v", err)
	}
	if bc.ID == 0 {
		t.Fatal("expected non-zero ID after Create")
	}

	got, err := repo.GetByCode("ETH")
	if err != nil {
		t.Fatalf("GetByCode: %v", err)
	}
	if got.Name != "Ethereum Mainnet" {
		t.Errorf("name mismatch: got %q want %q", got.Name, "Ethereum Mainnet")
	}
	if got.BlockchainFamily == nil {
		t.Fatal("expected BlockchainFamily to be preloaded, got nil")
	}
	if got.BlockchainFamily.Code != "EVM" {
		t.Errorf("family code mismatch: got %q want EVM", got.BlockchainFamily.Code)
	}

	// GetByID also preloads
	byID, err := repo.GetByID(bc.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if byID.BlockchainFamily == nil {
		t.Error("GetByID: expected BlockchainFamily preloaded")
	}
}

func TestBlockchainRepo_ListActive(t *testing.T) {
	db := setupBlockchainDB(t)
	familyRepo := NewBlockchainFamilyRepository(db)
	repo := NewBlockchainRepository(db)

	family := &models.BlockchainFamily{Name: "Multi", Code: "MULTI"}
	if err := familyRepo.Create(family); err != nil {
		t.Fatalf("Create family: %v", err)
	}

	active := &models.Blockchain{
		Code:               "ETH",
		Name:               "Ethereum",
		BlockchainFamilyID: family.ID,
		Status:             "active",
	}
	inactive := &models.Blockchain{
		Code:               "BSC",
		Name:               "BNB Chain",
		BlockchainFamilyID: family.ID,
		Status:             "inactive",
	}
	if err := repo.Create(active); err != nil {
		t.Fatalf("Create active: %v", err)
	}
	if err := repo.Create(inactive); err != nil {
		t.Fatalf("Create inactive: %v", err)
	}

	got, err := repo.ListActive()
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 active blockchain, got %d", len(got))
	}
	if got[0].Code != "ETH" {
		t.Errorf("expected ETH, got %q", got[0].Code)
	}
	if got[0].BlockchainFamily == nil {
		t.Error("expected BlockchainFamily to be preloaded")
	}
}

func TestBlockchainRepo_UpdateHeight(t *testing.T) {
	db := setupBlockchainDB(t)
	familyRepo := NewBlockchainFamilyRepository(db)
	repo := NewBlockchainRepository(db)

	family := &models.BlockchainFamily{Name: "Bitcoin", Code: "BTC"}
	if err := familyRepo.Create(family); err != nil {
		t.Fatalf("Create family: %v", err)
	}

	bc := &models.Blockchain{
		Code:               "BTC",
		Name:               "Bitcoin",
		BlockchainFamilyID: family.ID,
		Status:             "active",
		Height:             0,
	}
	if err := repo.Create(bc); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.UpdateHeight(bc.ID, 800000); err != nil {
		t.Fatalf("UpdateHeight: %v", err)
	}

	updated, err := repo.GetByID(bc.ID)
	if err != nil {
		t.Fatalf("GetByID after UpdateHeight: %v", err)
	}
	if updated.Height != 800000 {
		t.Errorf("height mismatch: got %d want 800000", updated.Height)
	}
}

// ─── Currency tests ───────────────────────────────────────────────────────────

func TestCurrencyRepo_CreateAndGetByCode(t *testing.T) {
	db := setupBlockchainDB(t)
	repo := NewCurrencyRepository(db)

	price := decimal.NewFromFloat(3500.00)
	c := &models.Currency{
		Name:    "Ethereum",
		Code:    "ETH",
		Type:    "crypto",
		Visible: true,
		Price:   &price,
	}
	if err := repo.Create(c); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if c.ID == 0 {
		t.Fatal("expected non-zero ID after Create")
	}

	got, err := repo.GetByCode("ETH")
	if err != nil {
		t.Fatalf("GetByCode: %v", err)
	}
	if got.Name != "Ethereum" {
		t.Errorf("name mismatch: got %q want %q", got.Name, "Ethereum")
	}
	if got.Price == nil || !got.Price.Equal(price) {
		t.Errorf("price mismatch: got %v want %v", got.Price, price)
	}

	// GetByID
	byID, err := repo.GetByID(c.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if byID.Code != "ETH" {
		t.Errorf("GetByID code mismatch: got %q", byID.Code)
	}

	// Add invisible currency by inserting with raw SQL so SQLite default is bypassed.
	// GORM skips false bool fields when the column has gorm:"default:true", so we
	// directly set the field after creation using a raw update.
	invisible := &models.Currency{Name: "Hidden Coin", Code: "HID", Type: "crypto", Visible: true}
	if err := repo.Create(invisible); err != nil {
		t.Fatalf("Create invisible: %v", err)
	}
	// Force visible=false with a raw update to bypass the GORM default handling.
	if err := db.Exec("UPDATE currencies SET visible = false WHERE code = ?", "HID").Error; err != nil {
		t.Fatalf("force visible=false: %v", err)
	}

	visible, err := repo.ListVisible()
	if err != nil {
		t.Fatalf("ListVisible: %v", err)
	}
	// Verify ETH is present and HID is absent.
	foundETH := false
	foundHID := false
	for _, v := range visible {
		if v.Code == "ETH" {
			foundETH = true
		}
		if v.Code == "HID" {
			foundHID = true
		}
	}
	if !foundETH {
		t.Error("expected ETH in visible list, not found")
	}
	if foundHID {
		t.Error("expected HID to be excluded from visible list")
	}

	// List with no opts returns all.
	all, err := repo.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) < 2 {
		t.Errorf("expected at least 2 currencies total, got %d", len(all))
	}

	// Non-existent code.
	_, err = repo.GetByCode("NONEXISTENT")
	if err == nil {
		t.Error("expected error for non-existent code, got nil")
	}
}

func TestCurrencyRepo_UpdatePrice(t *testing.T) {
	db := setupBlockchainDB(t)
	repo := NewCurrencyRepository(db)

	initialPrice := decimal.NewFromFloat(100.0)
	c := &models.Currency{
		Name:    "Tether",
		Code:    "USDT",
		Type:    "stablecoin",
		Visible: true,
		Price:   &initialPrice,
	}
	if err := repo.Create(c); err != nil {
		t.Fatalf("Create: %v", err)
	}

	newPrice := decimal.NewFromFloat(1.001)
	if err := repo.UpdatePrice("USDT", newPrice); err != nil {
		t.Fatalf("UpdatePrice: %v", err)
	}

	got, err := repo.GetByCode("USDT")
	if err != nil {
		t.Fatalf("GetByCode after UpdatePrice: %v", err)
	}
	if got.Price == nil {
		t.Fatal("expected price to be set after UpdatePrice, got nil")
	}
	if !got.Price.Equal(newPrice) {
		t.Errorf("price mismatch: got %v want %v", got.Price, newPrice)
	}
}

// ─── BlockchainCurrency tests ─────────────────────────────────────────────────

func seedBlockchainAndCurrency(t *testing.T, db *gorm.DB) (blockchain *models.Blockchain, currency *models.Currency) {
	t.Helper()

	familyRepo := NewBlockchainFamilyRepository(db)
	family := &models.BlockchainFamily{Name: "EVM Family", Code: "EVMF"}
	if err := familyRepo.Create(family); err != nil {
		t.Fatalf("seed family: %v", err)
	}

	bcRepo := NewBlockchainRepository(db)
	blockchain = &models.Blockchain{
		Code:               "ETH",
		Name:               "Ethereum",
		BlockchainFamilyID: family.ID,
		Status:             "active",
	}
	if err := bcRepo.Create(blockchain); err != nil {
		t.Fatalf("seed blockchain: %v", err)
	}

	currRepo := NewCurrencyRepository(db)
	currency = &models.Currency{
		Name:    "Ethereum",
		Code:    "ETH",
		Type:    "crypto",
		Visible: true,
	}
	if err := currRepo.Create(currency); err != nil {
		t.Fatalf("seed currency: %v", err)
	}

	return blockchain, currency
}

func TestBlockchainCurrencyRepo_GetByBlockchainAndCurrency(t *testing.T) {
	db := setupBlockchainDB(t)
	blockchain, currency := seedBlockchainAndCurrency(t, db)
	repo := NewBlockchainCurrencyRepository(db)

	depositFee := decimal.NewFromFloat(0.001)
	bc := &models.BlockchainCurrency{
		BlockchainID:   blockchain.ID,
		CurrencyID:     currency.ID,
		BlockchainCode: blockchain.Code,
		CurrencyCode:   currency.Code,
		DepositEnabled: true,
		Visible:        true,
		DepositFee:     &depositFee,
	}
	if err := repo.Create(bc); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if bc.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	got, err := repo.GetByBlockchainAndCurrency(blockchain.ID, currency.ID)
	if err != nil {
		t.Fatalf("GetByBlockchainAndCurrency: %v", err)
	}
	if got.ID != bc.ID {
		t.Errorf("ID mismatch: got %d want %d", got.ID, bc.ID)
	}
	if got.Currency == nil {
		t.Error("expected Currency to be preloaded")
	}
	if got.Blockchain == nil {
		t.Error("expected Blockchain to be preloaded")
	}
	if got.Currency.Code != "ETH" {
		t.Errorf("currency code mismatch: got %q", got.Currency.Code)
	}

	// Non-existent pair.
	_, err = repo.GetByBlockchainAndCurrency(9999, 9999)
	if err == nil {
		t.Error("expected error for non-existent pair, got nil")
	}
}

func TestBlockchainCurrencyRepo_GetByBlockchainCodeAndCurrencyCode(t *testing.T) {
	db := setupBlockchainDB(t)
	blockchain, currency := seedBlockchainAndCurrency(t, db)
	repo := NewBlockchainCurrencyRepository(db)

	bc := &models.BlockchainCurrency{
		BlockchainID:   blockchain.ID,
		CurrencyID:     currency.ID,
		BlockchainCode: blockchain.Code,
		CurrencyCode:   currency.Code,
		DepositEnabled: true,
		Visible:        true,
	}
	if err := repo.Create(bc); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByBlockchainCodeAndCurrencyCode("ETH", "ETH")
	if err != nil {
		t.Fatalf("GetByBlockchainCodeAndCurrencyCode: %v", err)
	}
	if got.ID != bc.ID {
		t.Errorf("ID mismatch: got %d want %d", got.ID, bc.ID)
	}
	if got.Currency == nil || got.Blockchain == nil {
		t.Error("expected both Currency and Blockchain to be preloaded")
	}

	// Non-existent codes.
	_, err = repo.GetByBlockchainCodeAndCurrencyCode("NOPE", "NOPE")
	if err == nil {
		t.Error("expected error for non-existent codes, got nil")
	}
}

func TestBlockchainCurrencyRepo_ListDepositEnabled(t *testing.T) {
	db := setupBlockchainDB(t)

	// Set up two blockchains with two currencies.
	familyRepo := NewBlockchainFamilyRepository(db)
	family := &models.BlockchainFamily{Name: "Test Family", Code: "TESTF"}
	if err := familyRepo.Create(family); err != nil {
		t.Fatalf("seed family: %v", err)
	}

	bcRepo := NewBlockchainRepository(db)
	ethChain := &models.Blockchain{Code: "ETH2", Name: "Ethereum2", BlockchainFamilyID: family.ID, Status: "active"}
	tronChain := &models.Blockchain{Code: "TRX2", Name: "Tron2", BlockchainFamilyID: family.ID, Status: "active"}
	if err := bcRepo.Create(ethChain); err != nil {
		t.Fatalf("seed ethChain: %v", err)
	}
	if err := bcRepo.Create(tronChain); err != nil {
		t.Fatalf("seed tronChain: %v", err)
	}

	currRepo := NewCurrencyRepository(db)
	ethCurr := &models.Currency{Name: "Ethereum2", Code: "ETH2", Type: "crypto", Visible: true}
	usdtCurr := &models.Currency{Name: "Tether2", Code: "USDT2", Type: "stablecoin", Visible: true}
	if err := currRepo.Create(ethCurr); err != nil {
		t.Fatalf("seed ethCurr: %v", err)
	}
	if err := currRepo.Create(usdtCurr); err != nil {
		t.Fatalf("seed usdtCurr: %v", err)
	}

	repo := NewBlockchainCurrencyRepository(db)

	// deposit_enabled=true, visible=true → should appear
	bc1 := &models.BlockchainCurrency{
		BlockchainID:   ethChain.ID,
		CurrencyID:     ethCurr.ID,
		BlockchainCode: ethChain.Code,
		CurrencyCode:   ethCurr.Code,
		DepositEnabled: true,
		Visible:        true,
	}
	// deposit_enabled=false → should NOT appear
	bc2 := &models.BlockchainCurrency{
		BlockchainID:   tronChain.ID,
		CurrencyID:     usdtCurr.ID,
		BlockchainCode: tronChain.Code,
		CurrencyCode:   usdtCurr.Code,
		DepositEnabled: false,
		Visible:        true,
	}
	// deposit_enabled=true, visible=false → should NOT appear
	bc3 := &models.BlockchainCurrency{
		BlockchainID:   ethChain.ID,
		CurrencyID:     usdtCurr.ID,
		BlockchainCode: ethChain.Code,
		CurrencyCode:   usdtCurr.Code,
		DepositEnabled: true,
		Visible:        false,
	}

	for _, bc := range []*models.BlockchainCurrency{bc1, bc2, bc3} {
		if err := repo.Create(bc); err != nil {
			t.Fatalf("Create bc: %v", err)
		}
	}
	// Force deposit_enabled=false and visible=false via raw SQL since GORM skips false
	// bools on columns that have gorm:"default:true".
	if err := db.Exec("UPDATE blockchain_currencies SET deposit_enabled = false WHERE id = ?", bc2.ID).Error; err != nil {
		t.Fatalf("force deposit_enabled=false: %v", err)
	}
	if err := db.Exec("UPDATE blockchain_currencies SET visible = false WHERE id = ?", bc3.ID).Error; err != nil {
		t.Fatalf("force visible=false: %v", err)
	}

	got, err := repo.ListDepositEnabled()
	if err != nil {
		t.Fatalf("ListDepositEnabled: %v", err)
	}
	// Verify bc1 is present; bc2 and bc3 are absent.
	foundBC1 := false
	foundBC2 := false
	foundBC3 := false
	for _, g := range got {
		switch g.ID {
		case bc1.ID:
			foundBC1 = true
		case bc2.ID:
			foundBC2 = true
		case bc3.ID:
			foundBC3 = true
		}
	}
	if !foundBC1 {
		t.Errorf("expected bc1 (id=%d) in ListDepositEnabled, not found", bc1.ID)
	}
	if foundBC2 {
		t.Errorf("expected bc2 (deposit_enabled=false) to be excluded from ListDepositEnabled")
	}
	if foundBC3 {
		t.Errorf("expected bc3 (visible=false) to be excluded from ListDepositEnabled")
	}
	if len(got) > 0 && (got[0].Currency == nil || got[0].Blockchain == nil) {
		t.Error("expected both Currency and Blockchain preloaded in ListDepositEnabled")
	}
}

func TestBlockchainCurrencyRepo_CreateOrUpdate(t *testing.T) {
	db := setupBlockchainDB(t)
	blockchain, currency := seedBlockchainAndCurrency(t, db)
	repo := NewBlockchainCurrencyRepository(db)

	initialFee := decimal.NewFromFloat(0.001)
	bc := &models.BlockchainCurrency{
		BlockchainID:   blockchain.ID,
		CurrencyID:     currency.ID,
		BlockchainCode: blockchain.Code,
		CurrencyCode:   currency.Code,
		DepositEnabled: true,
		Visible:        true,
		DepositFee:     &initialFee,
	}

	// Initial create via CreateOrUpdate.
	if err := repo.CreateOrUpdate(bc); err != nil {
		t.Fatalf("CreateOrUpdate (insert): %v", err)
	}
	if bc.ID == 0 {
		t.Fatal("expected non-zero ID after initial CreateOrUpdate")
	}

	// Now upsert with a new deposit fee.
	updatedFee := decimal.NewFromFloat(0.005)
	bc2 := &models.BlockchainCurrency{
		BlockchainID:   blockchain.ID,
		CurrencyID:     currency.ID,
		BlockchainCode: blockchain.Code,
		CurrencyCode:   currency.Code,
		DepositEnabled: true,
		Visible:        true,
		DepositFee:     &updatedFee,
	}
	if err := repo.CreateOrUpdate(bc2); err != nil {
		t.Fatalf("CreateOrUpdate (upsert): %v", err)
	}

	// Verify only one record exists (not duplicated) and fee is updated.
	got, err := repo.GetByBlockchainAndCurrency(blockchain.ID, currency.ID)
	if err != nil {
		t.Fatalf("GetByBlockchainAndCurrency after upsert: %v", err)
	}
	if got.DepositFee == nil || !got.DepositFee.Equal(updatedFee) {
		t.Errorf("fee not updated: got %v want %v", got.DepositFee, updatedFee)
	}

	// Confirm no duplicates.
	var count int64
	db.Model(&models.BlockchainCurrency{}).
		Where("blockchain_id = ? AND currency_id = ?", blockchain.ID, currency.ID).
		Count(&count)
	if count != 1 {
		t.Errorf("expected 1 record after upsert, got %d", count)
	}
}
