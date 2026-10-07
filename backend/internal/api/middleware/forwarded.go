package middleware

import (
	"log/slog"
	"sync"

	"github.com/gin-gonic/gin"
)

// WarnUntrustedForwarding logs once when a request carries X-Forwarded-For from a peer outside TRUSTED_PROXIES:
// behind a proxy that means every client shares the proxy's IP and its rate limits (docs/OPERATIONS.md, "Client IPs").
func WarnUntrustedForwarding(warn func(peer string)) gin.HandlerFunc {
	var once sync.Once
	if warn == nil {
		warn = func(peer string) {
			slog.Warn("X-Forwarded-For received from an untrusted peer; every client behind it is treated as one IP. Set TRUSTED_PROXIES to the proxy's address.",
				"peer", peer)
		}
	}
	return func(c *gin.Context) {
		if c.GetHeader("X-Forwarded-For") != "" && c.ClientIP() == c.RemoteIP() {
			peer := c.RemoteIP()
			once.Do(func() { warn(peer) })
		}
		c.Next()
	}
}
