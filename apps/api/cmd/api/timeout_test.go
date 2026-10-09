package main

import (
	"bufio"
	"bytes"
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
)

type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) { s.mu.Lock(); defer s.mu.Unlock(); return s.b.Write(p) }
func (s *syncBuf) String() string              { s.mu.Lock(); defer s.mu.Unlock(); return s.b.String() }

// trickle serves a JSON-decoding handler on the real server assembly with ReadTimeout 1 s, sends the
// headers, then a JSON body one byte per 100 ms for about 3 s, and returns the response status line.
func trickle(t *testing.T, extend time.Duration) (status string, log string) {
	t.Helper()
	var buf syncBuf
	api := httpx.H(func(w http.ResponseWriter, r *http.Request) error {
		if extend > 0 {
			if err := http.NewResponseController(w).SetReadDeadline(time.Now().Add(extend)); err != nil {
				return err
			}
		}
		var v struct {
			A string `json:"a"`
		}
		if err := httpx.DecodeJSON(r, &v); err != nil {
			return err
		}
		httpx.WriteJSON(w, 200, v)
		return nil
	})
	h := newHandler(slog.New(slog.NewJSONHandler(&buf, nil)), nil, func(context.Context) error { return nil }, api, nil)
	srv := httptest.NewUnstartedServer(h)
	srv.Config.ReadTimeout = time.Second
	srv.Start()
	defer srv.Close()

	c, err := net.Dial("tcp", srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	body := `{"a":"` + strings.Repeat("x", 25) + `"}`
	c.Write([]byte("POST /api/v1/x HTTP/1.1\r\nHost: x\r\nContent-Type: application/json\r\nContent-Length: " + strconv.Itoa(len(body)) + "\r\n\r\n"))
	for i := 0; i < len(body); i++ {
		if _, err := c.Write([]byte{body[i]}); err != nil {
			break // the server closed the connection
		}
		time.Sleep(100 * time.Millisecond)
	}
	c.SetReadDeadline(time.Now().Add(3 * time.Second))
	line, _ := bufio.NewReader(c).ReadString('\n') // empty when the connection was closed without a response
	return strings.TrimSpace(line), buf.String()
}

func TestReadTimeoutClosesTrickledBody(t *testing.T) {
	status, log := trickle(t, 0)
	if status != "" && !strings.Contains(status, "499") {
		t.Errorf("status line %q", status)
	}
	if !strings.Contains(log, `"status":499`) {
		t.Errorf("request log lacks 499: %s", log)
	}
}

func TestExtendedReadDeadlineAcceptsTrickledBody(t *testing.T) {
	status, log := trickle(t, 10*time.Second)
	if !strings.Contains(status, "200") || !strings.Contains(log, `"status":200`) {
		t.Errorf("status %q log %s", status, log)
	}
}
