package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/payminto/payminto/backend/internal/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newWebhookHandler(t *testing.T) (*WebhookHandler, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.ExternalPlatform{},
		&models.Webhook{},
		&models.WebhookDeliveryLog{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	webhookRepo := repository.NewWebhookRepository(db)
	deliveryRepo := repository.NewWebhookDeliveryLogRepository(db)
	return NewWebhookHandler(service.NewWebhookManagementService(webhookRepo, deliveryRepo)), db
}

func TestWebhookHandler_Create_UsesArrayEventsContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, db := newWebhookHandler(t)

	r := gin.New()
	r.POST("/webhooks", authInjector(1, 41), h.CreateWebhook)
	body := []byte(`{"url":"https://merchant.example/hooks","events":[" payment.confirmed ","withdrawal.sent","payment.confirmed"],"active":true}`)
	req := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	r.ServeHTTP(response, req)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusCreated, response.Body.String())
	}

	var envelope struct {
		Webhook struct {
			ExternalPlatformID uint     `json:"externalPlatformID"`
			Events             []string `json:"events"`
			Secret             string   `json:"secret"`
		} `json:"webhook"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.Webhook.ExternalPlatformID != 41 {
		t.Fatalf("externalPlatformID = %d, want 41", envelope.Webhook.ExternalPlatformID)
	}
	if len(envelope.Webhook.Events) != 2 || envelope.Webhook.Events[0] != "payment.confirmed" || envelope.Webhook.Events[1] != "withdrawal.sent" {
		t.Fatalf("events = %#v, want canonical event array", envelope.Webhook.Events)
	}
	if envelope.Webhook.Secret == "" {
		t.Fatal("create response must reveal the signing secret once")
	}

	var stored models.Webhook
	if err := db.First(&stored).Error; err != nil {
		t.Fatalf("load stored webhook: %v", err)
	}
	if stored.ExternalPlatformID != 41 {
		t.Fatalf("stored tenant = %d, want 41", stored.ExternalPlatformID)
	}
}

func TestWebhookHandler_Create_RejectsMalformedEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		body string
	}{
		{name: "empty array", body: `{"url":"https://merchant.example/hooks","events":[]}`},
		{name: "blank event", body: `{"url":"https://merchant.example/hooks","events":["  "]}`},
		{name: "comma delimited event", body: `{"url":"https://merchant.example/hooks","events":["payment.confirmed,withdrawal.sent"]}`},
		{name: "string instead of array", body: `{"url":"https://merchant.example/hooks","events":"payment.confirmed"}`},
		{name: "removed description", body: `{"url":"https://merchant.example/hooks","events":["payment.confirmed"],"description":"orders"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, _ := newWebhookHandler(t)
			r := gin.New()
			r.POST("/webhooks", authInjector(1, 41), h.CreateWebhook)
			req := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			r.ServeHTTP(response, req)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%s", response.Code, response.Body.String())
			}
		})
	}
}

func TestWebhookHandler_CreateListGetUpdate_RoundTripAndRedactSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newWebhookHandler(t)
	r := gin.New()
	r.POST("/webhooks", authInjector(1, 41), h.CreateWebhook)
	r.GET("/webhooks", authInjector(1, 41), h.ListWebhooks)
	r.GET("/webhooks/:id", authInjector(1, 41), h.GetWebhook)
	r.PUT("/webhooks/:id", authInjector(1, 41), h.UpdateWebhook)

	create := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewBufferString(
		`{"url":"https://merchant.example/hooks","events":["payment.confirmed"],"active":true}`,
	))
	create.Header.Set("Content-Type", "application/json")
	created := httptest.NewRecorder()
	r.ServeHTTP(created, create)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d; body=%s", created.Code, created.Body.String())
	}

	var createdEnvelope struct {
		Webhook struct {
			ID     uint     `json:"id"`
			Events []string `json:"events"`
			Secret string   `json:"secret"`
		} `json:"webhook"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &createdEnvelope); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if createdEnvelope.Webhook.Secret == "" {
		t.Fatal("create must reveal secret once")
	}
	idPath := "/webhooks/" + strconv.FormatUint(uint64(createdEnvelope.Webhook.ID), 10)

	update := httptest.NewRequest(http.MethodPut, idPath, bytes.NewBufferString(
		`{"events":[" withdrawal.sent ","payment.failed","withdrawal.sent"]}`,
	))
	update.Header.Set("Content-Type", "application/json")
	updated := httptest.NewRecorder()
	r.ServeHTTP(updated, update)
	if updated.Code != http.StatusOK {
		t.Fatalf("update status = %d; body=%s", updated.Code, updated.Body.String())
	}
	assertWebhookEventsAndSecret(t, updated.Body.Bytes(), []string{"withdrawal.sent", "payment.failed"}, false)

	got := httptest.NewRecorder()
	r.ServeHTTP(got, httptest.NewRequest(http.MethodGet, idPath, nil))
	if got.Code != http.StatusOK {
		t.Fatalf("get status = %d; body=%s", got.Code, got.Body.String())
	}
	assertWebhookEventsAndSecret(t, got.Body.Bytes(), []string{"withdrawal.sent", "payment.failed"}, false)

	listed := httptest.NewRecorder()
	r.ServeHTTP(listed, httptest.NewRequest(http.MethodGet, "/webhooks", nil))
	if listed.Code != http.StatusOK {
		t.Fatalf("list status = %d; body=%s", listed.Code, listed.Body.String())
	}
	var listEnvelope struct {
		Webhooks []json.RawMessage `json:"webhooks"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &listEnvelope); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listEnvelope.Webhooks) != 1 {
		t.Fatalf("webhooks length = %d, want 1", len(listEnvelope.Webhooks))
	}
	assertWebhookEventsAndSecret(t, listEnvelope.Webhooks[0], []string{"withdrawal.sent", "payment.failed"}, false)
}

func assertWebhookEventsAndSecret(t *testing.T, body []byte, wantEvents []string, wantSecret bool) {
	t.Helper()
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("decode webhook: %v", err)
	}
	if wrapped, ok := decoded["webhook"].(map[string]any); ok {
		decoded = wrapped
	}
	events, ok := decoded["events"].([]any)
	if !ok || len(events) != len(wantEvents) {
		t.Fatalf("events = %#v, want %#v", decoded["events"], wantEvents)
	}
	for i := range wantEvents {
		if events[i] != wantEvents[i] {
			t.Fatalf("events[%d] = %#v, want %q", i, events[i], wantEvents[i])
		}
	}
	_, hasSecret := decoded["secret"]
	if hasSecret != wantSecret {
		t.Fatalf("secret presence = %v, want %v", hasSecret, wantSecret)
	}
}

func TestWebhookHandler_Get_IsTenantScoped(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, db := newWebhookHandler(t)
	w := models.Webhook{
		URL:                "https://merchant.example/hooks",
		Secret:             "secret",
		Events:             "payment.confirmed",
		Active:             true,
		ExternalPlatformID: 41,
	}
	if err := db.Create(&w).Error; err != nil {
		t.Fatalf("seed webhook: %v", err)
	}
	r := gin.New()
	r.GET("/webhooks/:id", authInjector(1, 42), h.GetWebhook)
	response := httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/webhooks/"+strconv.FormatUint(uint64(w.ID), 10), nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", response.Code, response.Body.String())
	}
}
