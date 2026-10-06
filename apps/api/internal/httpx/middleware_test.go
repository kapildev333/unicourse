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
	for _, bad := range []string{"has space", strings.Repeat("a", 65), "a\nb", "a/b"} {
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("X-Request-Id", bad)
		seen = ""
		h.ServeHTTP(httptest.NewRecorder(), r)
		if seen == bad || len(seen) != 32 {
			t.Fatalf("accepted %q -> %q", bad, seen)
		}
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

func TestFullChainPanic(t *testing.T) {
	var buf bytes.Buffer
	l := slog.New(slog.NewJSONHandler(&buf, nil))
	h := RequestIDMiddleware(LogMiddleware(l, Recover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { panic("x") }))))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	id := rec.Header().Get("X-Request-Id")
	if rec.Code != 500 || id == "" || !strings.Contains(rec.Body.String(), `"code":"internal"`) {
		t.Fatal(rec.Code, rec.Header(), rec.Body)
	}
	var found int
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatal(err)
		}
		if m["msg"] == "request" {
			found++
			if m["status"] != float64(500) || m["requestId"] != id {
				t.Fatal(m)
			}
		}
	}
	if found != 1 {
		t.Fatalf("request lines: %d", found)
	}
}

func TestRecoverRepanicsAbortHandler(t *testing.T) {
	defer func() {
		if recover() != http.ErrAbortHandler {
			t.Fatal("ErrAbortHandler must propagate")
		}
	}()
	Recover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { panic(http.ErrAbortHandler) })).
		ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
}

func TestRecover(t *testing.T) {
	h := Recover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { panic("x") }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != 500 || !strings.Contains(rec.Body.String(), `"code":"internal"`) {
		t.Fatal(rec.Code, rec.Body)
	}
}

func TestLogMiddlewareLogsRoutePatternNotParams(t *testing.T) {
	var buf bytes.Buffer
	rt := NewRouter()
	rt.Handle("GET", "/x/{id}", func(w http.ResponseWriter, r *http.Request) error { return nil })
	h := LogMiddleware(slog.New(slog.NewJSONHandler(&buf, nil)), rt)
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/x/secret-token", nil))
	if strings.Contains(buf.String(), "secret-token") || !strings.Contains(buf.String(), `"path":"/x/{id}"`) {
		t.Fatalf("log %s", buf.String())
	}
	buf.Reset()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/nope", nil))
	if !strings.Contains(buf.String(), `"path":"/nope"`) {
		t.Fatalf("unmatched log %s", buf.String())
	}
}
