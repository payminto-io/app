package service

import (
	"fmt"
	"net/url"

	"github.com/payminto/payminto/backend/internal/models"
)

// OnrampService integrates with an external card-to-crypto onramp provider
// (e.g. Onramper) to generate redirect sessions for fiat payment flows.
type OnrampService struct {
	providerURL string
	apiKey      string
}

// NewOnrampService constructs an OnrampService. Pass empty strings for
// providerURL/apiKey to disable onramp (CreateSession returns an error).
func NewOnrampService(providerURL, apiKey string) *OnrampService {
	return &OnrampService{providerURL: providerURL, apiKey: apiKey}
}

// OnrampSession holds the redirect URL and session identifier returned by the
// onramp provider for a specific payment.
type OnrampSession struct {
	RedirectURL string `json:"redirectURL"`
	SessionID   string `json:"sessionID"`
}

// CreateSession builds a provider redirect URL pre-filled with payment details.
// Returns an error if the provider URL is not configured.
func (s *OnrampService) CreateSession(payment *models.PaymentRequest, depositAddress, currency string) (*OnrampSession, error) {
	if s.providerURL == "" {
		return nil, fmt.Errorf("onramp provider not configured")
	}
	params := url.Values{
		"apiKey":        {s.apiKey},
		"defaultAmount": {payment.AmountInUSD.String()},
		"defaultCrypto": {currency},
		"walletAddress": {depositAddress},
		"redirectUrl":   {fmt.Sprintf("/payments?reference_id=%s", payment.ReferenceID)},
	}
	return &OnrampSession{
		RedirectURL: s.providerURL + "?" + params.Encode(),
		SessionID:   payment.ReferenceID,
	}, nil
}
