package httpx

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	for path, want := range map[string]int{"/healthz": 200, "/readyz": 503} {
		rec := httptest.NewRecorder()
		Health(func(context.Context) error { return errors.New("down") }).ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != want {
			t.Errorf("%s: %d", path, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	Health(func(context.Context) error { return nil }).ServeHTTP(rec, httptest.NewRequest("GET", "/readyz", nil))
	if rec.Code != 200 {
		t.Fatal(rec.Code)
	}
}
