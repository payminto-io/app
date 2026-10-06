package repository

import (
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newVaultTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	if err := db.AutoMigrate(&models.SecretsVault{}, &models.SecretsVaultActivity{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return db
}

// ─── SecretsVaultRepository tests ────────────────────────────────────────────

func TestSecretsVaultRepo_CreateAndGetByLabel(t *testing.T) {
	db := newVaultTestDB(t)
	repo := NewSecretsVaultRepository(db)

	v := &models.SecretsVault{
		Label:      "hd_wallet.eth.mnemonic",
		Ciphertext: "deadbeef",
		SecretType: models.SecretTypeMnemonic,
	}
	if err := repo.Create(v); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if v.ID == 0 {
		t.Fatal("expected non-zero ID after Create")
	}

	got, err := repo.GetByLabel("hd_wallet.eth.mnemonic")
	if err != nil {
		t.Fatalf("GetByLabel: %v", err)
	}
	if got.Label != v.Label {
		t.Errorf("label mismatch: got %q want %q", got.Label, v.Label)
	}
	if got.SecretType != models.SecretTypeMnemonic {
		t.Errorf("secret type mismatch: got %q", got.SecretType)
	}
}

func TestSecretsVaultRepo_GetByID(t *testing.T) {
	db := newVaultTestDB(t)
	repo := NewSecretsVaultRepository(db)

	v := &models.SecretsVault{Label: "test.key", Ciphertext: "abc123", SecretType: models.SecretTypeAPIKey}
	if err := repo.Create(v); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(v.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ID != v.ID {
		t.Errorf("ID mismatch: got %d want %d", got.ID, v.ID)
	}
}

func TestSecretsVaultRepo_Update(t *testing.T) {
	db := newVaultTestDB(t)
	repo := NewSecretsVaultRepository(db)

	v := &models.SecretsVault{Label: "update.me", Ciphertext: "oldcipher", SecretType: models.SecretTypeOther}
	if err := repo.Create(v); err != nil {
		t.Fatalf("Create: %v", err)
	}

	v.Ciphertext = "newcipher"
	if err := repo.Update(v); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := repo.GetByID(v.ID)
	if got.Ciphertext != "newcipher" {
		t.Errorf("expected updated ciphertext, got %q", got.Ciphertext)
	}
}

func TestSecretsVaultRepo_Delete(t *testing.T) {
	db := newVaultTestDB(t)
	repo := NewSecretsVaultRepository(db)

	v := &models.SecretsVault{Label: "delete.me", Ciphertext: "cipher", SecretType: models.SecretTypeOther}
	if err := repo.Create(v); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(v.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := repo.GetByID(v.ID)
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
}

func TestSecretsVaultRepo_ListByType(t *testing.T) {
	db := newVaultTestDB(t)
	repo := NewSecretsVaultRepository(db)

	entries := []models.SecretsVault{
		{Label: "mnemonic.eth", Ciphertext: "c1", SecretType: models.SecretTypeMnemonic},
		{Label: "mnemonic.btc", Ciphertext: "c2", SecretType: models.SecretTypeMnemonic},
		{Label: "smtp.password", Ciphertext: "c3", SecretType: models.SecretTypeSMTPPassword},
	}
	for i := range entries {
		if err := repo.Create(&entries[i]); err != nil {
			t.Fatalf("Create entry[%d]: %v", i, err)
		}
	}

	mnemonics, err := repo.ListByType(models.SecretTypeMnemonic)
	if err != nil {
		t.Fatalf("ListByType: %v", err)
	}
	if len(mnemonics) != 2 {
		t.Errorf("expected 2 mnemonic entries, got %d", len(mnemonics))
	}

	smtpEntries, err := repo.ListByType(models.SecretTypeSMTPPassword)
	if err != nil {
		t.Fatalf("ListByType smtp: %v", err)
	}
	if len(smtpEntries) != 1 {
		t.Errorf("expected 1 smtp entry, got %d", len(smtpEntries))
	}
}

func TestSecretsVaultRepo_TouchLastUsed(t *testing.T) {
	db := newVaultTestDB(t)
	repo := NewSecretsVaultRepository(db)

	v := &models.SecretsVault{Label: "touch.me", Ciphertext: "cipher", SecretType: models.SecretTypeOther}
	if err := repo.Create(v); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if v.LastUsedAt != nil {
		t.Error("expected nil LastUsedAt before touch")
	}

	if err := repo.TouchLastUsed(v.ID); err != nil {
		t.Fatalf("TouchLastUsed: %v", err)
	}

	got, _ := repo.GetByID(v.ID)
	if got.LastUsedAt == nil {
		t.Error("expected non-nil LastUsedAt after touch")
	}
}

func TestSecretsVaultRepo_List(t *testing.T) {
	db := newVaultTestDB(t)
	repo := NewSecretsVaultRepository(db)

	for i := range 5 {
		v := &models.SecretsVault{
			Label:      "list.key." + string(rune('a'+i)),
			Ciphertext: "cipher",
			SecretType: models.SecretTypeOther,
		}
		if err := repo.Create(v); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	all, err := repo.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 5 {
		t.Errorf("expected 5 entries, got %d", len(all))
	}

	limited, err := repo.List(WithLimit(2))
	if err != nil {
		t.Fatalf("List with limit: %v", err)
	}
	if len(limited) != 2 {
		t.Errorf("expected 2 with limit, got %d", len(limited))
	}
}

// ─── SecretsVaultActivityRepository tests ────────────────────────────────────

func TestSecretsVaultActivityRepo_CreateAndListByLabel(t *testing.T) {
	db := newVaultTestDB(t)
	repo := NewSecretsVaultActivityRepository(db)

	label := "hd_wallet.eth.mnemonic"
	for i := range 3 {
		a := &models.SecretsVaultActivity{
			Label:      label,
			Action:     models.VaultActionStore,
			Successful: true,
		}
		if err := repo.Create(a); err != nil {
			t.Fatalf("Create activity[%d]: %v", i, err)
		}
	}
	// Also create one with a different label
	other := &models.SecretsVaultActivity{Label: "other.key", Action: models.VaultActionRetrieve, Successful: true}
	if err := repo.Create(other); err != nil {
		t.Fatalf("Create other: %v", err)
	}

	activities, err := repo.ListByLabel(label)
	if err != nil {
		t.Fatalf("ListByLabel: %v", err)
	}
	if len(activities) != 3 {
		t.Errorf("expected 3 activities for label, got %d", len(activities))
	}
}

func TestSecretsVaultActivityRepo_ListByVaultID(t *testing.T) {
	db := newVaultTestDB(t)
	vaultRepo := NewSecretsVaultRepository(db)
	actRepo := NewSecretsVaultActivityRepository(db)

	v := &models.SecretsVault{Label: "vault.for.activity", Ciphertext: "c", SecretType: models.SecretTypeOther}
	if err := vaultRepo.Create(v); err != nil {
		t.Fatalf("Create vault: %v", err)
	}

	for i := range 2 {
		a := &models.SecretsVaultActivity{
			VaultID:    &v.ID,
			Label:      v.Label,
			Action:     models.VaultActionRetrieve,
			Successful: true,
		}
		if err := actRepo.Create(a); err != nil {
			t.Fatalf("Create activity[%d]: %v", i, err)
		}
	}

	activities, err := actRepo.ListByVaultID(v.ID)
	if err != nil {
		t.Fatalf("ListByVaultID: %v", err)
	}
	if len(activities) != 2 {
		t.Errorf("expected 2 activities for vault, got %d", len(activities))
	}
}

func TestSecretsVaultActivityRepo_ListByMemberID(t *testing.T) {
	db := newVaultTestDB(t)
	repo := NewSecretsVaultActivityRepository(db)

	memberID := uint(42)
	otherMemberID := uint(99)
	for i := range 4 {
		mid := memberID
		if i%2 == 0 {
			mid = otherMemberID
		}
		a := &models.SecretsVaultActivity{
			Label:      "some.key",
			Action:     models.VaultActionStore,
			MemberID:   &mid,
			Successful: true,
		}
		if err := repo.Create(a); err != nil {
			t.Fatalf("Create activity[%d]: %v", i, err)
		}
	}

	activities, err := repo.ListByMemberID(memberID)
	if err != nil {
		t.Fatalf("ListByMemberID: %v", err)
	}
	if len(activities) != 2 {
		t.Errorf("expected 2 activities for memberID=42, got %d", len(activities))
	}
}

func TestSecretsVaultActivityRepo_ListRecent(t *testing.T) {
	db := newVaultTestDB(t)
	repo := NewSecretsVaultActivityRepository(db)

	for i := range 10 {
		a := &models.SecretsVaultActivity{
			Label:      "key",
			Action:     models.VaultActionUnlock,
			Successful: true,
		}
		if err := repo.Create(a); err != nil {
			t.Fatalf("Create activity[%d]: %v", i, err)
		}
	}

	recent, err := repo.ListRecent(5)
	if err != nil {
		t.Fatalf("ListRecent: %v", err)
	}
	if len(recent) != 5 {
		t.Errorf("expected 5 recent, got %d", len(recent))
	}

	all, err := repo.ListRecent(0)
	if err != nil {
		t.Fatalf("ListRecent(0): %v", err)
	}
	if len(all) != 10 {
		t.Errorf("expected 10 with limit=0, got %d", len(all))
	}
}
