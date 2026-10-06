package httpx

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestRequestIDMiddleware(t *testing.T) {
	var seen string
	h := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { seen = RequestID(r.Context()) }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(seen) || rec.Header().Get("X-Request-Id") != seen {
		t.Fatalf("generated %q header %q", seen, rec.Header().Get("X-Request-Id"))
	}
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Request-Id", "abc")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if seen != "abc" || rec.Header().Get("X-Request-Id") != "abc" {
		t.Fatal(seen)
	}
}

func logLine(t *testing.T, inner http.Handler, path string) map[string]any {
	t.Helper()
	var buf bytes.Buffer
	l := slog.New(slog.NewJSONHandler(&buf, nil))
	h := RequestIDMiddleware(LogMiddleware(l, inner))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", path, nil))
	if n := strings.Count(buf.String(), "\n"); n != 1 {
		t.Fatalf("want one line, got %d: %s", n, buf.String())
	}
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestLogMiddleware404(t *testing.T) {
	m := logLine(t, http.NotFoundHandler(), "/nope")
	if m["status"] != float64(404) || m["path"] != "/nope" || m["method"] != "GET" || m["requestId"] == "" {
		t.Fatal(m)
	}
	if _, ok := m["userId"]; ok {
		t.Fatal("userId must be absent when unset")
	}
}

// Inner handlers set ids after the outer log middleware has started.
func TestLogMiddlewareContextIDs(t *testing.T) {
	m := logLine(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WithUserID(r.Context(), "u1")
		WithUniversityID(r.Context(), "uni1")
	}), "/")
	if m["userId"] != "u1" || m["universityId"] != "uni1" || m["status"] != float64(200) {
		t.Fatal(m)
	}
}

func TestRecover(t *testing.T) {
	h := Recover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { panic("x") }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != 500 || !strings.Contains(rec.Body.String(), `"code":"internal"`) {
		t.Fatal(rec.Code, rec.Body)
	}
}
