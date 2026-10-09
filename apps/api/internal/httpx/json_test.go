package httpx

import (
	"errors"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestDecodeJSONUnknownField(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"a","bogus":1}`))
	var dst struct {
		Name string `json:"name"`
	}
	err := DecodeJSON(r, &dst)
	var e *Error
	if !errors.As(err, &e) || e.Code != "validation_failed" || len(e.Details) != 1 || e.Details[0].Field != "bogus" {
		t.Fatalf("%#v", err)
	}
}

func TestDecodeJSONTooLarge(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"`+strings.Repeat("a", 1<<20)+`"}`))
	var dst struct {
		Name string `json:"name"`
	}
	err := DecodeJSON(r, &dst)
	var e *Error
	if !errors.As(err, &e) || e.Code != "validation_failed" || e.Message != "body too large" {
		t.Fatalf("%#v", err)
	}
}

func TestDecodeJSONOK(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"a"}`))
	var dst struct {
		Name string `json:"name"`
	}
	if err := DecodeJSON(r, &dst); err != nil || dst.Name != "a" {
		t.Fatal(err, dst)
	}
}

func TestWriteJSONContentType(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteJSON(rec, 201, map[string]int{"a": 1})
	if rec.Code != 201 || rec.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatal(rec.Code, rec.Header())
	}
}

func decodeErr(t *testing.T, body string, dst any) *Error {
	t.Helper()
	var e *Error
	if err := DecodeJSON(httptest.NewRequest("POST", "/", strings.NewReader(body)), dst); !errors.As(err, &e) {
		t.Fatalf("%q: %#v", body, err)
	}
	return e
}

func TestDecodeJSONTrailingData(t *testing.T) {
	var dst struct{ A int }
	for _, b := range []string{`{"a":1} garbage`, `{"a":1}{"a":2}`} {
		if e := decodeErr(t, b, &dst); e.Code != "validation_failed" || e.Details[0].Field != "body" {
			t.Errorf("%q: %#v", b, e)
		}
	}
}

func TestDecodeJSONEmpty(t *testing.T) {
	var dst struct{ A int }
	if e := decodeErr(t, ``, &dst); e.Details[0].Field != "body" || e.Details[0].Message != "body is required" {
		t.Fatalf("%#v", e)
	}
}

func TestDecodeJSONTypeError(t *testing.T) {
	var dst struct {
		A int `json:"a"`
	}
	if e := decodeErr(t, `{"a":"x"}`, &dst); e.Details[0].Field != "a" || e.Details[0].Message != "invalid value" {
		t.Fatalf("%#v", e)
	}
}

func TestDecodeJSONExactCaseKeys(t *testing.T) {
	type dst struct {
		Name string `json:"name"`
		Opt  string `json:"opt,omitempty"`
		Skip string `json:"-"`
	}
	for _, c := range []struct{ body, bad string }{
		{`{"name":"x"}`, ""},
		{`{"opt":"x"}`, ""},
		{`{"NAME":"x"}`, "NAME"},
		{`{"Name":"x"}`, "Name"},
		{`{"OPT":"x"}`, "OPT"},
		{`{"-":"x"}`, "-"},
		{`{"Skip":"x"}`, "Skip"},
		{`{"skip":"x"}`, "skip"},
	} {
		err := DecodeJSON(httptest.NewRequest("POST", "/", strings.NewReader(c.body)), &dst{})
		var e *Error
		if c.bad == "" {
			if err != nil {
				t.Errorf("%s: %v", c.body, err)
			}
		} else if !errors.As(err, &e) || e.Code != "validation_failed" || len(e.Details) != 1 || e.Details[0].Field != c.bad || e.Details[0].Message != "unknown field" {
			t.Errorf("%s: %#v", c.body, err)
		}
	}
	// non-struct destinations skip the check
	var m map[string]string
	if err := DecodeJSON(httptest.NewRequest("POST", "/", strings.NewReader(`{"NAME":"x"}`)), &m); err != nil {
		t.Error(err)
	}
}

type stallAfter struct{ data *strings.Reader }

func (s stallAfter) Read(p []byte) (int, error) {
	if n, _ := s.data.Read(p); n > 0 {
		return n, nil
	}
	return 0, os.ErrDeadlineExceeded
}

func TestDecodeJSONDeadlineIsNotValidation(t *testing.T) {
	for _, body := range []string{`{"a"`, `{"a":"b"}`} { // stalls mid-value, and on the trailing check
		r := httptest.NewRequest("POST", "/", stallAfter{strings.NewReader(body)})
		var v struct {
			A string `json:"a"`
		}
		if err := DecodeJSON(r, &v); !errors.Is(err, os.ErrDeadlineExceeded) {
			t.Errorf("%s: %v", body, err)
		}
	}
}
