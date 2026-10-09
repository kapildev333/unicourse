package main

import (
	"bufio"
	"bytes"
	"context"
	"log/slog"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/blob"
	"github.com/kapildev333/unicourse/apps/api/internal/content"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
)

type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) { s.mu.Lock(); defer s.mu.Unlock(); return s.b.Write(p) }
func (s *syncBuf) String() string              { s.mu.Lock(); defer s.mu.Unlock(); return s.b.String() }

// trickle serves api on the real server assembly with ReadTimeout 1 s, sends the headers, then the body
// in 30 chunks 100 ms apart (about 3 s), and returns the response status line and the request log.
func trickle(t *testing.T, api http.Handler, contentType, body string) (status string, log string) {
	t.Helper()
	var buf syncBuf
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
	c.Write([]byte("POST /api/v1/x HTTP/1.1\r\nHost: x\r\nContent-Type: " + contentType + "\r\nContent-Length: " + strconv.Itoa(len(body)) + "\r\n\r\n"))
	step := len(body)/30 + 1
	for i := 0; i < len(body); i += step {
		if _, err := c.Write([]byte(body[i:min(i+step, len(body))])); err != nil {
			break // the server closed the connection
		}
		time.Sleep(100 * time.Millisecond)
	}
	c.SetReadDeadline(time.Now().Add(3 * time.Second))
	line, _ := bufio.NewReader(c).ReadString('\n') // empty when the connection was closed without a response
	return strings.TrimSpace(line), buf.String()
}

func jsonAPI(extend time.Duration) http.Handler {
	return httpx.H(func(w http.ResponseWriter, r *http.Request) error {
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
}

// uploadAPI runs the real content.ReceiveFile with the given upload timeout.
func uploadAPI(t *testing.T, timeout time.Duration) http.Handler {
	files := blob.Dir{Root: t.TempDir(), MaxBytes: 1 << 20}
	return httpx.H(func(w http.ResponseWriter, r *http.Request) error {
		up, err := content.ReceiveFile(files, timeout, "00000000-0000-0000-0000-000000000001", w, r, nil)
		if err != nil {
			return err
		}
		up.Temp.Remove()
		httpx.WriteJSON(w, 201, map[string]string{"fileName": up.FileName})
		return nil
	})
}

func multipartPDF(t *testing.T) (contentType, body string) {
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	fw, err := mw.CreateFormFile("file", "a.pdf")
	if err != nil {
		t.Fatal(err)
	}
	fw.Write([]byte("%PDF-1.4\n" + strings.Repeat("x", 600)))
	mw.Close()
	return mw.FormDataContentType(), b.String()
}

func TestReadTimeoutClosesTrickledBody(t *testing.T) {
	status, log := trickle(t, jsonAPI(0), "application/json", `{"a":"`+strings.Repeat("x", 600)+`"}`)
	if status != "" && !strings.Contains(status, "499") {
		t.Errorf("status line %q", status)
	}
	if !strings.Contains(log, `"status":499`) {
		t.Errorf("request log lacks 499: %s", log)
	}
}

func TestReadTimeoutClosesTrickledUpload(t *testing.T) {
	ct, body := multipartPDF(t)
	status, log := trickle(t, uploadAPI(t, 0), ct, body)
	if status != "" && !strings.Contains(status, "499") {
		t.Errorf("status line %q", status)
	}
	if !strings.Contains(log, `"status":499`) {
		t.Errorf("request log lacks 499: %s", log)
	}
}

func TestUploadTimeoutExtendsReadDeadline(t *testing.T) {
	ct, body := multipartPDF(t)
	status, log := trickle(t, uploadAPI(t, 10*time.Second), ct, body)
	if !strings.Contains(status, "201") || !strings.Contains(log, `"status":201`) {
		t.Errorf("status %q log %s", status, log)
	}
}
