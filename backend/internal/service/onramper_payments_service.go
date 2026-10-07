package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ErrOnramperSessionNotFound signals an unknown session.
var ErrOnramperSessionNotFound = errors.New("onramper session not found")

// ErrOnramperSignatureInvalid signals a webhook with a bad HMAC signature.
var ErrOnramperSignatureInvalid = errors.New("onramper webhook signature invalid")

// ErrOnramperAlreadyTerminal signals a webhook callback for a session that
// has already been completed/failed/expired.
var ErrOnramperAlreadyTerminal = errors.New("onramper session already in terminal state")

// CreateOnramperSessionInput is the request payload for CreateSession.
type CreateOnramperSessionInput struct {
	PlatformID       uint
	PaymentRequestID *uint
	FiatAmount       decimal.Decimal
	FiatCurrency     string
	CryptoCurrency   string
	BlockchainCode   string
	WalletAddress    string
}

// OnramperPaymentsService manages Onramper card-to-crypto sessions. Replaces
// the URL-builder stub from earlier phases. Persists every session to the
// onramper_payments table and tracks lifecycle states via atomic conditional
// UPDATEs in the repo layer.
//
// Real on-chain settlement happens out-of-band via Onramper's provider; we
// only track the state machine and link the session to a PaymentRequest
// once the customer's fiat→crypto conversion lands on-chain.
type OnramperPaymentsService struct {
	repo          repository.OnramperPaymentsRepository
	providerURL   string
	apiKey        string
	webhookSecret string
}

// NewOnramperPaymentsService wires the service. providerURL/apiKey/webhookSecret
// come from configuration; pass empty strings to disable provider integration
// (CreateSession still works for tests).
func NewOnramperPaymentsService(
	repo repository.OnramperPaymentsRepository,
	providerURL, apiKey, webhookSecret string,
) *OnramperPaymentsService {
	return &OnramperPaymentsService{
		repo:          repo,
		providerURL:   providerURL,
		apiKey:        apiKey,
		webhookSecret: webhookSecret,
	}
}

// CreateSession persists a new session row and returns the redirect URL the
// customer should be sent to. The session ID is a UUID generated locally —
// in production this would be the provider-issued ID, but for the stub the
// local UUID is good enough until Phase K wires the real Onramper API call.
func (s *OnramperPaymentsService) CreateSession(input CreateOnramperSessionInput) (*models.OnramperPayments, string, error) {
	if input.PlatformID == 0 {
		return nil, "", errors.New("platform id is required")
	}
	if input.WalletAddress == "" {
		return nil, "", errors.New("wallet address is required")
	}
	if input.FiatAmount.LessThanOrEqual(decimal.Zero) {
		return nil, "", errors.New("fiat amount must be positive")
	}

	sessionID := "ompr_" + uuid.NewString()
	row := &models.OnramperPayments{
		SessionID:          sessionID,
		PaymentRequestID:   input.PaymentRequestID,
		ExternalPlatformID: input.PlatformID,
		Status:             models.OnramperStatusCreated,
		FiatAmount:         input.FiatAmount,
		FiatCurrency:       input.FiatCurrency,
		CryptoCurrency:     input.CryptoCurrency,
		BlockchainCode:     input.BlockchainCode,
		WalletAddress:      input.WalletAddress,
		OnrampProvider:     "onramper",
	}
	if err := s.repo.Create(row); err != nil {
		return nil, "", fmt.Errorf("create onramper session: %w", err)
	}

	redirectURL := s.buildRedirectURL(row)
	return row, redirectURL, nil
}

// buildRedirectURL constructs the provider redirect URL pre-filled with
// session details. Returns an empty string if no provider URL is configured.
func (s *OnramperPaymentsService) buildRedirectURL(row *models.OnramperPayments) string {
	if s.providerURL == "" {
		return ""
	}
	params := url.Values{
		"apiKey":        {s.apiKey},
		"sessionId":     {row.SessionID},
		"defaultAmount": {row.FiatAmount.String()},
		"defaultFiat":   {row.FiatCurrency},
		"defaultCrypto": {row.CryptoCurrency},
		"walletAddress": {row.WalletAddress},
	}
	return s.providerURL + "?" + params.Encode()
}

// GetByID returns a session by primary key.
func (s *OnramperPaymentsService) GetByID(id uint) (*models.OnramperPayments, error) {
	row, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOnramperSessionNotFound
		}
		return nil, err
	}
	return row, nil
}

// GetBySessionID looks up a session by its public session ID.
func (s *OnramperPaymentsService) GetBySessionID(sessionID string) (*models.OnramperPayments, error) {
	row, err := s.repo.GetBySessionID(sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOnramperSessionNotFound
		}
		return nil, err
	}
	return row, nil
}

// ListByPlatform returns sessions for a platform with optional pagination.
func (s *OnramperPaymentsService) ListByPlatform(platformID uint) ([]models.OnramperPayments, error) {
	return s.repo.ListByPlatform(platformID)
}

// HandleWebhook validates the HMAC signature and updates the session state
// based on the payload. Mock-friendly: takes the raw payload bytes and the
// signature header so callers don't need to know the wire format.
//
// Expected payload (JSON):
//
//	{
//	  "sessionId": "ompr_...",
//	  "status": "completed" | "failed" | "expired",
//	  "providerTxId": "...",
//	  "onchainTxHash": "0x...",
//	  "cryptoAmount": "0.0123",
//	  "failureReason": "..."  // only on failed
//	}
//
// Returns nil on successful state advance, ErrOnramperAlreadyTerminal when
// the session was already in a terminal state, ErrOnramperSignatureInvalid
// on bad HMAC.
func (s *OnramperPaymentsService) HandleWebhook(payload []byte, signature string, parsed OnramperWebhookPayload) error {
	if !s.verifySignature(payload, signature) {
		return ErrOnramperSignatureInvalid
	}

	row, err := s.GetBySessionID(parsed.SessionID)
	if err != nil {
		return err
	}

	switch parsed.Status {
	case models.OnramperStatusCompleted:
		rowsAffected, err := s.repo.MarkCompleted(row.ID, parsed.ProviderTxID, parsed.OnchainTxHash, parsed.CryptoAmount)
		if err != nil {
			return fmt.Errorf("mark completed: %w", err)
		}
		if rowsAffected == 0 {
			return ErrOnramperAlreadyTerminal
		}
	case models.OnramperStatusFailed:
		rowsAffected, err := s.repo.MarkFailed(row.ID, parsed.FailureReason)
		if err != nil {
			return fmt.Errorf("mark failed: %w", err)
		}
		if rowsAffected == 0 {
			return ErrOnramperAlreadyTerminal
		}
	default:
		return fmt.Errorf("unsupported webhook status %q", parsed.Status)
	}
	return nil
}

// OnramperWebhookPayload is the parsed shape of an Onramper webhook callback.
// The handler unmarshals the raw bytes into this before passing to HandleWebhook.
type OnramperWebhookPayload struct {
	SessionID     string `json:"sessionId"`
	Status        string `json:"status"`
	ProviderTxID  string `json:"providerTxId"`
	OnchainTxHash string `json:"onchainTxHash"`
	CryptoAmount  string `json:"cryptoAmount"`
	FailureReason string `json:"failureReason"`
}

// verifySignature performs constant-time HMAC-SHA256 verification of the
// webhook payload against the configured webhook secret. Returns true on
// match, false otherwise. Fail-closed: an unset secret rejects every
// webhook. The boot path MUST provide a non-empty secret in any environment
// that exposes the webhook endpoint.
func (s *OnramperPaymentsService) verifySignature(payload []byte, signature string) bool {
	if s.webhookSecret == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(s.webhookSecret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// MarkExpired flips stale pending sessions to expired. Called by the
// onramper expiry worker every 5 minutes. olderThan is typically time.Now() - 30min.
func (s *OnramperPaymentsService) MarkExpired(olderThan time.Time) (int64, error) {
	return s.repo.MarkExpired(olderThan)
}
