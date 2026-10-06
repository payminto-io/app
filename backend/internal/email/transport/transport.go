// Package transport provides pluggable email delivery backends for the email
// service. The Sender interface decouples message rendering (in the service)
// from delivery, so the same service can log in development and send via SMTP
// in production without code changes.
package transport

import (
	"fmt"
	"log"
	"net/smtp"
)

// Sender delivers a rendered email. Implementations must be safe for concurrent
// use by the email consumer worker.
type Sender interface {
	Send(to, subject, body string) error
}

// Config holds SMTP connection settings. When Host is empty, New returns a
// NoopSender so development and tests never attempt real delivery.
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// Enabled reports whether SMTP delivery is configured.
func (c Config) Enabled() bool { return c.Host != "" && c.Port != 0 && c.From != "" }

// New selects a Sender from config: a real SMTPSender when configured, else a
// NoopSender that logs (the safe default for dev/test).
func New(cfg Config) Sender {
	if cfg.Enabled() {
		return &SMTPSender{cfg: cfg}
	}
	return &NoopSender{}
}

// NoopSender logs the recipient and subject without sending. Used when SMTP is
// not configured.
type NoopSender struct{}

// Send implements Sender.
func (n *NoopSender) Send(to, subject, _ string) error {
	log.Printf("[email:noop] to=%s subject=%q (SMTP not configured)", to, subject)
	return nil
}

// SMTPSender delivers email over SMTP with PLAIN auth and STARTTLS as offered
// by the server (handled by smtp.SendMail).
type SMTPSender struct {
	cfg Config
}

// Send implements Sender. It composes a minimal RFC 5322 message and delivers
// it via smtp.SendMail.
func (s *SMTPSender) Send(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}
	msg := buildMessage(s.cfg.From, to, subject, body)
	if err := smtp.SendMail(addr, auth, s.cfg.From, []string{to}, msg); err != nil {
		return fmt.Errorf("smtp send to %s: %w", to, err)
	}
	return nil
}

// buildMessage assembles RFC 5322 headers + body as a byte slice.
func buildMessage(from, to, subject, body string) []byte {
	headers := "From: " + from + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n"
	return []byte(headers + body)
}
