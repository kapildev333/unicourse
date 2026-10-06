package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/smtp"
	"net/url"
	"strings"
)

type Mailer interface {
	Send(ctx context.Context, to, subject, text string) error
}

// LogMailer writes the mail to the log (dev, tests, e2e).
type LogMailer struct{ L *slog.Logger }

func (m LogMailer) Send(ctx context.Context, to, subject, text string) error {
	m.L.InfoContext(ctx, "mail", "to", to, "subject", subject, "text", text)
	return nil
}

type smtpMailer struct {
	addr, from string
	auth       smtp.Auth
}

// NewSMTPMailer parses smtp://user:pass@host:port?from=noreply@x.edu. STARTTLS is negotiated by net/smtp.
// ponytail: net/smtp is frozen and synchronous; move to a queue or provider API when volume or latency matters.
func NewSMTPMailer(rawURL string) (Mailer, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "smtp" {
		return nil, errors.New("smtp url must use the smtp scheme")
	}
	from := u.Query().Get("from")
	if from == "" || u.Host == "" {
		return nil, errors.New("smtp url needs a host and ?from=")
	}
	m := &smtpMailer{addr: u.Host, from: from}
	if u.User != nil {
		pw, _ := u.User.Password()
		m.auth = smtp.PlainAuth("", u.User.Username(), pw, u.Hostname())
	}
	return m, nil
}

func (m *smtpMailer) Send(_ context.Context, to, subject, text string) error {
	return smtp.SendMail(m.addr, m.auth, m.from, []string{to}, buildMessage(m.from, to, subject, text))
}

func buildMessage(from, to, subject, text string) []byte {
	// strip CR/LF from header values so callers cannot inject headers
	h := strings.NewReplacer("\r", "", "\n", "")
	body := strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\n", "\r\n")
	return []byte("From: " + h.Replace(from) + "\r\nTo: " + h.Replace(to) + "\r\nSubject: " + h.Replace(subject) +
		"\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n" + body)
}
