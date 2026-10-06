package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"regexp"
	"strings"
	"testing"
)

var urlRe = regexp.MustCompile(`https?://\S+`)

// lastURL applies the design §8 parse rule: the URL in the last "mail" log line.
func lastURL(t *testing.T, buf *bytes.Buffer) string {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		var m struct{ Msg, To, Text string }
		if json.Unmarshal([]byte(lines[i]), &m) == nil && m.Msg == "mail" {
			if u := urlRe.FindString(m.Text); u != "" {
				return u
			}
		}
	}
	t.Fatal("no mail line with a URL")
	return ""
}

func TestLogMailer(t *testing.T) {
	var buf bytes.Buffer
	m := LogMailer{L: slog.New(slog.NewJSONHandler(&buf, nil))}
	if err := m.Send(context.Background(), "a@a.edu", "Hi", "click\nhttps://x.test/p?q=1\nbye"); err != nil {
		t.Fatal(err)
	}
	if u := lastURL(t, &buf); u != "https://x.test/p?q=1" {
		t.Fatalf("url %q", u)
	}
}

func TestSMTPMailerURL(t *testing.T) {
	if _, err := NewSMTPMailer("smtp://h:25"); err == nil {
		t.Fatal("missing from accepted")
	}
	if _, err := NewSMTPMailer("smtps://h:465?from=a@b.edu"); err == nil {
		t.Fatal("smtps accepted")
	}
	if _, err := NewSMTPMailer("smtp://u:p@h:25?from=a@b.edu"); err != nil {
		t.Fatal(err)
	}
	msg := string(buildMessage("n@x.edu", "a@a.edu", "Sub", "l1\nl2"))
	for _, want := range []string{"From: n@x.edu\r\n", "To: a@a.edu\r\n", "Subject: Sub\r\n", "MIME-Version: 1.0\r\n",
		"Content-Type: text/plain; charset=utf-8\r\n\r\nl1\r\nl2"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message lacks %q:\n%q", want, msg)
		}
	}
}
