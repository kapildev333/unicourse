package httpx

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strings"
)

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// DecodeJSON reads a body of at most 1 MB, rejecting unknown fields.
func DecodeJSON(r *http.Request, dst any) error {
	var raw bytes.Buffer
	dec := json.NewDecoder(io.TeeReader(http.MaxBytesReader(nil, r.Body, 1<<20), &raw))
	dec.DisallowUnknownFields()
	err := dec.Decode(dst)
	if err == nil {
		// A second value (or garbage) after the first is an error; io.EOF means clean end.
		if err = dec.Decode(&struct{}{}); err != io.EOF {
			return Validation(FieldError{Field: "body", Message: "must contain a single JSON value"})
		}
		return exactKeys(raw.Bytes(), dst)
	}
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return err // WriteError answers 499
	}
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		return &Error{Status: 400, Code: "validation_failed", Message: "body too large", Details: []FieldError{}}
	}
	if errors.Is(err, io.EOF) {
		return Validation(FieldError{Field: "body", Message: "body is required"})
	}
	if rest, ok := strings.CutPrefix(err.Error(), "json: unknown field "); ok {
		return Validation(FieldError{Field: strings.Trim(rest, `"`), Message: "unknown field"})
	}
	var ute *json.UnmarshalTypeError
	if errors.As(err, &ute) {
		field := ute.Field
		if field == "" {
			field = "body"
		}
		return Validation(FieldError{Field: field, Message: "invalid value"})
	}
	return Validation(FieldError{Field: "body", Message: "invalid JSON"})
}

// exactKeys rejects top-level keys that are not byte-equal to a json tag of dst, because
// encoding/json matches keys case-insensitively. Non-struct dst is skipped.
// ponytail: top level only; nested objects still match case-insensitively. Recurse if a contract needs it. Embedded structs: VisibleFields also admits the embedded type's own name and ignores json tag dominance; no request struct embeds today.
func exactKeys(body []byte, dst any) error {
	t := reflect.TypeOf(dst)
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == nil || t.Kind() != reflect.Struct {
		return nil
	}
	allowed := map[string]bool{}
	for _, f := range reflect.VisibleFields(t) {
		if !f.IsExported() {
			continue
		}
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if f.Tag.Get("json") == "-" {
			continue
		}
		if name == "" {
			name = f.Name
		}
		allowed[name] = true
	}
	var keys map[string]json.RawMessage
	if json.Unmarshal(body, &keys) != nil {
		return nil // not an object (or null); the decode above already accepted it
	}
	names := make([]string, 0, len(keys))
	for k := range keys {
		names = append(names, k)
	}
	sort.Strings(names) // deterministic first offender
	for _, k := range names {
		if !allowed[k] {
			return Validation(FieldError{Field: k, Message: "unknown field"})
		}
	}
	return nil
}
