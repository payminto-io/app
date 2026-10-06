package dto

import (
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/shopspring/decimal"
)

type CreatePaymentRequest struct {
	AmountInUSD    decimal.Decimal `json:"amountInUSD" binding:"required"`
	CustomerEmail  *string         `json:"customerEmail"`
	CustomerID     *string         `json:"customerID"`
	InvoiceID      *string         `json:"invoiceID"`
	BlockchainCode string          `json:"blockchainCode"`
	CurrencyCode   string          `json:"currencyCode"`
}

type CreatePaymentResponse struct {
	Host           string  `json:"host"`
	ReferenceID    string  `json:"reference_id"`
	URL            string  `json:"url"`
	DepositAddress *string `json:"depositAddress,omitempty"`
	BlockchainCode *string `json:"blockchainCode,omitempty"`
}

type PaymentResponse struct {
	ReferenceID   string          `json:"referenceID"`
	AmountInUSD   decimal.Decimal `json:"amountInUSD"`
	State         string          `json:"paymentState"`
	CustomerEmail *string         `json:"customerEmail,omitempty"`
	CustomerID    *string         `json:"customerID,omitempty"`
	InvoiceID     *string         `json:"invoiceID,omitempty"`
	CreatedAt     string          `json:"createdAt"`
	ExpiresAt     *string         `json:"expiresAt,omitempty"`
}

func ToPaymentResponse(p *models.PaymentRequest) PaymentResponse {
	resp := PaymentResponse{
		ReferenceID:   p.ReferenceID,
		AmountInUSD:   p.AmountInUSD,
		State:         p.State,
		CustomerEmail: p.CustomerEmail,
		CustomerID:    p.CustomerID,
		InvoiceID:     p.InvoiceID,
		CreatedAt:     p.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if p.ExpiresAt != nil {
		s := p.ExpiresAt.Format("2006-01-02T15:04:05Z")
		resp.ExpiresAt = &s
	}
	return resp
}
