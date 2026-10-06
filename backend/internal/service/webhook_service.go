package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
)

// WebhookService delivers outbound webhook payloads to merchant-configured
// URLs and records each attempt in the WebhookDeliveryLog.
type WebhookService struct {
	webhookRepo     repository.WebhookRepository
	deliveryLogRepo repository.WebhookDeliveryLogRepository
}

// NewWebhookService constructs a WebhookService backed by the given repositories.
func NewWebhookService(webhookRepo repository.WebhookRepository, deliveryLogRepo repository.WebhookDeliveryLogRepository) *WebhookService {
	return &WebhookService{webhookRepo: webhookRepo, deliveryLogRepo: deliveryLogRepo}
}

// GenerateWebhookSignature computes the HMAC-SHA256 signature of payload using secret.
func GenerateWebhookSignature(payload, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyWebhookSignature reports whether the signature matches the expected
// HMAC-SHA256 of payload with secret. Uses hmac.Equal for constant-time comparison.
func VerifyWebhookSignature(payload, signature, secret string) bool {
	expected := GenerateWebhookSignature(payload, secret)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// WebhookPayload is the JSON body posted to merchant webhook endpoints.
type WebhookPayload struct {
	Event     string `json:"event"`
	Timestamp string `json:"timestamp"`
	Data      any    `json:"data"`
}

// Deliver posts event data to the webhook URL, signs the request with HMAC-SHA256,
// and logs the delivery attempt regardless of success.
func (s *WebhookService) Deliver(webhook *models.Webhook, event string, data any) error {
	payload := WebhookPayload{
		Event:     event,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data:      data,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	signature := GenerateWebhookSignature(string(body), webhook.Secret)

	req, err := http.NewRequest("POST", webhook.URL, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Payminto-Signature", signature)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)

	status := "failed"
	var respCode *int
	if resp != nil {
		respCode = &resp.StatusCode
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			status = "delivered"
		}
		resp.Body.Close()
	}

	deliveryLog := models.WebhookDeliveryLog{
		Event:        event,
		Payload:      string(body),
		Status:       status,
		ResponseCode: respCode,
		Attempts:     1,
		WebhookID:    webhook.ID,
	}
	if status == "delivered" {
		now := time.Now()
		deliveryLog.DeliveredAt = &now
	}
	s.deliveryLogRepo.Create(&deliveryLog) //nolint:errcheck

	if err != nil {
		return fmt.Errorf("webhook delivery: %w", err)
	}
	return nil
}
