package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupOnramperDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.OnramperPayments{},
		&models.PaymentRequest{},
		&models.ExternalPlatform{},
	); err != nil {
		t.Fatal(err)
	}
	return db
}

func newOnramperService(t *testing.T, secret string) (*OnramperPaymentsService, *gorm.DB) {
	t.Helper()
	db := setupOnramperDB(t)
	return NewOnramperPaymentsService(
		repository.NewOnramperPaymentsRepository(db),
		"https://onramper.example",
		"test-api-key",
		secret,
	), db
}

func TestOnramperPaymentsService_CreateSession_Success(t *testing.T) {
	svc, _ := newOnramperService(t, "")

	row, redirectURL, err := svc.CreateSession(CreateOnramperSessionInput{
		PlatformID:     1,
		FiatAmount:     decimal.NewFromInt(100),
		FiatCurrency:   "USD",
		CryptoCurrency: "USDC",
		BlockchainCode: "BASE",
		WalletAddress:  "0xdest",
	})
	if err != nil {
		t.Fatal(err)
	}
	if row.SessionID == "" || row.Status != models.OnramperStatusCreated {
		t.Errorf("session not initialized correctly: %+v", row)
	}
	if redirectURL == "" {
		t.Error("expected non-empty redirect URL")
	}
}

func TestOnramperPaymentsService_CreateSession_Validation(t *testing.T) {
	svc, _ := newOnramperService(t, "")
	cases := []CreateOnramperSessionInput{
		{},
		{PlatformID: 1, WalletAddress: "0x", FiatAmount: decimal.Zero},
		{PlatformID: 1, FiatAmount: decimal.NewFromInt(10), WalletAddress: ""},
	}
	for i, c := range cases {
		if _, _, err := svc.CreateSession(c); err == nil {
			t.Errorf("case %d: expected validation error", i)
		}
	}
}

func TestOnramperPaymentsService_HandleWebhook_BadSignature(t *testing.T) {
	svc, _ := newOnramperService(t, "shared-secret")

	row, _, err := svc.CreateSession(CreateOnramperSessionInput{
		PlatformID:    1,
		FiatAmount:    decimal.NewFromInt(50),
		FiatCurrency:  "USD",
		CryptoCurrency: "USDC", BlockchainCode: "BASE", WalletAddress: "0x",
	})
	if err != nil {
		t.Fatal(err)
	}

	payload := []byte(`{"sessionId":"` + row.SessionID + `","status":"completed"}`)
	err = svc.HandleWebhook(payload, "wrong-signature", OnramperWebhookPayload{
		SessionID: row.SessionID,
		Status:    models.OnramperStatusCompleted,
	})
	if !errors.Is(err, ErrOnramperSignatureInvalid) {
		t.Errorf("expected ErrOnramperSignatureInvalid, got %v", err)
	}
}

func TestOnramperPaymentsService_HandleWebhook_Completed(t *testing.T) {
	secret := "shared-secret"
	svc, db := newOnramperService(t, secret)

	row, _, err := svc.CreateSession(CreateOnramperSessionInput{
		PlatformID:    1,
		FiatAmount:    decimal.NewFromInt(50),
		FiatCurrency:  "USD",
		CryptoCurrency: "USDC", BlockchainCode: "BASE", WalletAddress: "0xwallet",
	})
	if err != nil {
		t.Fatal(err)
	}

	parsed := OnramperWebhookPayload{
		SessionID:     row.SessionID,
		Status:        models.OnramperStatusCompleted,
		ProviderTxID:  "ompr_tx_123",
		OnchainTxHash: "0xdeadbeef",
		CryptoAmount:  "0.0123",
	}
	payload := []byte(`{"sessionId":"` + row.SessionID + `","status":"completed"}`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))

	if err := svc.HandleWebhook(payload, signature, parsed); err != nil {
		t.Fatal(err)
	}

	var updated models.OnramperPayments
	db.First(&updated, row.ID)
	if updated.Status != models.OnramperStatusCompleted {
		t.Errorf("expected completed, got %s", updated.Status)
	}
	if updated.OnchainTxHash == nil || *updated.OnchainTxHash != "0xdeadbeef" {
		t.Errorf("expected onchain tx hash recorded, got %v", updated.OnchainTxHash)
	}
}

func TestOnramperPaymentsService_HandleWebhook_AlreadyTerminal(t *testing.T) {
	secret := "shared-secret"
	svc, _ := newOnramperService(t, secret)

	row, _, _ := svc.CreateSession(CreateOnramperSessionInput{
		PlatformID:    1,
		FiatAmount:    decimal.NewFromInt(50),
		FiatCurrency:  "USD",
		CryptoCurrency: "USDC", BlockchainCode: "BASE", WalletAddress: "0x",
	})

	parsed := OnramperWebhookPayload{
		SessionID: row.SessionID,
		Status:    models.OnramperStatusCompleted,
	}
	payload := []byte(`{"sessionId":"` + row.SessionID + `","status":"completed"}`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sig := hex.EncodeToString(mac.Sum(nil))

	if err := svc.HandleWebhook(payload, sig, parsed); err != nil {
		t.Fatal(err)
	}
	// Second call should hit terminal state guard.
	if err := svc.HandleWebhook(payload, sig, parsed); !errors.Is(err, ErrOnramperAlreadyTerminal) {
		t.Errorf("expected ErrOnramperAlreadyTerminal, got %v", err)
	}
}

func TestOnramperPaymentsService_HandleWebhook_EmptySecretFailsClosed(t *testing.T) {
	svc, _ := newOnramperService(t, "")

	row, _, _ := svc.CreateSession(CreateOnramperSessionInput{
		PlatformID:    1,
		FiatAmount:    decimal.NewFromInt(50),
		FiatCurrency:  "USD",
		CryptoCurrency: "USDC", BlockchainCode: "BASE", WalletAddress: "0x",
	})
	parsed := OnramperWebhookPayload{SessionID: row.SessionID, Status: models.OnramperStatusCompleted}
	if err := svc.HandleWebhook([]byte("{}"), "", parsed); !errors.Is(err, ErrOnramperSignatureInvalid) {
		t.Errorf("empty secret must reject; got %v", err)
	}
}

func TestOnramperPaymentsService_MarkExpired(t *testing.T) {
	svc, db := newOnramperService(t, "")

	row, _, _ := svc.CreateSession(CreateOnramperSessionInput{
		PlatformID:    1,
		FiatAmount:    decimal.NewFromInt(50),
		FiatCurrency:  "USD",
		CryptoCurrency: "USDC", BlockchainCode: "BASE", WalletAddress: "0x",
	})

	// Force created_at into the past so MarkExpired picks it up.
	pastTime := time.Now().Add(-1 * time.Hour)
	db.Model(&models.OnramperPayments{}).Where("id = ?", row.ID).Update("created_at", pastTime)

	rowsAffected, err := svc.MarkExpired(time.Now().Add(-30 * time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if rowsAffected != 1 {
		t.Errorf("expected 1 row marked expired, got %d", rowsAffected)
	}
}
