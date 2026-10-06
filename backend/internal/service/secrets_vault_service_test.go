package service

import (
	"errors"
	"sync"
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newVaultServiceDB(t *testing.T) *gorm.DB {
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

func newVaultService(t *testing.T) (*SecretsVaultService, repository.SecretsVaultActivityRepository) {
	t.Helper()
	db := newVaultServiceDB(t)
	vaultRepo := repository.NewSecretsVaultRepository(db)
	actRepo := repository.NewSecretsVaultActivityRepository(db)
	svc := NewSecretsVaultService(vaultRepo, actRepo)
	return svc, actRepo
}

const testPassphrase = "hunter2-test-passphrase"

func TestVault_UnlockFirstBoot_CreatesSentinel(t *testing.T) {
	svc, _ := newVaultService(t)

	if svc.IsUnlocked() {
		t.Fatal("expected vault to start locked")
	}

	if err := svc.Unlock(testPassphrase, nil); err != nil {
		t.Fatalf("Unlock (first boot): %v", err)
	}

	if !svc.IsUnlocked() {
		t.Error("expected vault to be unlocked after Unlock")
	}
}

func TestVault_UnlockSecondBoot_VerifiesPassphrase(t *testing.T) {
	db := newVaultServiceDB(t)
	vaultRepo := repository.NewSecretsVaultRepository(db)
	actRepo := repository.NewSecretsVaultActivityRepository(db)

	// First boot — initializes sentinel and salt
	svc1 := NewSecretsVaultService(vaultRepo, actRepo)
	if err := svc1.Unlock(testPassphrase, nil); err != nil {
		t.Fatalf("first boot Unlock: %v", err)
	}

	// Simulate restart: new service instance, same db (sentinel and salt already exist)
	svc2 := NewSecretsVaultService(vaultRepo, actRepo)
	if err := svc2.Unlock(testPassphrase, nil); err != nil {
		t.Fatalf("second boot Unlock: %v", err)
	}
	if !svc2.IsUnlocked() {
		t.Error("expected vault to be unlocked on second boot")
	}
}

func TestVault_UnlockWrongPassphrase_Fails(t *testing.T) {
	db := newVaultServiceDB(t)
	vaultRepo := repository.NewSecretsVaultRepository(db)
	actRepo := repository.NewSecretsVaultActivityRepository(db)

	// Initialize with correct passphrase
	svc1 := NewSecretsVaultService(vaultRepo, actRepo)
	if err := svc1.Unlock(testPassphrase, nil); err != nil {
		t.Fatalf("init Unlock: %v", err)
	}

	// Try wrong passphrase
	svc2 := NewSecretsVaultService(vaultRepo, actRepo)
	err := svc2.Unlock("wrong-passphrase-totally-different", nil)
	if err == nil {
		t.Fatal("expected error with wrong passphrase, got nil")
	}
	if svc2.IsUnlocked() {
		t.Error("vault should remain locked after wrong passphrase")
	}
}

func TestVault_StoreAndGet_Roundtrip(t *testing.T) {
	svc, _ := newVaultService(t)

	if err := svc.Unlock(testPassphrase, nil); err != nil {
		t.Fatalf("Unlock: %v", err)
	}

	const label = "hd_wallet.eth.mnemonic"
	const secret = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

	if err := svc.StoreKey(label, secret, models.SecretTypeMnemonic, nil); err != nil {
		t.Fatalf("StoreKey: %v", err)
	}

	got, err := svc.GetKey(label, nil)
	if err != nil {
		t.Fatalf("GetKey: %v", err)
	}
	if got != secret {
		t.Errorf("plaintext mismatch: got %q want %q", got, secret)
	}
}

func TestVault_StoreWhenLocked_Fails(t *testing.T) {
	svc, _ := newVaultService(t)
	// Vault is locked by default — do NOT unlock

	err := svc.StoreKey("some.label", "secret", models.SecretTypeOther, nil)
	if err == nil {
		t.Fatal("expected error when storing to locked vault, got nil")
	}
}

func TestVault_Lock_ClearsMasterKey(t *testing.T) {
	svc, _ := newVaultService(t)

	if err := svc.Unlock(testPassphrase, nil); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	if !svc.IsUnlocked() {
		t.Fatal("expected unlocked")
	}

	svc.Lock(nil)
	if svc.IsUnlocked() {
		t.Error("expected vault to be locked after Lock()")
	}

	// masterKeyBytes must be nil (zeroed and released) after Lock.
	svc.mu.RLock()
	keyIsNil := svc.masterKeyBytes == nil
	svc.mu.RUnlock()
	if !keyIsNil {
		t.Error("expected masterKeyBytes to be nil after Lock()")
	}

	// GetKey should fail after lock
	_, err := svc.GetKey("any.label", nil)
	if err == nil {
		t.Error("expected error from GetKey when locked, got nil")
	}
}

func TestVault_DeleteKey(t *testing.T) {
	svc, _ := newVaultService(t)

	if err := svc.Unlock(testPassphrase, nil); err != nil {
		t.Fatalf("Unlock: %v", err)
	}

	const label = "delete.this.key"
	if err := svc.StoreKey(label, "supersecret", models.SecretTypeAPIKey, nil); err != nil {
		t.Fatalf("StoreKey: %v", err)
	}

	// Verify it exists
	if _, err := svc.GetKey(label, nil); err != nil {
		t.Fatalf("GetKey before delete: %v", err)
	}

	if err := svc.DeleteKey(label, nil); err != nil {
		t.Fatalf("DeleteKey: %v", err)
	}

	// Should be gone
	_, err := svc.GetKey(label, nil)
	if err == nil {
		t.Error("expected error after DeleteKey, got nil")
	}
}

// TestVault_DeleteKey_WhenLocked_Fails verifies I2: DeleteKey checks lock state.
func TestVault_DeleteKey_WhenLocked_Fails(t *testing.T) {
	svc, _ := newVaultService(t)
	// Vault is locked by default — DeleteKey must fail immediately.
	err := svc.DeleteKey("any.label", nil)
	if err == nil {
		t.Fatal("expected error from DeleteKey when vault is locked, got nil")
	}
}

func TestVault_RotateMasterKey(t *testing.T) {
	db := newVaultServiceDB(t)
	vaultRepo := repository.NewSecretsVaultRepository(db)
	actRepo := repository.NewSecretsVaultActivityRepository(db)
	svc := NewSecretsVaultService(vaultRepo, actRepo)

	// Initialize vault
	if err := svc.Unlock(testPassphrase, nil); err != nil {
		t.Fatalf("Unlock: %v", err)
	}

	// Store a few secrets
	secrets := map[string]string{
		"wallet.eth.mnemonic": "word1 word2 word3",
		"wallet.btc.key":      "privkey-btc-hex",
		"smtp.password":       "smtp-secret-pass",
	}
	for label, value := range secrets {
		if err := svc.StoreKey(label, value, models.SecretTypeOther, nil); err != nil {
			t.Fatalf("StoreKey %s: %v", label, err)
		}
	}

	const newPassphrase = "new-secure-passphrase-for-rotation"

	// Rotate
	if err := svc.RotateMasterKey(testPassphrase, newPassphrase, nil); err != nil {
		t.Fatalf("RotateMasterKey: %v", err)
	}

	// All secrets must still be retrievable via the now-unlocked service.
	for label, expected := range secrets {
		got, err := svc.GetKey(label, nil)
		if err != nil {
			t.Fatalf("GetKey(%s) after rotation (same svc): %v", label, err)
		}
		if got != expected {
			t.Errorf("GetKey(%s): got %q want %q", label, got, expected)
		}
	}

	// Simulate restart — new service instance, unlock with new passphrase.
	svc2 := NewSecretsVaultService(vaultRepo, actRepo)
	if err := svc2.Unlock(newPassphrase, nil); err != nil {
		t.Fatalf("Unlock with new passphrase: %v", err)
	}

	for label, expected := range secrets {
		got, err := svc2.GetKey(label, nil)
		if err != nil {
			t.Fatalf("GetKey(%s) after rotation: %v", label, err)
		}
		if got != expected {
			t.Errorf("GetKey(%s): got %q want %q", label, got, expected)
		}
	}

	// Old passphrase should no longer work.
	svc3 := NewSecretsVaultService(vaultRepo, actRepo)
	if err := svc3.Unlock(testPassphrase, nil); err == nil {
		t.Error("expected old passphrase to fail after rotation")
	}
}

// TestVault_RotateMasterKey_WrongOldPassphrase ensures state is not mutated
// when the old passphrase is incorrect (covers I3).
func TestVault_RotateMasterKey_WrongOldPassphrase(t *testing.T) {
	svc, _ := newVaultService(t)

	if err := svc.Unlock(testPassphrase, nil); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	if err := svc.StoreKey("some.secret", "value", models.SecretTypeOther, nil); err != nil {
		t.Fatalf("StoreKey: %v", err)
	}

	// Capture current key before the failed rotation.
	svc.mu.RLock()
	keyBefore := make([]byte, len(svc.masterKeyBytes))
	copy(keyBefore, svc.masterKeyBytes)
	svc.mu.RUnlock()

	err := svc.RotateMasterKey("wrong-old-passphrase", "new-passphrase", nil)
	if err == nil {
		t.Fatal("expected error when old passphrase is wrong, got nil")
	}

	// Service key must be unchanged — vault is still usable with original passphrase.
	svc.mu.RLock()
	keyAfter := svc.masterKeyBytes
	svc.mu.RUnlock()

	if len(keyAfter) != len(keyBefore) {
		t.Fatalf("key length changed after failed rotation: %d -> %d", len(keyBefore), len(keyAfter))
	}
	for i := range keyBefore {
		if keyBefore[i] != keyAfter[i] {
			t.Fatal("master key was mutated by a failed RotateMasterKey call")
		}
	}

	// Verify we can still retrieve the secret with the original (unchanged) key.
	got, err := svc.GetKey("some.secret", nil)
	if err != nil {
		t.Fatalf("GetKey after failed rotation: %v", err)
	}
	if got != "value" {
		t.Errorf("unexpected value: %q", got)
	}
}

// TestVault_PerVaultSalt verifies C3: two independently initialised vaults
// with the same passphrase produce DIFFERENT derived keys.
func TestVault_PerVaultSalt_DifferentKeys(t *testing.T) {
	db1 := newVaultServiceDB(t)
	db2 := newVaultServiceDB(t)

	svc1 := NewSecretsVaultService(
		repository.NewSecretsVaultRepository(db1),
		repository.NewSecretsVaultActivityRepository(db1),
	)
	svc2 := NewSecretsVaultService(
		repository.NewSecretsVaultRepository(db2),
		repository.NewSecretsVaultActivityRepository(db2),
	)

	const sharedPassphrase = "same-passphrase-both-vaults"

	if err := svc1.Unlock(sharedPassphrase, nil); err != nil {
		t.Fatalf("svc1.Unlock: %v", err)
	}
	if err := svc2.Unlock(sharedPassphrase, nil); err != nil {
		t.Fatalf("svc2.Unlock: %v", err)
	}

	svc1.mu.RLock()
	key1 := make([]byte, len(svc1.masterKeyBytes))
	copy(key1, svc1.masterKeyBytes)
	svc1.mu.RUnlock()

	svc2.mu.RLock()
	key2 := make([]byte, len(svc2.masterKeyBytes))
	copy(key2, svc2.masterKeyBytes)
	svc2.mu.RUnlock()

	if len(key1) != 32 || len(key2) != 32 {
		t.Fatalf("expected 32-byte keys, got %d and %d", len(key1), len(key2))
	}

	equal := true
	for i := range key1 {
		if key1[i] != key2[i] {
			equal = false
			break
		}
	}
	if equal {
		t.Error("expected different derived keys for independent vaults with the same passphrase (per-vault salt should differ)")
	}
}

// TestVault_Lock_ZeroesBytes verifies C5: after Lock(), the byte slice is nil.
func TestVault_Lock_ZeroesBytes(t *testing.T) {
	svc, _ := newVaultService(t)

	if err := svc.Unlock(testPassphrase, nil); err != nil {
		t.Fatalf("Unlock: %v", err)
	}

	// Key must be populated before lock.
	svc.mu.RLock()
	hasKey := len(svc.masterKeyBytes) == 32
	svc.mu.RUnlock()
	if !hasKey {
		t.Fatal("expected 32-byte master key after unlock")
	}

	svc.Lock(nil)

	svc.mu.RLock()
	nilAfterLock := svc.masterKeyBytes == nil
	svc.mu.RUnlock()
	if !nilAfterLock {
		t.Error("expected masterKeyBytes to be nil after Lock()")
	}
}

// TestVault_RotateMasterKey_Atomic_AllSecretsReadable verifies the atomic
// rotation: every secret stored before rotation is readable after rotation.
func TestVault_RotateMasterKey_Atomic_AllSecretsReadable(t *testing.T) {
	db := newVaultServiceDB(t)
	vaultRepo := repository.NewSecretsVaultRepository(db)
	actRepo := repository.NewSecretsVaultActivityRepository(db)
	svc := NewSecretsVaultService(vaultRepo, actRepo)

	if err := svc.Unlock(testPassphrase, nil); err != nil {
		t.Fatalf("Unlock: %v", err)
	}

	// Store many secrets to increase chance of partial failure detection.
	want := make(map[string]string)
	for i := range 10 {
		label := "secret." + string(rune('a'+i))
		value := "value-" + label
		want[label] = value
		if err := svc.StoreKey(label, value, models.SecretTypeOther, nil); err != nil {
			t.Fatalf("StoreKey %s: %v", label, err)
		}
	}

	const newPass = "atomic-rotation-new-pass"
	if err := svc.RotateMasterKey(testPassphrase, newPass, nil); err != nil {
		t.Fatalf("RotateMasterKey: %v", err)
	}

	// New service instance unlocks with new passphrase.
	svc2 := NewSecretsVaultService(vaultRepo, actRepo)
	if err := svc2.Unlock(newPass, nil); err != nil {
		t.Fatalf("Unlock with new passphrase: %v", err)
	}
	for label, expected := range want {
		got, err := svc2.GetKey(label, nil)
		if err != nil {
			t.Fatalf("GetKey(%s) after rotation: %v", label, err)
		}
		if got != expected {
			t.Errorf("GetKey(%s) = %q, want %q", label, got, expected)
		}
	}
}

// TestVault_RotateMasterKey_BulkFailure_StateUntouched uses a repo wrapper
// that fails BulkUpdateCiphertexts and verifies the service key is not changed.
func TestVault_RotateMasterKey_BulkFailure_StateUntouched(t *testing.T) {
	db := newVaultServiceDB(t)
	realRepo := repository.NewSecretsVaultRepository(db)
	actRepo := repository.NewSecretsVaultActivityRepository(db)

	svc := NewSecretsVaultService(realRepo, actRepo)
	if err := svc.Unlock(testPassphrase, nil); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	if err := svc.StoreKey("k", "v", models.SecretTypeOther, nil); err != nil {
		t.Fatalf("StoreKey: %v", err)
	}

	// Wrap the repo with one that fails BulkUpdateCiphertexts.
	failing := &bulkFailRepo{SecretsVaultRepository: realRepo}
	svc.repo = failing

	svc.mu.RLock()
	keyBefore := make([]byte, len(svc.masterKeyBytes))
	copy(keyBefore, svc.masterKeyBytes)
	svc.mu.RUnlock()

	err := svc.RotateMasterKey(testPassphrase, "new-pass", nil)
	if err == nil {
		t.Fatal("expected error from RotateMasterKey when BulkUpdateCiphertexts fails")
	}

	svc.mu.RLock()
	keyAfter := svc.masterKeyBytes
	svc.mu.RUnlock()

	if len(keyAfter) != len(keyBefore) {
		t.Fatalf("key length changed: %d -> %d", len(keyBefore), len(keyAfter))
	}
	for i := range keyBefore {
		if keyBefore[i] != keyAfter[i] {
			t.Fatal("master key was mutated despite BulkUpdateCiphertexts failure")
		}
	}
}

// bulkFailRepo wraps a real SecretsVaultRepository and injects an error in
// BulkUpdateCiphertexts to simulate a transactional failure.
type bulkFailRepo struct {
	repository.SecretsVaultRepository
	mu sync.Mutex
}

func (r *bulkFailRepo) BulkUpdateCiphertexts(_ map[uint]string) error {
	return errors.New("simulated bulk update failure")
}

func TestVault_ActivityLogged(t *testing.T) {
	svc, actRepo := newVaultService(t)

	memberID := uint(1)

	// Unlock (first boot — logs VaultActionInitialize)
	if err := svc.Unlock(testPassphrase, &memberID); err != nil {
		t.Fatalf("Unlock: %v", err)
	}

	// Store (logs VaultActionStore)
	if err := svc.StoreKey("log.test.key", "mysecret", models.SecretTypeSMTPPassword, &memberID); err != nil {
		t.Fatalf("StoreKey: %v", err)
	}

	// Get (logs VaultActionRetrieve)
	if _, err := svc.GetKey("log.test.key", &memberID); err != nil {
		t.Fatalf("GetKey: %v", err)
	}

	// Verify at least 3 activity rows were written
	activities, err := actRepo.ListRecent(0)
	if err != nil {
		t.Fatalf("ListRecent: %v", err)
	}
	if len(activities) < 3 {
		t.Errorf("expected at least 3 activity rows, got %d", len(activities))
	}
}

func TestVault_StoreKey_OverwritesExisting(t *testing.T) {
	svc, _ := newVaultService(t)

	if err := svc.Unlock(testPassphrase, nil); err != nil {
		t.Fatalf("Unlock: %v", err)
	}

	const label = "overwrite.key"

	if err := svc.StoreKey(label, "original-value", models.SecretTypeOther, nil); err != nil {
		t.Fatalf("StoreKey (original): %v", err)
	}

	if err := svc.StoreKey(label, "updated-value", models.SecretTypeOther, nil); err != nil {
		t.Fatalf("StoreKey (updated): %v", err)
	}

	got, err := svc.GetKey(label, nil)
	if err != nil {
		t.Fatalf("GetKey: %v", err)
	}
	if got != "updated-value" {
		t.Errorf("expected updated value, got %q", got)
	}
}

func TestVault_GetKey_WhenLocked_Fails(t *testing.T) {
	svc, _ := newVaultService(t)
	// Never unlocked — GetKey must fail
	_, err := svc.GetKey("any.key", nil)
	if err == nil {
		t.Error("expected error when vault is locked")
	}
}

// TestDeriveKey_WithSalt ensures the same passphrase + same salt → same key,
// and same passphrase + different salt → different key (C3).
func TestDeriveKey_WithSalt(t *testing.T) {
	salt1 := []byte("salt-aaaaaaaaaaaaa")
	salt2 := []byte("salt-bbbbbbbbbbbbb")

	k1a, err := deriveKey([]byte("my-passphrase"), salt1)
	if err != nil {
		t.Fatalf("deriveKey: %v", err)
	}
	k1b, err := deriveKey([]byte("my-passphrase"), salt1)
	if err != nil {
		t.Fatalf("deriveKey: %v", err)
	}

	// Same passphrase + same salt → same key.
	if string(k1a) != string(k1b) {
		t.Error("expected same key for same passphrase+salt")
	}
	if len(k1a) != 32 {
		t.Errorf("expected 32-byte key, got %d", len(k1a))
	}

	// Same passphrase + different salt → different key.
	k2, err := deriveKey([]byte("my-passphrase"), salt2)
	if err != nil {
		t.Fatalf("deriveKey: %v", err)
	}
	if string(k1a) == string(k2) {
		t.Error("expected different keys when salts differ")
	}

	// Different passphrase + same salt → different key.
	k3, err := deriveKey([]byte("different-passphrase"), salt1)
	if err != nil {
		t.Fatalf("deriveKey: %v", err)
	}
	if string(k1a) == string(k3) {
		t.Error("expected different keys for different passphrases")
	}
}

func TestSha256Hex(t *testing.T) {
	h := sha256Hex("hello")
	if len(h) != 64 {
		t.Errorf("expected 64-char hex, got %d", len(h))
	}
}
