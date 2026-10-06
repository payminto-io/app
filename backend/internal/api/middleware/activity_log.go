package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
)

// sensitiveKeys is the set of JSON field names (all lowercase) that must never
// appear in the request body preview stored in the audit log. The lookup
// lowercases the incoming key before checking this map.
var sensitiveKeys = map[string]struct{}{
	"password":   {},
	"token":      {},
	"apikey":     {},
	"api_key":    {},
	"privatekey": {},
	"private_key": {},
	"mnemonic":   {},
	"secret":     {},
	"apisecret":  {},
	"passphrase": {},
}

const (
	// activityLogBufSize is the maximum number of log entries queued before
	// the background writer forces a flush.
	activityLogBufSize = 256
	// activityLogBatchSize is how many rows to insert per bulk operation.
	activityLogBatchSize = 100
	// activityLogFlushInterval is the maximum time between flushes even when
	// the buffer has fewer than activityLogBatchSize entries.
	activityLogFlushInterval = 5 * time.Second
	// maxBodyPreviewBytes limits the body snapshot stored in the DB.
	// Only this many bytes are read; the remainder is still forwarded to the handler.
	maxBodyPreviewBytes = 4096
)

// bodyCapturingWriter wraps gin.ResponseWriter so we can capture the response body.
type bodyCapturingWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyCapturingWriter) Write(b []byte) (int, error) {
	w.body.Write(b) //nolint:errcheck // capturing only; not returned
	return w.ResponseWriter.Write(b)
}

// ActivityLog returns a Gin middleware that records every request to the
// activity_logs table. Captures method, path, status, redacted body preview,
// IP, user agent, member ID, external platform ID, and duration.
//
// Logging is fully asynchronous: the middleware drops each entry on a buffered
// channel and a background goroutine performs bulk inserts every 100 rows or
// every 5 seconds—whichever comes first. The request path is never blocked.
//
// The channel and background goroutine are created once when this function is
// called. Provide a stopCh that is closed on server shutdown to drain the
// remaining entries before exit.
func ActivityLog(repo repository.ActivityLogRepository, stopCh <-chan struct{}) gin.HandlerFunc {
	ch := make(chan models.ActivityLog, activityLogBufSize)

	// flush inserts the accumulated buffer and resets it. It is called inside
	// the writer goroutine loop wrapped with its own panic recovery so a single
	// bad insert does not kill the goroutine permanently.
	flushBuf := func(buf *[]models.ActivityLog) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("activity_log flush panic (recovered): %v", r)
				// Reset the buffer so we don't retry the same poisoned batch.
				*buf = (*buf)[:0]
			}
		}()
		if len(*buf) == 0 {
			return
		}
		if err := repo.BulkCreate(*buf); err != nil {
			log.Printf("activity_log bulk insert error: %v", err)
		}
		*buf = (*buf)[:0]
	}

	// Background bulk-insert goroutine. The outer defer/recover only fires if
	// the loop infrastructure itself panics; per-iteration panics are caught by
	// flushBuf's own recovery so the loop continues.
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("activity_log writer goroutine panic: %v", r)
			}
		}()
		buf := make([]models.ActivityLog, 0, activityLogBatchSize)
		ticker := time.NewTicker(activityLogFlushInterval)
		defer ticker.Stop()

		for {
			select {
			case entry, ok := <-ch:
				if !ok {
					flushBuf(&buf)
					return
				}
				buf = append(buf, entry)
				if len(buf) >= activityLogBatchSize {
					flushBuf(&buf)
				}
			case <-ticker.C:
				flushBuf(&buf)
			case <-stopCh:
				// Drain remaining items from the channel before exiting.
			drain:
				for {
					select {
					case entry := <-ch:
						buf = append(buf, entry)
					default:
						break drain
					}
				}
				flushBuf(&buf)
				return
			}
		}
	}()

	return func(c *gin.Context) {
		start := time.Now()

		// Capture request body preview only for JSON content types, capped at
		// maxBodyPreviewBytes. multipart and octet-stream are skipped entirely.
		var reqPreview *string
		ct := c.Request.Header.Get("Content-Type")
		if c.Request.Body != nil && strings.HasPrefix(ct, "application/json") {
			limited := io.LimitReader(c.Request.Body, maxBodyPreviewBytes)
			previewBuf := &bytes.Buffer{}
			if _, err := previewBuf.ReadFrom(limited); err == nil {
				// Reattach: the preview bytes first, then whatever remains in
				// the original body (beyond the limit).
				c.Request.Body = io.NopCloser(io.MultiReader(
					bytes.NewReader(previewBuf.Bytes()),
					c.Request.Body,
				))
				preview := redactBody(previewBuf.Bytes())
				reqPreview = &preview
			}
		}

		// Wrap the response writer to capture response body.
		respBuf := &bytes.Buffer{}
		wrapped := &bodyCapturingWriter{ResponseWriter: c.Writer, body: respBuf}
		c.Writer = wrapped

		c.Next()

		durationMs := time.Since(start).Milliseconds()

		// Collect auth context injected by JWTAuth / APIKeyAuth.
		var memberID *uint
		var platformID *uint
		if mid, ok := c.Get("memberID"); ok {
			if v, ok := mid.(uint); ok && v != 0 {
				memberID = &v
			}
		}
		if pid, ok := c.Get("externalPlatformID"); ok {
			if v, ok := pid.(uint); ok && v != 0 {
				platformID = &v
			}
		}

		ip := c.ClientIP()
		ua := c.Request.UserAgent()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		entry := models.ActivityLog{
			MemberID:           memberID,
			ExternalPlatformID: platformID,
			Method:             c.Request.Method,
			Path:               path,
			StatusCode:         c.Writer.Status(),
			DurationMs:         durationMs,
			RequestBodyPreview: reqPreview,
			IPAddress:          &ip,
		}
		if ua != "" {
			entry.UserAgent = &ua
		}

		// Non-blocking send: drop if the channel is full rather than stalling.
		select {
		case ch <- entry:
		default:
			log.Printf("activity_log channel full, dropping entry for %s %s", c.Request.Method, path)
		}
	}
}

// redactBody walks the JSON tree and replaces sensitive key values with
// "[REDACTED]". Non-JSON bodies are truncated and returned as-is.
func redactBody(raw []byte) string {
	if len(raw) > maxBodyPreviewBytes {
		raw = raw[:maxBodyPreviewBytes]
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		// Not JSON — return raw bytes as string (already truncated).
		return string(raw)
	}
	redactMap(data)
	out, err := json.Marshal(data)
	if err != nil {
		return string(raw)
	}
	return string(out)
}

// redactMap recursively walks a JSON object and replaces sensitive values.
// All key lookups are lowercased before checking sensitiveKeys so that
// camelCase variants ("apiKey", "privateKey") match their lowercase entries.
func redactMap(m map[string]any) {
	for k, v := range m {
		lk := strings.ToLower(k)
		if _, sensitive := sensitiveKeys[lk]; sensitive {
			m[k] = "[REDACTED]"
			continue
		}
		switch child := v.(type) {
		case map[string]any:
			redactMap(child)
		case []any:
			redactSlice(child)
		}
	}
}

func redactSlice(s []any) {
	for _, item := range s {
		if m, ok := item.(map[string]any); ok {
			redactMap(m)
		}
	}
}
