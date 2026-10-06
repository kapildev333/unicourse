package auth

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"net"
	"net/smtp"
	"net/url"
	"strings"
	"time"
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
	addr, from  string
	auth        smtp.Auth
	dialTimeout time.Duration // fields so tests can shorten them
	ioTimeout   time.Duration
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
	if from == "" || u.Hostname() == "" || u.Port() == "" {
		return nil, errors.New("smtp url needs host:port and ?from=")
	}
	m := &smtpMailer{addr: u.Host, from: from, dialTimeout: 10 * time.Second, ioTimeout: 30 * time.Second}
	if u.User != nil {
		pw, _ := u.User.Password()
		m.auth = smtp.PlainAuth("", u.User.Username(), pw, u.Hostname())
	}
	return m, nil
}

func (m *smtpMailer) Send(ctx context.Context, to, subject, text string) error {
	host, _, _ := net.SplitHostPort(m.addr)
	conn, err := (&net.Dialer{Timeout: m.dialTimeout}).DialContext(ctx, "tcp", m.addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	deadline := time.Now().Add(m.ioTimeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return err
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer c.Close()
	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: host}); err != nil {
			return err
		}
	}
	if m.auth != nil {
		if err := c.Auth(m.auth); err != nil {
			return err
		}
	}
	if err := c.Mail(m.from); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(buildMessage(m.from, to, subject, text)); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

func buildMessage(from, to, subject, text string) []byte {
	// strip CR/LF from header values so callers cannot inject headers
	h := strings.NewReplacer("\r", "", "\n", "")
	body := strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\n", "\r\n")
	return []byte("From: " + h.Replace(from) + "\r\nTo: " + h.Replace(to) + "\r\nSubject: " + h.Replace(subject) +
		"\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n" + body)
}
