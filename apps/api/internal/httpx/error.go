// Package httpx holds the shared HTTP plumbing: error envelope, JSON helpers,
// pagination, middleware and health endpoints.
package httpx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/kapildev333/unicourse/apps/api/internal/pg"
)

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type Error struct {
	Status  int
	Code    string
	Message string
	Details []FieldError
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func Unauthenticated() *Error {
	return &Error{Status: 401, Code: "unauthenticated", Message: "authentication required"}
}
func Forbidden() *Error { return &Error{Status: 403, Code: "forbidden", Message: "forbidden"} }
func NotFound(what string) *Error {
	return &Error{Status: 404, Code: "not_found", Message: what + " not found"}
}
func Conflict(msg string) *Error { return &Error{Status: 409, Code: "conflict", Message: msg} }
func RateLimited() *Error {
	return &Error{Status: 429, Code: "rate_limited", Message: "too many requests"}
}

func Validation(details ...FieldError) *Error {
	msg := fmt.Sprintf("%d fields are invalid", len(details))
	if len(details) == 1 {
		msg = "1 field is invalid"
	}
	return &Error{Status: 400, Code: "validation_failed", Message: msg, Details: details}
}

type Handler func(w http.ResponseWriter, r *http.Request) error

// H adapts a Handler; a returned error goes through WriteError.
func H(h Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			WriteError(w, r, err)
		}
	}
}

type envelope struct {
	Error struct {
		Code    string       `json:"code"`
		Message string       `json:"message"`
		Details []FieldError `json:"details"`
	} `json:"error"`
}

// WriteError renders err as the error envelope. Unknown errors become a generic
// 500 and are logged with the request id, which is also returned as X-Request-Id.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	var e *Error
	switch {
	case errors.Is(err, context.Canceled) && r.Context().Err() != nil:
		w.WriteHeader(499) // client closed the request; not a server fault, so no error log and no body
		return
	case errors.As(err, &e) && e != nil:
	case errors.Is(err, pg.ErrNotFound):
		e = &Error{Status: 404, Code: "not_found", Message: "not found"}
	case errors.Is(err, pg.ErrConflict):
		e = &Error{Status: 409, Code: "conflict", Message: "conflict"}
	default:
		id := RequestID(r.Context())
		slog.Error("internal error", "err", err, "requestId", id)
		if id != "" {
			w.Header().Set("X-Request-Id", id)
		}
		e = &Error{Status: 500, Code: "internal", Message: "internal error"}
	}
	var env envelope
	env.Error.Code, env.Error.Message, env.Error.Details = e.Code, e.Message, e.Details
	if env.Error.Details == nil {
		env.Error.Details = []FieldError{}
	}
	WriteJSON(w, e.Status, env)
}
