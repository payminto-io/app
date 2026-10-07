package cre

import (
	"regexp"
	"strings"
)

var (
	urlPattern   = regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9+.-]*://[^\s"'<>)]+`)
	tokenPattern = regexp.MustCompile(`\b(0x)?[A-Fa-f0-9]{32,}\b|\b[A-Za-z0-9_-]{24,}\b`)
)

// Sanitize strips URLs (which embed RPC API keys) and token-shaped strings from an error or provider
// message before it reaches health, status JSON, the dashboard or a log line.
func Sanitize(msg string) string {
	msg = urlPattern.ReplaceAllString(msg, "[url]")
	msg = tokenPattern.ReplaceAllString(msg, "[redacted]")
	return strings.TrimSpace(msg)
}

// SanitizeError is Sanitize over err.Error(); nil yields "".
func SanitizeError(err error) string {
	if err == nil {
		return ""
	}
	return Sanitize(err.Error())
}
