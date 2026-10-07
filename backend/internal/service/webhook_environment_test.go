package service

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestWebhookDeliver_NamesTheEnvironmentInsideTheSignedBody(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Webhook{}, &models.WebhookDeliveryLog{}); err != nil {
		t.Fatal(err)
	}
	var body []byte
	var signature string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		signature = r.Header.Get("X-Payminto-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	svc := NewWebhookService(repository.NewWebhookRepository(db), repository.NewWebhookDeliveryLogRepository(db))
	hook := &models.Webhook{URL: srv.URL, Secret: "whsec", Events: "payment.completed", Active: true}
	hook.ID = 1
	if err := svc.Deliver(hook, "payment.completed", map[string]any{"id": 1}); !errors.Is(err, environment.ErrUnconfigured) {
		t.Fatalf("Deliver without an environment = %v, want ErrUnconfigured", err)
	}
	svc.SetEnvironment(environment.Live)
	if err := svc.Deliver(hook, "payment.completed", map[string]any{"id": 1}); err != nil {
		t.Fatal(err)
	}
	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Environment != "live" || payload.Event != "payment.completed" {
		t.Fatalf("payload = %+v", payload)
	}
	if !VerifyWebhookSignature(string(body), signature, "whsec") {
		t.Fatal("signature does not cover the body that names the environment")
	}
}
