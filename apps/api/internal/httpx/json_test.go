package httpx

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSONUnknownField(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"a","bogus":1}`))
	var dst struct{ Name string }
	err := DecodeJSON(r, &dst)
	var e *Error
	if !errors.As(err, &e) || e.Code != "validation_failed" || len(e.Details) != 1 || e.Details[0].Field != "bogus" {
		t.Fatalf("%#v", err)
	}
}

func TestDecodeJSONTooLarge(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"`+strings.Repeat("a", 1<<20)+`"}`))
	var dst struct{ Name string }
	err := DecodeJSON(r, &dst)
	var e *Error
	if !errors.As(err, &e) || e.Code != "validation_failed" || e.Message != "body too large" {
		t.Fatalf("%#v", err)
	}
}

func TestDecodeJSONOK(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"a"}`))
	var dst struct{ Name string }
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
