package outbox

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"strings"
	"testing"
	"time"
)

func TestLogMailer(t *testing.T) {
	var buf bytes.Buffer
	m := LogMailer{L: slog.New(slog.NewJSONHandler(&buf, nil))}
	if err := m.Send(context.Background(), "a@a.edu", "Hi", "click\nhttps://x.test/p?q=1\nbye"); err != nil {
		t.Fatal(err)
	}
	var line struct{ Msg, To, Subject, Text string }
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil || line.Msg != "mail" || line.To != "a@a.edu" || line.Subject != "Hi" ||
		line.Text != "click\nhttps://x.test/p?q=1\nbye" {
		t.Fatalf("log line %q (%v)", buf.String(), err)
	}
}

func TestSMTPMailerURL(t *testing.T) {
	if _, err := NewSMTPMailer("smtp://h:25"); err == nil {
		t.Fatal("missing from accepted")
	}
	if _, err := NewSMTPMailer("smtps://h:465?from=a@b.edu"); err == nil {
		t.Fatal("smtps accepted")
	}
	if _, err := NewSMTPMailer("smtp://h?from=a@b.edu"); err == nil {
		t.Fatal("missing port accepted")
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

func newTestMailer(t *testing.T, addr string) *smtpMailer {
	t.Helper()
	m, err := NewSMTPMailer("smtp://" + addr + "?from=noreply@x.edu")
	if err != nil {
		t.Fatal(err)
	}
	sm := m.(*smtpMailer)
	sm.dialTimeout, sm.ioTimeout = 200*time.Millisecond, 200*time.Millisecond
	return sm
}

func TestSMTPMailerSilentServerTimesOut(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			defer c.Close()
		}
	}()
	start := time.Now()
	if err := newTestMailer(t, ln.Addr().String()).Send(context.Background(), "a@a.edu", "S", "x"); err == nil {
		t.Fatal("want error")
	}
	if d := time.Since(start); d > 3*time.Second {
		t.Fatalf("took %v", d)
	}
}

func TestSMTPMailerDelivers(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	got := make(chan string, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		r := bufio.NewReader(c)
		say := func(s string) { c.Write([]byte(s + "\r\n")) }
		say("220 fake")
		var data strings.Builder
		inData := false
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")
			switch {
			case inData && line == ".":
				inData = false
				got <- data.String()
				say("250 ok")
			case inData:
				data.WriteString(line + "\n")
			case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
				say("250 fake")
			case strings.HasPrefix(line, "DATA"):
				inData = true
				say("354 go")
			case strings.HasPrefix(line, "QUIT"):
				say("221 bye")
				return
			default:
				say("250 ok")
			}
		}
	}()
	m := newTestMailer(t, ln.Addr().String())
	m.ioTimeout = 5 * time.Second
	if err := m.Send(context.Background(), "a@a.edu", "Hello\r\nBcc: evil@x", "l1\nl2"); err != nil {
		t.Fatal(err)
	}
	msg := <-got
	for _, want := range []string{"From: noreply@x.edu\n", "To: a@a.edu\n", "Subject: HelloBcc: evil@x\n", "\nl1\nl2\n"} {
		if !strings.Contains(msg, want) {
			t.Errorf("missing %q in %q", want, msg)
		}
	}
	if strings.Contains(msg, "\nBcc:") {
		t.Fatal("header injection")
	}
}
