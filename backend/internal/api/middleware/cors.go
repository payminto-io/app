package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS returns a Gin middleware applying CORS headers.
//
// allowedOrigins controls cross-origin access:
//   - empty slice  → reflect any origin WITHOUT credentials (safe wildcard;
//     bearer-token API calls work, cookies are not sent cross-origin).
//   - non-empty    → only the listed origins are allowed, WITH credentials,
//     so cookie-based cross-origin flows work for trusted dashboards.
//
// This avoids the spec-violating combination of `Access-Control-Allow-Origin: *`
// together with `Access-Control-Allow-Credentials: true`, which browsers reject.
func CORS(allowedOrigins ...string) gin.HandlerFunc {
	cfg := cors.Config{
		AllowMethods:  []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:  []string{"Origin", "Content-Type", "Authorization", "X-API-Key"},
		ExposeHeaders: []string{"Content-Length"},
	}
	if len(allowedOrigins) == 0 {
		// Public API surface: allow any origin but never with credentials.
		cfg.AllowOriginFunc = func(string) bool { return true }
		cfg.AllowCredentials = false
	} else {
		cfg.AllowOrigins = allowedOrigins
		cfg.AllowCredentials = true
	}
	return cors.New(cfg)
}
