package httpx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type ctxKey int

const (
	requestIDKey ctxKey = iota
	logFieldsKey
)

// logFields is filled by inner layers (auth, tenant) and read by the outer LogMiddleware.
type logFields struct{ userID, universityID, route string }

func withRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// WithUserID records the authenticated user for the request log line. It mutates
// state placed in ctx by LogMiddleware, so the outer middleware sees it; it is a
// no-op when ctx has no such state.
func WithUserID(ctx context.Context, id string) {
	if f, ok := ctx.Value(logFieldsKey).(*logFields); ok {
		f.userID = id
	}
}

// WithUniversityID records the resolved university for the request log line; see WithUserID.
func WithUniversityID(ctx context.Context, id string) {
	if f, ok := ctx.Value(logFieldsKey).(*logFields); ok {
		f.universityID = id
	}
}

// setRoute records the matched route pattern (path part only, e.g. /api/v1/auth/magic-link/{token}) so
// the log line never carries path parameters such as secret tokens. No-op without LogMiddleware.
func setRoute(ctx context.Context, pattern string) {
	if f, ok := ctx.Value(logFieldsKey).(*logFields); ok {
		if i := strings.IndexByte(pattern, ' '); i >= 0 {
			pattern = pattern[i+1:]
		}
		f.route = pattern
	}
}

var validRequestID = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if !validRequestID.MatchString(id) {
			b := make([]byte, 16)
			_, _ = rand.Read(b)
			id = hex.EncodeToString(b)
		}
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(withRequestID(r.Context(), id)))
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(code int) {
	if s.status == 0 {
		s.status = code
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusWriter) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	return s.ResponseWriter.Write(b)
}

func (s *statusWriter) Unwrap() http.ResponseWriter { return s.ResponseWriter }

func LogMiddleware(l *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		f := &logFields{}
		sw := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r.WithContext(context.WithValue(r.Context(), logFieldsKey, f)))
		if sw.status == 0 {
			sw.status = http.StatusOK
		}
		path := r.URL.Path
		if f.route != "" {
			path = f.route
		}
		attrs := []any{
			"method", r.Method, "path", path, "status", sw.status,
			"durationMs", time.Since(start).Milliseconds(), "requestId", RequestID(r.Context()),
		}
		if f.userID != "" {
			attrs = append(attrs, "userId", f.userID)
		}
		if f.universityID != "" {
			attrs = append(attrs, "universityId", f.universityID)
		}
		l.Info("request", attrs...)
	})
}

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if v == http.ErrAbortHandler {
					panic(v)
				}
				slog.Error("panic", "panic", v, "requestId", RequestID(r.Context()))
				WriteError(w, r, &Error{Status: 500, Code: "internal", Message: "internal error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
