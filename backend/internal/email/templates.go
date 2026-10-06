// Package email provides embedded email templates and rendering utilities
// for the Payminto email pipeline.
package email

import "embed"

// FS holds all compiled-in email templates. Imported by EmailService at boot.
//
//go:embed templates/*.tmpl
var FS embed.FS
