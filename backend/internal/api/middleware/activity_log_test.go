package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/models"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// mockActivityLogRepo is a goroutine-safe in-memory ActivityLogRepository for tests.
type mockActivityLogRepo struct {
	mu      sync.Mutex
	created []models.ActivityLog
}

func (m *mockActivityLogRepo) Create(log *models.ActivityLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.created = append(m.created, *log)
	return nil
}

func (m *mockActivityLogRepo) BulkCreate(logs []models.ActivityLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.created = append(m.created, logs...)
	return nil
}

func (m *mockActivityLogRepo) ListByMember(memberID uint, limit int) ([]models.ActivityLog, error) {
	return nil, nil
}

func (m *mockActivityLogRepo) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.created)
}

func (m *mockActivityLogRepo) first() models.ActivityLog {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.created[0]
}

func TestActivityLog_RequestLogged(t *testing.T) {
	repo := &mockActivityLogRepo{}
	stopCh := make(chan struct{})

	r := gin.New()
	r.Use(ActivityLog(repo, stopCh))
	r.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	body := `{"name":"test"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = int64(len(body))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Signal shutdown — this drains the channel and flushes immediately.
	close(stopCh)
	time.Sleep(100 * time.Millisecond)

	if repo.count() == 0 {
		t.Fatal("expected at least one log entry")
	}
	entry := repo.first()
	if entry.Method != http.MethodPost {
		t.Errorf("expected POST, got %s", entry.Method)
	}
	if entry.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", entry.StatusCode)
	}
}

func TestActivityLog_BodyRedaction(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		mustFind string
		mustNot  string
	}{
		{
			name:     "password redacted",
			input:    `{"email":"a@b.com","password":"secret123"}`,
			mustFind: "[REDACTED]",
			mustNot:  "secret123",
		},
		{
			name:     "token redacted",
			input:    `{"token":"abc","data":"ok"}`,
			mustFind: "[REDACTED]",
			mustNot:  "abc",
		},
		{
			name:     "mnemonic redacted",
			input:    `{"mnemonic":"word1 word2 word3"}`,
			mustFind: "[REDACTED]",
			mustNot:  "word1",
		},
		{
			name:     "non-sensitive preserved",
			input:    `{"name":"alice","amount":"100"}`,
			mustFind: "alice",
			mustNot:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := redactBody([]byte(tc.input))
			if !strings.Contains(out, tc.mustFind) {
				t.Errorf("expected %q in output %q", tc.mustFind, out)
			}
			if tc.mustNot != "" && strings.Contains(out, tc.mustNot) {
				t.Errorf("sensitive value %q should not appear in output %q", tc.mustNot, out)
			}
		})
	}
}

func TestActivityLog_NonJSONBodyNotPanics(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("unexpected panic: %v", r)
		}
	}()
	out := redactBody([]byte("not json at all"))
	if len(out) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestActivityLog_AsyncDrain(t *testing.T) {
	repo := &mockActivityLogRepo{}
	stopCh := make(chan struct{})

	r := gin.New()
	r.Use(ActivityLog(repo, stopCh))
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"pong": true})
	})

	// Send several requests.
	for range 5 {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		r.ServeHTTP(w, req)
	}

	// Signal shutdown to drain.
	close(stopCh)

	// Give the goroutine time to flush.
	time.Sleep(200 * time.Millisecond)

	if repo.count() == 0 {
		t.Error("expected entries after drain")
	}
}

func TestRedactBody_MaxBytes(t *testing.T) {
	// Body larger than maxBodyPreviewBytes should be truncated.
	big := bytes.Repeat([]byte("x"), maxBodyPreviewBytes+100)
	out := redactBody(big)
	if len(out) > maxBodyPreviewBytes+10 { // +10 tolerance for JSON quoting
		t.Errorf("body preview too large: %d", len(out))
	}
}

// TestActivityLog_Redaction_MixedCaseKeys verifies that camelCase sensitive
// keys ("apiKey", "privateKey") are redacted even though the map stores
// lowercase versions ("apikey", "privatekey").
func TestActivityLog_Redaction_MixedCaseKeys(t *testing.T) {
	input := `{"apiKey":"pm_abcdef","privateKey":"0xdead","safe":"ok"}`
	out := redactBody([]byte(input))

	if strings.Contains(out, "pm_abcdef") {
		t.Errorf("apiKey value should be redacted, got: %s", out)
	}
	if strings.Contains(out, "0xdead") {
		t.Errorf("privateKey value should be redacted, got: %s", out)
	}
	if !strings.Contains(out, `"ok"`) {
		t.Errorf("non-sensitive field 'safe' should be preserved, got: %s", out)
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Errorf("expected [REDACTED] in output, got: %s", out)
	}
}

// TestActivityLog_LargeBody_Truncated checks that bodies larger than
// maxBodyPreviewBytes are captured at the limit and not beyond.
func TestActivityLog_LargeBody_Truncated(t *testing.T) {
	repo := &mockActivityLogRepo{}
	stopCh := make(chan struct{})

	r := gin.New()
	r.Use(ActivityLog(repo, stopCh))
	r.POST("/upload", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// 10 KB body — well above maxBodyPreviewBytes (4 KB).
	bigBody := bytes.Repeat([]byte("a"), 10*1024)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(bigBody))
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = int64(len(bigBody))
	r.ServeHTTP(w, req)

	close(stopCh)
	time.Sleep(150 * time.Millisecond)

	if repo.count() == 0 {
		t.Fatal("expected at least one log entry")
	}
	entry := repo.first()
	if entry.RequestBodyPreview == nil {
		t.Fatal("expected body preview to be set")
	}
	if len(*entry.RequestBodyPreview) > maxBodyPreviewBytes+50 {
		t.Errorf("body preview too large: %d bytes (limit %d)", len(*entry.RequestBodyPreview), maxBodyPreviewBytes)
	}
}

// panicOnceRepo panics on the first BulkCreate call, then succeeds on
// subsequent calls. Used to verify the writer goroutine's per-flush recovery.
type panicOnceRepo struct {
	mu      sync.Mutex
	paniced bool
	created []models.ActivityLog
}

func (r *panicOnceRepo) Create(log *models.ActivityLog) error { return nil }
func (r *panicOnceRepo) ListByMember(memberID uint, limit int) ([]models.ActivityLog, error) {
	return nil, nil
}

func (r *panicOnceRepo) BulkCreate(logs []models.ActivityLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.paniced {
		r.paniced = true
		panic("simulated flush panic")
	}
	r.created = append(r.created, logs...)
	return nil
}

func (r *panicOnceRepo) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.created)
}

// TestActivityLog_WriterPanicRecovery_ContinuesLogging asserts that a panic
// inside one flush does not kill the writer goroutine — subsequent entries are
// still processed.
func TestActivityLog_WriterPanicRecovery_ContinuesLogging(t *testing.T) {
	repo := &panicOnceRepo{}
	stopCh := make(chan struct{})

	r := gin.New()
	r.Use(ActivityLog(repo, stopCh))
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"pong": true})
	})

	// First batch of requests — will fill the buffer and trigger the panicking flush.
	for range activityLogBatchSize {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		r.ServeHTTP(w, req)
	}

	// Give the goroutine time to flush+panic+recover.
	time.Sleep(200 * time.Millisecond)

	// Second batch — should be processed by the still-alive goroutine.
	for range activityLogBatchSize {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		r.ServeHTTP(w, req)
	}

	close(stopCh)
	time.Sleep(300 * time.Millisecond)

	if repo.count() == 0 {
		t.Error("expected entries after the panic was recovered; writer goroutine appears dead")
	}
}
