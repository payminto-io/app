package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func TestLocalLimiterWindows(t *testing.T) {
	now := time.Unix(0, 0)
	l := NewLocalLimiter(2, time.Minute)
	l.now = func() time.Time { return now }
	for i, want := range []bool{true, true, false} {
		if ok, _ := l.Allow("a"); ok != want {
			t.Fatalf("request %d: %v", i, ok)
		}
	}
	if ok, _ := l.Allow("b"); !ok {
		t.Fatal("another key shares a's budget")
	}
	now = now.Add(time.Minute)
	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("window did not reset")
	}
}

func strictRouter(rdb *redis.Client, limit int) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	_ = r.SetTrustedProxies(nil)
	r.GET("/x", RateLimitStrict(rdb, "t", limit, time.Minute), func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func hit(r *gin.Engine, xff string) int { return hitWith(r, "198.51.100.7:1234", xff) }

func hitFrom(r *gin.Engine, remote string) int { return hitWith(r, remote, "") }

func hitWith(r *gin.Engine, remote, xff string) int {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.RemoteAddr = remote
	if xff != "" {
		req.Header.Set("X-Forwarded-For", xff)
	}
	r.ServeHTTP(w, req)
	return w.Code
}

func TestRateLimitStrictFailsClosedWhenRedisIsDown(t *testing.T) {
	down := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 50 * time.Millisecond, MaxRetries: -1})
	defer down.Close()
	r := strictRouter(down, 2)
	codes := []int{hit(r, ""), hit(r, "1.1.1.1"), hit(r, "2.2.2.2")}
	if codes[0] != 200 || codes[1] != 200 || codes[2] != http.StatusTooManyRequests {
		t.Fatalf("codes %v: a Redis outage must degrade to the local limit, and X-Forwarded-For must not mint clients", codes)
	}
}
