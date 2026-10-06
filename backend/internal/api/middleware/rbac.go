package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/service"
)

// RequirePermission returns a Gin middleware that verifies the authenticated
// member holds the given permission on the current external platform.
//
// It expects the upstream JWTAuth or APIKeyAuth middleware to have already
// populated c.MustGet("memberID") and c.MustGet("externalPlatformID").
//
//   - Returns 401 when not authenticated (missing context values).
//   - Returns 403 when authenticated but lacking the permission.
//
// Usage:
//
//	adminGroup.Use(middleware.RequirePermission(mepRoleSvc, "system.admin"))
func RequirePermission(mepRoleSvc *service.MemberExternalPlatformRoleService, permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		memberIDRaw, ok := c.Get("memberID")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		platformIDRaw, ok := c.Get("externalPlatformID")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		memberID, ok := memberIDRaw.(uint)
		if !ok || memberID == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		platformID, ok := platformIDRaw.(uint)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		if !mepRoleSvc.HasPermission(c.Request.Context(), memberID, platformID, permission) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":      "forbidden",
				"permission": permission,
			})
			return
		}

		c.Next()
	}
}
