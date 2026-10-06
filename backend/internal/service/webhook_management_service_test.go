package service

import (
	"errors"
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newWebhookMgmtTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Webhook{},
		&models.WebhookDeliveryLog{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func newWebhookMgmtSvc(t *testing.T) (*WebhookManagementService, *gorm.DB) {
	t.Helper()
	db := newWebhookMgmtTestDB(t)
	webhookRepo := repository.NewWebhookRepository(db)
	deliveryRepo := repository.NewWebhookDeliveryLogRepository(db)
	return NewWebhookManagementService(webhookRepo, deliveryRepo), db
}

func TestWebhookManagement_Create(t *testing.T) {
	svc, _ := newWebhookMgmtSvc(t)

	w, err := svc.Create(CreateWebhookInput{
		PlatformID: 1,
		URL:        "https://example.com/webhook",
		Events:     "payment.confirmed",
		Active:     true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if w.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if w.Secret == "" {
		t.Error("expected non-empty secret")
	}
	if w.ExternalPlatformID != 1 {
		t.Errorf("platform ID: got %d want 1", w.ExternalPlatformID)
	}
}

func TestWebhookManagement_SecretIsRandom(t *testing.T) {
	svc, _ := newWebhookMgmtSvc(t)

	w1, _ := svc.Create(CreateWebhookInput{PlatformID: 1, URL: "https://a.com/wh", Active: true})
	w2, _ := svc.Create(CreateWebhookInput{PlatformID: 1, URL: "https://b.com/wh", Active: true})

	if w1.Secret == w2.Secret {
		t.Error("different webhooks should have different secrets")
	}
}

func TestWebhookManagement_GetByID_TenantIsolation(t *testing.T) {
	svc, _ := newWebhookMgmtSvc(t)

	// Platform 1 creates a webhook.
	w, _ := svc.Create(CreateWebhookInput{PlatformID: 1, URL: "https://a.com/wh", Active: true})

	// Platform 2 cannot see it.
	_, err := svc.GetByID(w.ID, 2)
	if !errors.Is(err, ErrWebhookTenantMismatch) {
		t.Errorf("expected ErrWebhookTenantMismatch, got %v", err)
	}

	// Platform 1 can see it.
	got, err := svc.GetByID(w.ID, 1)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ID != w.ID {
		t.Errorf("ID mismatch: got %d want %d", got.ID, w.ID)
	}
}

func TestWebhookManagement_ListByPlatform(t *testing.T) {
	svc, _ := newWebhookMgmtSvc(t)

	svc.Create(CreateWebhookInput{PlatformID: 10, URL: "https://a.com/wh", Active: true})
	svc.Create(CreateWebhookInput{PlatformID: 10, URL: "https://b.com/wh", Active: true})
	svc.Create(CreateWebhookInput{PlatformID: 99, URL: "https://c.com/wh", Active: true})

	list, err := svc.ListByPlatform(10)
	if err != nil {
		t.Fatalf("ListByPlatform: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 webhooks for platform 10, got %d", len(list))
	}
}

func TestWebhookManagement_Update(t *testing.T) {
	svc, _ := newWebhookMgmtSvc(t)

	w, _ := svc.Create(CreateWebhookInput{PlatformID: 5, URL: "https://old.com/wh", Active: true})
	active := false
	updated, err := svc.Update(w.ID, 5, UpdateWebhookInput{
		URL:    "https://new.com/wh",
		Active: &active,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.URL != "https://new.com/wh" {
		t.Errorf("URL not updated: %s", updated.URL)
	}
	if updated.Active {
		t.Error("expected active to be false")
	}
}

func TestWebhookManagement_Delete(t *testing.T) {
	svc, _ := newWebhookMgmtSvc(t)

	w, _ := svc.Create(CreateWebhookInput{PlatformID: 7, URL: "https://a.com/wh", Active: true})
	if err := svc.Delete(w.ID, 7); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Should now return not-found.
	_, err := svc.GetByID(w.ID, 7)
	if !errors.Is(err, ErrWebhookNotFound) {
		t.Errorf("expected ErrWebhookNotFound after delete, got %v", err)
	}
}

func TestWebhookManagement_RegenerateSecret(t *testing.T) {
	svc, _ := newWebhookMgmtSvc(t)

	w, _ := svc.Create(CreateWebhookInput{PlatformID: 3, URL: "https://a.com/wh", Active: true})
	original := w.Secret

	newSecret, err := svc.RegenerateSecret(w.ID, 3)
	if err != nil {
		t.Fatalf("RegenerateSecret: %v", err)
	}
	if newSecret == original {
		t.Error("regenerated secret should differ from original")
	}
	if newSecret == "" {
		t.Error("regenerated secret should not be empty")
	}
}

func TestWebhookManagement_RegenerateSecret_WrongPlatform(t *testing.T) {
	svc, _ := newWebhookMgmtSvc(t)

	w, _ := svc.Create(CreateWebhookInput{PlatformID: 3, URL: "https://a.com/wh", Active: true})

	_, err := svc.RegenerateSecret(w.ID, 99)
	if !errors.Is(err, ErrWebhookTenantMismatch) {
		t.Errorf("expected ErrWebhookTenantMismatch, got %v", err)
	}
}
