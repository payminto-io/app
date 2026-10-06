package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/metrics"
)

// Metrics records per-request Prometheus metrics (count + latency) labeled by
// the matched route template (not the raw path) to keep label cardinality
// bounded — e.g. "/api/v1/public/payment/:reference_id" rather than every UUID.
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		metrics.ObserveHTTP(c.Request.Method, route, c.Writer.Status(), time.Since(start))
	}
}
