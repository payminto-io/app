package repository

import (
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// openWebhookTestDB opens an in-memory SQLite database and auto-migrates the
// models required for webhook tests.
func openWebhookTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.ExternalPlatform{},
		&models.Webhook{},
		&models.WebhookDeliveryLog{},
	); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}
	return db
}

// seedWebhookPlatform creates a minimal ExternalPlatform and returns its ID.
func seedWebhookPlatform(t *testing.T, db *gorm.DB) uint {
	t.Helper()
	platform := models.ExternalPlatform{
		Name:            "Webhook Test Platform",
		BrandColor:      "#ffffff",
		Website:         "https://example.com",
		SuccessEndpoint: "https://example.com/success",
	}
	if err := db.Create(&platform).Error; err != nil {
		t.Fatalf("seed ExternalPlatform: %v", err)
	}
	return platform.ID
}

// ---------------------------------------------------------------------------
// WebhookRepository tests
// ---------------------------------------------------------------------------

func TestWebhookRepo_CreateAndGet(t *testing.T) {
	db := openWebhookTestDB(t)
	platformID := seedWebhookPlatform(t, db)
	repo := NewWebhookRepository(db)

	w := &models.Webhook{
		URL:                "https://example.com/hook",
		Secret:             "supersecret",
		Events:             "payment.completed",
		Active:             true,
		ExternalPlatformID: platformID,
	}
	if err := repo.Create(w); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if w.ID == 0 {
		t.Fatal("expected non-zero ID after Create")
	}

	got, err := repo.GetByID(w.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.URL != "https://example.com/hook" {
		t.Errorf("expected URL %q, got %q", "https://example.com/hook", got.URL)
	}
}

func TestWebhookRepo_GetActiveByPlatform(t *testing.T) {
	db := openWebhookTestDB(t)
	platformID := seedWebhookPlatform(t, db)
	repo := NewWebhookRepository(db)

	active := &models.Webhook{
		URL:                "https://example.com/active",
		Secret:             "s1",
		Events:             "payment.completed",
		Active:             true,
		ExternalPlatformID: platformID,
	}
	// Create as active first (default:true), then disable via SetActive so that
	// the false value is persisted despite GORM's zero-value skip on Create.
	inactive := &models.Webhook{
		URL:                "https://example.com/inactive",
		Secret:             "s2",
		Events:             "payment.completed",
		Active:             true,
		ExternalPlatformID: platformID,
	}
	if err := repo.Create(active); err != nil {
		t.Fatalf("Create active: %v", err)
	}
	if err := repo.Create(inactive); err != nil {
		t.Fatalf("Create inactive: %v", err)
	}
	// Now disable the second one.
	if err := repo.SetActive(inactive.ID, false); err != nil {
		t.Fatalf("SetActive(false): %v", err)
	}

	results, err := repo.GetActiveByPlatform(platformID)
	if err != nil {
		t.Fatalf("GetActiveByPlatform: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 active webhook, got %d", len(results))
	}
	if results[0].ID != active.ID {
		t.Errorf("expected active webhook ID %d, got %d", active.ID, results[0].ID)
	}
}

func TestWebhookRepo_SetActive(t *testing.T) {
	db := openWebhookTestDB(t)
	platformID := seedWebhookPlatform(t, db)
	repo := NewWebhookRepository(db)

	w := &models.Webhook{
		URL:                "https://example.com/hook2",
		Secret:             "s3",
		Events:             "all",
		Active:             true,
		ExternalPlatformID: platformID,
	}
	if err := repo.Create(w); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.SetActive(w.ID, false); err != nil {
		t.Fatalf("SetActive(false): %v", err)
	}

	got, err := repo.GetByID(w.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Active {
		t.Error("expected Active=false after SetActive(false)")
	}

	if err := repo.SetActive(w.ID, true); err != nil {
		t.Fatalf("SetActive(true): %v", err)
	}
	got, err = repo.GetByID(w.ID)
	if err != nil {
		t.Fatalf("GetByID after re-enable: %v", err)
	}
	if !got.Active {
		t.Error("expected Active=true after SetActive(true)")
	}
}

func TestWebhookRepo_UpdateSecret(t *testing.T) {
	db := openWebhookTestDB(t)
	platformID := seedWebhookPlatform(t, db)
	repo := NewWebhookRepository(db)

	w := &models.Webhook{
		URL:                "https://example.com/hook3",
		Secret:             "oldsecret",
		Events:             "all",
		Active:             true,
		ExternalPlatformID: platformID,
	}
	if err := repo.Create(w); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.UpdateSecret(w.ID, "newsecret"); err != nil {
		t.Fatalf("UpdateSecret: %v", err)
	}

	// Fetch the raw record to verify secret was updated.
	var updated models.Webhook
	if err := db.First(&updated, w.ID).Error; err != nil {
		t.Fatalf("fetch updated: %v", err)
	}
	if updated.Secret != "newsecret" {
		t.Errorf("expected secret 'newsecret', got %q", updated.Secret)
	}
}

// ---------------------------------------------------------------------------
// WebhookDeliveryLogRepository tests
// ---------------------------------------------------------------------------

func TestWebhookDeliveryLogRepo_CreateAndGet(t *testing.T) {
	db := openWebhookTestDB(t)
	platformID := seedWebhookPlatform(t, db)
	webhookRepo := NewWebhookRepository(db)
	logRepo := NewWebhookDeliveryLogRepository(db)

	w := &models.Webhook{
		URL:                "https://example.com/hook4",
		Secret:             "s4",
		Events:             "all",
		Active:             true,
		ExternalPlatformID: platformID,
	}
	if err := webhookRepo.Create(w); err != nil {
		t.Fatalf("Create webhook: %v", err)
	}

	l := &models.WebhookDeliveryLog{
		Event:     "payment.completed",
		Payload:   `{"id":1}`,
		Status:    "pending",
		WebhookID: w.ID,
	}
	if err := logRepo.Create(l); err != nil {
		t.Fatalf("Create log: %v", err)
	}
	if l.ID == 0 {
		t.Fatal("expected non-zero ID after Create")
	}

	got, err := logRepo.GetByID(l.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Event != "payment.completed" {
		t.Errorf("expected Event 'payment.completed', got %q", got.Event)
	}
	if got.Webhook == nil {
		t.Error("expected Webhook to be preloaded")
	}
}

func TestWebhookDeliveryLogRepo_ListByStatus(t *testing.T) {
	db := openWebhookTestDB(t)
	platformID := seedWebhookPlatform(t, db)
	webhookRepo := NewWebhookRepository(db)
	logRepo := NewWebhookDeliveryLogRepository(db)

	w := &models.Webhook{
		URL:                "https://example.com/hook5",
		Secret:             "s5",
		Events:             "all",
		Active:             true,
		ExternalPlatformID: platformID,
	}
	if err := webhookRepo.Create(w); err != nil {
		t.Fatalf("Create webhook: %v", err)
	}

	for i, status := range []string{"pending", "failed", "pending"} {
		l := &models.WebhookDeliveryLog{
			Event:     "payment.completed",
			Payload:   `{}`,
			Status:    status,
			WebhookID: w.ID,
		}
		l.CreatedAt = time.Now().Add(time.Duration(i) * time.Second)
		if err := logRepo.Create(l); err != nil {
			t.Fatalf("Create log %d: %v", i, err)
		}
	}

	results, err := logRepo.ListByStatus("pending", 10)
	if err != nil {
		t.Fatalf("ListByStatus: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 pending logs, got %d", len(results))
	}
	for _, r := range results {
		if r.Webhook == nil {
			t.Error("expected Webhook to be preloaded in ListByStatus result")
		}
	}
}

func TestWebhookDeliveryLogRepo_ListPendingRetries(t *testing.T) {
	db := openWebhookTestDB(t)
	platformID := seedWebhookPlatform(t, db)
	webhookRepo := NewWebhookRepository(db)
	logRepo := NewWebhookDeliveryLogRepository(db)

	w := &models.Webhook{
		URL:                "https://example.com/hook6",
		Secret:             "s6",
		Events:             "all",
		Active:             true,
		ExternalPlatformID: platformID,
	}
	if err := webhookRepo.Create(w); err != nil {
		t.Fatalf("Create webhook: %v", err)
	}

	// A failed log with a past next_retry_at — should be returned.
	pastRetry := time.Now().Add(-1 * time.Hour)
	eligible := &models.WebhookDeliveryLog{
		Event:       "payment.completed",
		Payload:     `{}`,
		Status:      "failed",
		WebhookID:   w.ID,
		NextRetryAt: &pastRetry,
	}
	if err := logRepo.Create(eligible); err != nil {
		t.Fatalf("Create eligible log: %v", err)
	}

	// A failed log with a future next_retry_at — should NOT be returned.
	futureRetry := time.Now().Add(1 * time.Hour)
	notYet := &models.WebhookDeliveryLog{
		Event:       "payment.completed",
		Payload:     `{}`,
		Status:      "failed",
		WebhookID:   w.ID,
		NextRetryAt: &futureRetry,
	}
	if err := logRepo.Create(notYet); err != nil {
		t.Fatalf("Create not-yet log: %v", err)
	}

	results, err := logRepo.ListPendingRetries(time.Now(), 10)
	if err != nil {
		t.Fatalf("ListPendingRetries: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 eligible retry, got %d", len(results))
	}
	if results[0].ID != eligible.ID {
		t.Errorf("expected eligible log ID %d, got %d", eligible.ID, results[0].ID)
	}
	if results[0].Webhook == nil {
		t.Error("expected Webhook to be preloaded")
	}
}

func TestWebhookDeliveryLogRepo_IncrementAttempts(t *testing.T) {
	db := openWebhookTestDB(t)
	platformID := seedWebhookPlatform(t, db)
	webhookRepo := NewWebhookRepository(db)
	logRepo := NewWebhookDeliveryLogRepository(db)

	w := &models.Webhook{
		URL:                "https://example.com/hook7",
		Secret:             "s7",
		Events:             "all",
		Active:             true,
		ExternalPlatformID: platformID,
	}
	if err := webhookRepo.Create(w); err != nil {
		t.Fatalf("Create webhook: %v", err)
	}

	l := &models.WebhookDeliveryLog{
		Event:     "payment.completed",
		Payload:   `{}`,
		Status:    "pending",
		Attempts:  0,
		WebhookID: w.ID,
	}
	if err := logRepo.Create(l); err != nil {
		t.Fatalf("Create log: %v", err)
	}

	nextRetry := time.Now().Add(5 * time.Minute)
	if err := logRepo.IncrementAttempts(l.ID, &nextRetry); err != nil {
		t.Fatalf("IncrementAttempts: %v", err)
	}

	var updated models.WebhookDeliveryLog
	if err := db.First(&updated, l.ID).Error; err != nil {
		t.Fatalf("fetch updated: %v", err)
	}
	if updated.Attempts != 1 {
		t.Errorf("expected Attempts=1, got %d", updated.Attempts)
	}
	if updated.NextRetryAt == nil {
		t.Error("expected NextRetryAt to be set")
	}
}

func TestWebhookDeliveryLogRepo_MarkDelivered(t *testing.T) {
	db := openWebhookTestDB(t)
	platformID := seedWebhookPlatform(t, db)
	webhookRepo := NewWebhookRepository(db)
	logRepo := NewWebhookDeliveryLogRepository(db)

	w := &models.Webhook{
		URL:                "https://example.com/hook8",
		Secret:             "s8",
		Events:             "all",
		Active:             true,
		ExternalPlatformID: platformID,
	}
	if err := webhookRepo.Create(w); err != nil {
		t.Fatalf("Create webhook: %v", err)
	}

	l := &models.WebhookDeliveryLog{
		Event:     "payment.completed",
		Payload:   `{}`,
		Status:    "pending",
		WebhookID: w.ID,
	}
	if err := logRepo.Create(l); err != nil {
		t.Fatalf("Create log: %v", err)
	}

	if err := logRepo.MarkDelivered(l.ID, 200); err != nil {
		t.Fatalf("MarkDelivered: %v", err)
	}

	var updated models.WebhookDeliveryLog
	if err := db.First(&updated, l.ID).Error; err != nil {
		t.Fatalf("fetch updated: %v", err)
	}
	if updated.Status != "delivered" {
		t.Errorf("expected Status='delivered', got %q", updated.Status)
	}
	if updated.ResponseCode == nil || *updated.ResponseCode != 200 {
		t.Errorf("expected ResponseCode=200")
	}
	if updated.DeliveredAt == nil {
		t.Error("expected DeliveredAt to be set")
	}
}
