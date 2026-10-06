package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/payminto/payminto/backend/internal/observability"
)

// RequestIDHeader is the canonical header name for the request ID.
const RequestIDHeader = "X-Request-ID"

// RequestID injects a request ID into the Gin context and response headers.
// It reuses an incoming X-Request-ID header when present, otherwise generates
// a fresh UUID. The ID is also stashed on the request context so downstream
// observability.FromContext logs can pick it up.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		c.Set("requestID", id)
		c.Writer.Header().Set(RequestIDHeader, id)

		ctx := observability.WithRequestID(c.Request.Context(), id)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
