package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimit returns a Gin middleware that limits each IP to `limit` requests
// per `window` using Redis INCR+EXPIRE. If rdb is nil the middleware is a
// no-op (useful in environments without Redis).
func RateLimit(rdb *redis.Client, limit int, window time.Duration) gin.HandlerFunc {
	return RateLimitScoped(rdb, "", limit, window)
}

// RateLimitScoped is RateLimit with its own counter per scope, so one route group cannot spend another's budget.
func RateLimitScoped(rdb *redis.Client, scope string, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rdb == nil {
			c.Next()
			return
		}
		key := rateLimitKey(scope, c.ClientIP())
		ctx := context.Background()

		count, err := incrWindow(ctx, rdb, key, window)
		if err != nil {
			c.Next()
			return
		}
		if count > int64(limit) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
				"code":  "rate_limit_exceeded",
			})
			return
		}
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", int64(limit)-count))
		c.Next()
	}
}

func rateLimitKey(scope, ip string) string {
	if scope == "" {
		return fmt.Sprintf("ratelimit:%s", ip)
	}
	return fmt.Sprintf("ratelimit:%s:%s", scope, ip)
}

// incrWindowScript counts and sets the TTL in one atomic step, and restores a TTL a key somehow lost.
var incrWindowScript = redis.NewScript(`
local count = redis.call('INCR', KEYS[1])
if redis.call('PTTL', KEYS[1]) < 0 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return count`)

// incrWindow counts one request in key's window; the key always expires.
func incrWindow(ctx context.Context, rdb *redis.Client, key string, window time.Duration) (int64, error) {
	return incrWindowScript.Run(ctx, rdb, []string{key}, window.Milliseconds()).Int64()
}
