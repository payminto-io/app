package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// LocalLimiter is a fixed-window per-key counter in process memory: the floor when Redis is absent or failing.
type LocalLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	now     func() time.Time
	buckets map[string]localBucket
}

type localBucket struct {
	start time.Time
	count int
}

// maxLocalKeys bounds memory; when full, expired buckets are swept and, failing that, new keys are refused.
const maxLocalKeys = 100000

func NewLocalLimiter(limit int, window time.Duration) *LocalLimiter {
	return &LocalLimiter{limit: limit, window: window, now: time.Now, buckets: map[string]localBucket{}}
}

// Allow counts one request for key and reports whether it is within the limit, with what remains.
func (l *LocalLimiter) Allow(key string) (bool, int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	b, ok := l.buckets[key]
	if !ok || now.Sub(b.start) >= l.window {
		if !ok && len(l.buckets) >= maxLocalKeys {
			for k, v := range l.buckets {
				if now.Sub(v.start) >= l.window {
					delete(l.buckets, k)
				}
			}
			if len(l.buckets) >= maxLocalKeys {
				return false, 0
			}
		}
		b = localBucket{start: now}
	}
	b.count++
	l.buckets[key] = b
	return b.count <= l.limit, max(l.limit-b.count, 0)
}

// RateLimitStrict limits per client IP with Redis when it answers and with an in-process limiter when it
// does not; unlike RateLimit it never lets a request through unlimited.
func RateLimitStrict(rdb *redis.Client, scope string, limit int, window time.Duration) gin.HandlerFunc {
	local := NewLocalLimiter(limit, window)
	return func(c *gin.Context) {
		key := rateLimitKey(scope, c.ClientIP())
		allowed, remaining := false, 0
		redisOK := false
		if rdb != nil {
			ctx := c.Request.Context()
			count, err := rdb.Incr(ctx, key).Result()
			if err == nil {
				redisOK = true
				if count == 1 {
					rdb.Expire(ctx, key, window)
				}
				allowed, remaining = count <= int64(limit), max(limit-int(count), 0)
			}
		}
		if !redisOK {
			allowed, remaining = local.Allow(key)
		}
		if !allowed {
			c.Header("Retry-After", fmt.Sprintf("%d", int(window.Seconds())))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded", "code": "rate_limit_exceeded"})
			return
		}
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Next()
	}
}
