package links

import (
	"crypto/rand"
	"fmt"
	"regexp"
)

const (
	shortCodeAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	// ShortCodeLength gives 62^12, about 71 bits: unguessable by enumeration.
	ShortCodeLength  = 12
	shortCodeRetries = 5
)

var shortCodePattern = regexp.MustCompile(`^[A-Za-z0-9]{8,32}$`)

// ValidShortCode reports whether s could be a short code; anything else is refused before a lookup.
func ValidShortCode(s string) bool { return shortCodePattern.MatchString(s) }

// NewShortCode draws ShortCodeLength characters from crypto/rand with rejection sampling (no modulo bias).
func NewShortCode() (string, error) {
	out := make([]byte, 0, ShortCodeLength)
	buf := make([]byte, ShortCodeLength*2)
	// 248 is the largest multiple of 62 that fits a byte; bytes at or above it are redrawn.
	const limit = 256 - 256%len(shortCodeAlphabet)
	for len(out) < ShortCodeLength {
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("links: short code entropy: %w", err)
		}
		for _, b := range buf {
			if int(b) < limit && len(out) < ShortCodeLength {
				out = append(out, shortCodeAlphabet[int(b)%len(shortCodeAlphabet)])
			}
		}
	}
	return string(out), nil
}
