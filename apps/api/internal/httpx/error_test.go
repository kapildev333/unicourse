package httpx

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kapildev333/unicourse/apps/api/internal/pg"
)

func do(err error) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(withRequestID(r.Context(), "req-1"))
	WriteError(rec, r, err)
	return rec
}

func TestWriteErrorValidation(t *testing.T) {
	rec := do(Validation(FieldError{"endsOn", "must be after startsOn"}))
	want := `{"error":{"code":"validation_failed","message":"1 field is invalid","details":[{"field":"endsOn","message":"must be after startsOn"}]}}`
	if rec.Code != 400 || strings.TrimSpace(rec.Body.String()) != want {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if got := do(Validation(FieldError{"a", "x"}, FieldError{"b", "y"})).Body.String(); !strings.Contains(got, "2 fields are invalid") {
		t.Fatal(got)
	}
}

func TestWriteErrorInternal(t *testing.T) {
	rec := do(errors.New("boom"))
	want := `{"error":{"code":"internal","message":"internal error","details":[]}}`
	if rec.Code != 500 || strings.TrimSpace(rec.Body.String()) != want {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if rec.Header().Get("X-Request-Id") != "req-1" {
		t.Fatal("missing request id header")
	}
}

func TestWriteErrorMapping(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{NotFound("section"), 404, "not_found"},
		{Unauthenticated(), 401, "unauthenticated"},
		{Forbidden(), 403, "forbidden"},
		{Conflict("dup"), 409, "conflict"},
		{RateLimited(), 429, "rate_limited"},
		{pg.ErrNotFound, 404, "not_found"},
		{fmt.Errorf("wrapped: %w", pg.ErrConflict), 409, "conflict"},
	}
	for _, c := range cases {
		rec := do(c.err)
		if rec.Code != c.status || !strings.Contains(rec.Body.String(), `"code":"`+c.code+`"`) {
			t.Errorf("%v: %d %s", c.err, rec.Code, rec.Body)
		}
	}
	if got := do(NotFound("section")).Body.String(); !strings.Contains(got, `"message":"section not found"`) {
		t.Fatal(got)
	}
}

func TestH(t *testing.T) {
	rec := httptest.NewRecorder()
	H(func(w http.ResponseWriter, r *http.Request) error { return Forbidden() })(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != 403 {
		t.Fatal(rec.Code)
	}
}

func TestWriteErrorTypedNil(t *testing.T) {
	var e *Error
	rec := do(e)
	if rec.Code != 500 {
		t.Fatal(rec.Code, rec.Body)
	}
}
