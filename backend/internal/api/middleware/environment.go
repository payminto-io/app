package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/environment"
)

// Environment tags every request context with the process environment so services can resolve it.
func Environment(env environment.Environment) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request = c.Request.WithContext(environment.WithContext(c.Request.Context(), env))
		c.Next()
	}
}
