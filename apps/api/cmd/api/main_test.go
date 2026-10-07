package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kapildev333/unicourse/apps/api/internal/app"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
)

func TestUnknownRoutesUseEnvelope(t *testing.T) {
	h := newHandler(slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, func(context.Context) error { return errors.New("down") }, nil, nil)
	for _, c := range [][2]string{{"GET", "/api/v1/nope"}, {"POST", "/healthz"}} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(c[0], c[1], nil))
		if rec.Code != 404 || !strings.Contains(rec.Body.String(), `"code":"not_found"`) {
			t.Errorf("%v: %d %s", c, rec.Code, rec.Body)
		}
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatal(rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/readyz", nil))
	if rec.Code != 503 || !strings.Contains(rec.Body.String(), `"status":"unavailable"`) {
		t.Fatal(rec.Code, rec.Body)
	}
}

func TestReadyzReflectsDatabase(t *testing.T) {
	h := newHandler(slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, func(context.Context) error { return nil }, nil, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/readyz", nil))
	if rec.Code != http.StatusOK {
		t.Fatal(rec.Code, rec.Body)
	}
}

func TestAPIMountedUnderV1(t *testing.T) {
	h := newHandler(slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, func(context.Context) error { return nil }, app.NewRouter(app.Deps{}), nil)
	for _, c := range []struct {
		path string
		code int
		body string
	}{{"/api/v1/me", 401, `"code":"unauthenticated"`}, {"/api/v1/nope", 404, `"code":"not_found"`}} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", c.path, nil))
		if rec.Code != c.code || !strings.Contains(rec.Body.String(), c.body) {
			t.Errorf("%s: %d %s", c.path, rec.Code, rec.Body)
		}
	}
}

func TestWebAppMounted(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>app</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	web, err := httpx.Static(dir)
	if err != nil {
		t.Fatal(err)
	}
	h := newHandler(slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, func(context.Context) error { return nil }, app.NewRouter(app.Deps{}), web)
	for _, c := range []struct {
		path string
		code int
		body string
	}{{"/healthz", 200, `"status"`}, {"/api/v1/me", 401, `"code":"unauthenticated"`}, {"/api/v1/nope", 404, `"code":"not_found"`}, {"/api/other", 404, `"code":"not_found"`}, {"/a/me", 200, "<html>app</html>"}} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", c.path, nil))
		if rec.Code != c.code || !strings.Contains(rec.Body.String(), c.body) {
			t.Errorf("%s: %d %s", c.path, rec.Code, rec.Body)
		}
	}
}
