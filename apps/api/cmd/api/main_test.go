package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kapildev333/unicourse/apps/api/internal/app"
)

func TestUnknownRoutesUseEnvelope(t *testing.T) {
	h := newHandler(slog.New(slog.NewJSONHandler(io.Discard, nil)), func(context.Context) error { return errors.New("down") }, nil)
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
	h := newHandler(slog.New(slog.NewJSONHandler(io.Discard, nil)), func(context.Context) error { return nil }, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/readyz", nil))
	if rec.Code != http.StatusOK {
		t.Fatal(rec.Code, rec.Body)
	}
}

func TestAPIMountedUnderV1(t *testing.T) {
	h := newHandler(slog.New(slog.NewJSONHandler(io.Discard, nil)), func(context.Context) error { return nil }, app.NewRouter(app.Deps{}))
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
