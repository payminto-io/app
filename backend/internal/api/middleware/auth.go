package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/service"
)

// APIKeyAuth returns a Gin middleware that authenticates requests via either
// the X-API-Key header or an "Authorization: Bearer <key>" header. On success
// it injects "apiKey", "memberID", and "externalPlatformID" into the context.
func APIKeyAuth(authSvc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-API-Key")
		if key == "" {
			auth := c.GetHeader("Authorization")
			if token, ok := strings.CutPrefix(auth, "Bearer "); ok {
				key = token
			}
		}
		if key == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
			return
		}

		apiKey, err := authSvc.ValidateAPIKey(key)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		if !setAPIKeyIdentity(c, apiKey) {
			return
		}
		c.Next()
	}
}

// JWTOrAPIKey returns a Gin middleware that accepts either a JWT Bearer
// token or an X-API-Key header. It is used on merchant route groups so that
// both the human-facing dashboard (JWT) and server-to-server integrations
// (API key) can reach the same endpoints without duplicating handlers.
//
// Precedence:
//  1. If X-API-Key is present, validate as an API key.
//  2. Otherwise, if Authorization is "Bearer <token>", try JWT first, then
//     fall back to API-key validation on the same Bearer value (legacy
//     integrations sent their API key via Bearer too).
//
// On success it injects the same context keys the individual middlewares do:
// "memberID", "externalPlatformID", plus ("apiKey") when authenticated via
// an API key, or ("email", "memberType") when authenticated via JWT.
func JWTOrAPIKey(authSvc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if key := c.GetHeader("X-API-Key"); key != "" {
			apiKey, err := authSvc.ValidateAPIKey(key)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				return
			}
			if !setAPIKeyIdentity(c, apiKey) {
				return
			}
			c.Next()
			return
		}

		auth := c.GetHeader("Authorization")
		tokenStr, ok := strings.CutPrefix(auth, "Bearer ")
		if !ok || tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "JWT or API key required"})
			return
		}

		if claims, err := authSvc.ValidateJWT(tokenStr); err == nil {
			c.Set("memberID", claims.MemberID)
			c.Set("email", claims.Email)
			c.Set("memberType", claims.MemberType)
			c.Set("externalPlatformID", claims.ExternalPlatformID)
			c.Next()
			return
		}

		// Legacy: some integrations sent the API key via the Bearer header.
		if apiKey, err := authSvc.ValidateAPIKey(tokenStr); err == nil {
			if !setAPIKeyIdentity(c, apiKey) {
				return
			}
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
	}
}

func setAPIKeyIdentity(c *gin.Context, apiKey *models.APIKey) bool {
	if apiKey == nil || apiKey.MemberID == nil || *apiKey.MemberID == 0 || apiKey.ExternalPlatformID == 0 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "API key identity is incomplete"})
		return false
	}
	c.Set("apiKey", apiKey)
	c.Set("memberID", *apiKey.MemberID)
	c.Set("externalPlatformID", apiKey.ExternalPlatformID)
	return true
}

// JWTAuth returns a Gin middleware that validates a JWT Bearer token. On
// success it injects "memberID", "email", "memberType", and
// "externalPlatformID" into the Gin context.
func JWTAuth(authSvc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		tokenStr, ok := strings.CutPrefix(auth, "Bearer ")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Bearer token required"})
			return
		}

		claims, err := authSvc.ValidateJWT(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		c.Set("memberID", claims.MemberID)
		c.Set("email", claims.Email)
		c.Set("memberType", claims.MemberType)
		c.Set("externalPlatformID", claims.ExternalPlatformID)
		c.Next()
	}
}
