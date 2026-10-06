package service

import (
	"bytes"
	"fmt"
	"log"
	"text/template"

	"github.com/payminto/payminto/backend/internal/email"
	"github.com/payminto/payminto/backend/internal/email/transport"
)

// EmailMessage contains all information needed to send one email.
type EmailMessage struct {
	To       string
	Subject  string
	Body     string
	Template string         // "welcome", "otp", "password_reset", etc.
	Data     map[string]any // template variables
}

// EmailService renders templates and sends emails via SMTP. Templates are
// loaded from internal/email/templates/ as Go text/template files at
// construction time and kept in an in-memory map for zero-latency rendering.
//
// The Consumer worker calls Send — never called directly from handlers.
type EmailService struct {
	templates map[string]*template.Template
	sender    transport.Sender
}

// NewEmailService loads all email templates at startup and returns an
// EmailService. sender delivers rendered messages; pass transport.New(cfg) —
// a NoopSender is used automatically when SMTP is not configured. A nil sender
// defaults to NoopSender so existing callers/tests keep working.
//
// Template parse errors are logged as warnings — a missing template will cause
// Send to return an error at delivery time, not a startup panic.
func NewEmailService(sender transport.Sender) *EmailService {
	if sender == nil {
		sender = &transport.NoopSender{}
	}
	svc := &EmailService{
		templates: make(map[string]*template.Template),
		sender:    sender,
	}

	entries, err := email.FS.ReadDir("templates")
	if err != nil {
		log.Printf("[email] warn: failed to list templates: %v", err)
		return svc
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		path := "templates/" + name
		content, err := email.FS.ReadFile(path)
		if err != nil {
			log.Printf("[email] warn: failed to read template %s: %v", name, err)
			continue
		}
		key := templateKey(name)
		tmpl, err := template.New(key).Parse(string(content))
		if err != nil {
			log.Printf("[email] warn: failed to parse template %s: %v", name, err)
			continue
		}
		svc.templates[key] = tmpl
	}
	return svc
}

// Render executes a named template with data, returning the rendered body.
func (s *EmailService) Render(templateName string, data map[string]any) (string, error) {
	tmpl, ok := s.templates[templateName]
	if !ok {
		return "", fmt.Errorf("email template %q not found", templateName)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render template %q: %w", templateName, err)
	}
	return buf.String(), nil
}

// Send renders (if needed) and delivers an email message via the configured
// transport. When SMTP is not configured the transport is a NoopSender that
// logs instead of sending — so this is safe in every environment.
func (s *EmailService) Send(msg EmailMessage) error {
	body := msg.Body
	if body == "" && msg.Template != "" {
		rendered, err := s.Render(msg.Template, msg.Data)
		if err != nil {
			return fmt.Errorf("email render: %w", err)
		}
		body = rendered
	}
	return s.sender.Send(msg.To, msg.Subject, body)
}

// templateKey converts a filename like "welcome.tmpl" → "welcome".
func templateKey(filename string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[:i]
		}
	}
	return filename
}
