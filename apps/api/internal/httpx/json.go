package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// DecodeJSON reads a body of at most 1 MB, rejecting unknown fields.
func DecodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	err := dec.Decode(dst)
	if err == nil {
		// A second value (or garbage) after the first is an error; io.EOF means clean end.
		if err = dec.Decode(&struct{}{}); err == io.EOF {
			return nil
		}
		return Validation(FieldError{Field: "body", Message: "must contain a single JSON value"})
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
