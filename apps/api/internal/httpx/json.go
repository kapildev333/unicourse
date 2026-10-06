package httpx

import (
	"encoding/json"
	"errors"
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
		return nil
	}
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		return &Error{Status: 400, Code: "validation_failed", Message: "body too large", Details: []FieldError{}}
	}
	field, msg := "body", err.Error()
	if rest, ok := strings.CutPrefix(msg, "json: unknown field "); ok {
		field, msg = strings.Trim(rest, `"`), "unknown field"
	}
	var ute *json.UnmarshalTypeError
	if errors.As(err, &ute) && ute.Field != "" {
		field = ute.Field
	}
	return Validation(FieldError{Field: field, Message: msg})
}
