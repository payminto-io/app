package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"

	"github.com/payminto/payminto/backend/internal/crypto"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"golang.org/x/crypto/scrypt"
)

// SecretsVaultService stores encrypted secrets (HD wallet mnemonics, private
// keys, SMTP passwords) using a scrypt-derived AES-256-GCM key. The master key
// is held in memory only after Unlock; Lock zeroes and clears it. Every
// operation writes an activity row for audit.
//
// Thread-safe for concurrent Unlock/Lock and key operations.
type SecretsVaultService struct {
	repo         repository.SecretsVaultRepository
	activityRepo repository.SecretsVaultActivityRepository

	mu             sync.RWMutex
	unlocked       bool
	masterKeyBytes []byte // 32 raw bytes. Explicitly zeroed on Lock().
}

// NewSecretsVaultService constructs a SecretsVaultService backed by the given
// repositories.
func NewSecretsVaultService(repo repository.SecretsVaultRepository, activityRepo repository.SecretsVaultActivityRepository) *SecretsVaultService {
	return &SecretsVaultService{repo: repo, activityRepo: activityRepo}
}

// IsUnlocked reports whether the vault has been unlocked in this process.
func (s *SecretsVaultService) IsUnlocked() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.unlocked
}

// Unlock derives the master AES key from the passphrase and marks the vault
// as unlocked. It uses a per-vault random salt stored in "_vault_salt". If
// neither that entry nor the sentinel exist, this is first boot — a new salt is
// generated, stored, and the sentinel is created.
//
// Returns an error if the passphrase is wrong (sentinel decryption fails) or
// if any persistence operation fails.
func (s *SecretsVaultService) Unlock(passphrase string, memberID *uint) error {
	if passphrase == "" {
		return errors.New("passphrase required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Attempt to fetch the per-vault salt.
	saltEntry, saltErr := s.repo.GetByLabel("_vault_salt")

	if saltErr != nil {
		// First boot: no salt row (and therefore no sentinel either).
		// Generate a 16-byte random salt and persist it as hex plaintext.
		rawSalt := make([]byte, 16)
		if _, err := rand.Read(rawSalt); err != nil {
			return fmt.Errorf("generate salt: %w", err)
		}
		saltHex := hex.EncodeToString(rawSalt)

		saltRow := &models.SecretsVault{
			Label:       "_vault_salt",
			Ciphertext:  saltHex, // stored in the ciphertext column but NOT encrypted
			SecretType:  models.SecretTypeOther,
			CreatedByID: memberID,
		}
		if err := s.repo.Create(saltRow); err != nil {
			return fmt.Errorf("store salt: %w", err)
		}

		key, err := deriveKey([]byte(passphrase), rawSalt)
		if err != nil {
			return fmt.Errorf("derive key: %w", err)
		}

		// Create the sentinel encrypted with the derived key.
		ciphertext, encErr := crypto.EncryptBytes("payminto-vault-v1", key)
		if encErr != nil {
			s.logActivity(nil, "_vault_sentinel", models.VaultActionFailedUnlock, memberID, false, "encrypt: "+encErr.Error())
			return fmt.Errorf("encrypt sentinel: %w", encErr)
		}
		sentinel := &models.SecretsVault{
			Label:       "_vault_sentinel",
			Ciphertext:  ciphertext,
			SecretType:  models.SecretTypeOther,
			CreatedByID: memberID,
		}
		if err := s.repo.Create(sentinel); err != nil {
			return fmt.Errorf("create sentinel: %w", err)
		}

		s.masterKeyBytes = key
		s.unlocked = true
		s.logActivity(&sentinel.ID, "_vault_sentinel", models.VaultActionInitialize, memberID, true, "vault initialized")
		return nil
	}

	// Existing vault: decode stored salt and derive key.
	rawSalt, err := hex.DecodeString(saltEntry.Ciphertext)
	if err != nil {
		return fmt.Errorf("decode vault salt: %w", err)
	}

	key, err := deriveKey([]byte(passphrase), rawSalt)
	if err != nil {
		return fmt.Errorf("derive key: %w", err)
	}

	sentinel, err := s.repo.GetByLabel("_vault_sentinel")
	if err != nil {
		return fmt.Errorf("get sentinel: %w", err)
	}

	plaintext, err := crypto.DecryptBytes(sentinel.Ciphertext, key)
	if err != nil || plaintext != "payminto-vault-v1" {
		s.logActivity(&sentinel.ID, "_vault_sentinel", models.VaultActionFailedUnlock, memberID, false, "wrong passphrase")
		return errors.New("wrong passphrase")
	}

	s.masterKeyBytes = key
	s.unlocked = true
	s.logActivity(&sentinel.ID, "_vault_sentinel", models.VaultActionUnlock, memberID, true, "")
	return nil
}

// Lock zeroes the in-memory master key and marks the vault as locked.
func (s *SecretsVaultService) Lock(memberID *uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.masterKeyBytes {
		s.masterKeyBytes[i] = 0
	}
	s.masterKeyBytes = nil
	s.unlocked = false
	s.logActivity(nil, "_vault", models.VaultActionLock, memberID, true, "")
}

// StoreKey encrypts plaintext and persists it under label. Overwrites any
// existing row with the same label.
func (s *SecretsVaultService) StoreKey(label, plaintext, secretType string, memberID *uint) error {
	s.mu.RLock()
	if !s.unlocked {
		s.mu.RUnlock()
		return errors.New("vault locked")
	}
	key := make([]byte, len(s.masterKeyBytes))
	copy(key, s.masterKeyBytes)
	s.mu.RUnlock()

	ciphertext, err := crypto.EncryptBytes(plaintext, key)
	if err != nil {
		s.logActivity(nil, label, models.VaultActionStore, memberID, false, err.Error())
		return fmt.Errorf("encrypt: %w", err)
	}

	existing, err := s.repo.GetByLabel(label)
	if err == nil && existing != nil {
		existing.Ciphertext = ciphertext
		existing.SecretType = secretType
		if err := s.repo.Update(existing); err != nil {
			s.logActivity(&existing.ID, label, models.VaultActionStore, memberID, false, err.Error())
			return err
		}
		s.logActivity(&existing.ID, label, models.VaultActionStore, memberID, true, "updated")
		return nil
	}

	v := &models.SecretsVault{
		Label:       label,
		Ciphertext:  ciphertext,
		SecretType:  secretType,
		CreatedByID: memberID,
	}
	if err := s.repo.Create(v); err != nil {
		s.logActivity(nil, label, models.VaultActionStore, memberID, false, err.Error())
		return err
	}
	s.logActivity(&v.ID, label, models.VaultActionStore, memberID, true, "created")
	return nil
}

// GetKey returns the decrypted plaintext for the given label.
func (s *SecretsVaultService) GetKey(label string, memberID *uint) (string, error) {
	s.mu.RLock()
	if !s.unlocked {
		s.mu.RUnlock()
		return "", errors.New("vault locked")
	}
	key := make([]byte, len(s.masterKeyBytes))
	copy(key, s.masterKeyBytes)
	s.mu.RUnlock()

	v, err := s.repo.GetByLabel(label)
	if err != nil {
		s.logActivity(nil, label, models.VaultActionRetrieve, memberID, false, "not found")
		return "", fmt.Errorf("not found: %w", err)
	}

	plaintext, err := crypto.DecryptBytes(v.Ciphertext, key)
	if err != nil {
		s.logActivity(&v.ID, label, models.VaultActionRetrieve, memberID, false, "decrypt failed")
		return "", fmt.Errorf("decrypt: %w", err)
	}

	_ = s.repo.TouchLastUsed(v.ID)
	s.logActivity(&v.ID, label, models.VaultActionRetrieve, memberID, true, "")
	return plaintext, nil
}

// DeleteKey removes a secret and audits the deletion.
// Returns an error if the vault is locked.
func (s *SecretsVaultService) DeleteKey(label string, memberID *uint) error {
	if !s.IsUnlocked() {
		return errors.New("vault locked")
	}
	v, err := s.repo.GetByLabel(label)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(v.ID); err != nil {
		s.logActivity(&v.ID, label, models.VaultActionDelete, memberID, false, err.Error())
		return err
	}
	s.logActivity(&v.ID, label, models.VaultActionDelete, memberID, true, "")
	return nil
}

// RotateMasterKey re-encrypts every secret under a new passphrase. The
// operation is fully atomic: all ciphertext updates are applied in a single
// database transaction. Service state (masterKeyBytes) is only updated after
// the transaction commits successfully.
//
// The old passphrase is verified without mutating service state; the vault may
// be locked or unlocked when this is called.
func (s *SecretsVaultService) RotateMasterKey(oldPassphrase, newPassphrase string, memberID *uint) error {
	if oldPassphrase == "" || newPassphrase == "" {
		return errors.New("passphrases must not be empty")
	}

	// Verify the old passphrase and obtain the old key without mutating state.
	oldKey, err := s.verifyPassphrase(oldPassphrase)
	if err != nil {
		return fmt.Errorf("verify old passphrase: %w", err)
	}

	// Fetch the persisted salt — it stays the same across rotations.
	saltEntry, err := s.repo.GetByLabel("_vault_salt")
	if err != nil {
		return fmt.Errorf("get vault salt: %w", err)
	}
	rawSalt, err := hex.DecodeString(saltEntry.Ciphertext)
	if err != nil {
		return fmt.Errorf("decode vault salt: %w", err)
	}

	newKey, err := deriveKey([]byte(newPassphrase), rawSalt)
	if err != nil {
		return fmt.Errorf("derive new key: %w", err)
	}

	list, err := s.repo.List()
	if err != nil {
		return fmt.Errorf("list vault entries: %w", err)
	}

	// Build the atomic update map: re-encrypt every entry (including sentinel,
	// excluding the salt row which is stored as plaintext hex).
	updates := make(map[uint]string, len(list))
	for _, entry := range list {
		if entry.Label == "_vault_salt" {
			// The salt is not encrypted; skip it.
			continue
		}
		plaintext, err := crypto.DecryptBytes(entry.Ciphertext, oldKey)
		if err != nil {
			return fmt.Errorf("decrypt %s: %w", entry.Label, err)
		}
		newCipher, err := crypto.EncryptBytes(plaintext, newKey)
		if err != nil {
			return fmt.Errorf("encrypt %s: %w", entry.Label, err)
		}
		updates[entry.ID] = newCipher
	}

	// Commit all ciphertext updates in one transaction.
	if err := s.repo.BulkUpdateCiphertexts(updates); err != nil {
		return fmt.Errorf("bulk update ciphertexts: %w", err)
	}

	// Only now update in-memory state.
	s.mu.Lock()
	for i := range s.masterKeyBytes {
		s.masterKeyBytes[i] = 0
	}
	s.masterKeyBytes = newKey
	s.unlocked = true
	s.mu.Unlock()

	s.logActivity(nil, "_vault_master", models.VaultActionRotateKey, memberID, true, fmt.Sprintf("rotated %d entries", len(updates)))
	return nil
}

// verifyPassphrase derives the key from passphrase using the stored salt and
// verifies it by decrypting the sentinel. It returns the derived key without
// mutating any service state. Must NOT hold s.mu.
func (s *SecretsVaultService) verifyPassphrase(passphrase string) ([]byte, error) {
	saltEntry, err := s.repo.GetByLabel("_vault_salt")
	if err != nil {
		return nil, fmt.Errorf("get vault salt: %w", err)
	}
	rawSalt, err := hex.DecodeString(saltEntry.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decode vault salt: %w", err)
	}

	key, err := deriveKey([]byte(passphrase), rawSalt)
	if err != nil {
		return nil, fmt.Errorf("derive key: %w", err)
	}

	sentinel, err := s.repo.GetByLabel("_vault_sentinel")
	if err != nil {
		return nil, fmt.Errorf("get sentinel: %w", err)
	}

	plaintext, err := crypto.DecryptBytes(sentinel.Ciphertext, key)
	if err != nil || plaintext != "payminto-vault-v1" {
		return nil, errors.New("wrong passphrase")
	}

	return key, nil
}

// logActivity writes an audit row. Must NOT hold s.mu when called (it doesn't
// access s fields). Errors are silently dropped — audit failures must not
// block vault operations.
func (s *SecretsVaultService) logActivity(vaultID *uint, label, action string, memberID *uint, successful bool, details string) {
	var d *string
	if details != "" {
		d = &details
	}
	_ = s.activityRepo.Create(&models.SecretsVaultActivity{
		VaultID:    vaultID,
		Label:      label,
		Action:     action,
		MemberID:   memberID,
		Details:    d,
		Successful: successful,
	})
}

// deriveKey derives a 32-byte AES key from passphrase and salt using scrypt
// (N=2^15, r=8, p=1). Returns raw bytes suitable for EncryptBytes/DecryptBytes.
func deriveKey(passphrase, salt []byte) ([]byte, error) {
	// scrypt parameters: N=2^15, r=8, p=1 — standard for memory-hard derivation.
	dk, err := scrypt.Key(passphrase, salt, 32768, 8, 1, 32)
	if err != nil {
		return nil, err
	}
	return dk, nil
}

// sha256Hex is a small helper used by tests.
func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
