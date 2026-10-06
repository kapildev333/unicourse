// Package auth holds sign-in primitives: actor, sessions, magic links, mailer, rate limiter.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/mail"
	"strings"

	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
)

type Actor struct{ UserID, Email string }

type actorKey struct{}

func FromContext(ctx context.Context) (Actor, bool) {
	a, ok := ctx.Value(actorKey{}).(Actor)
	return a, ok
}

func withActor(ctx context.Context, a Actor) context.Context {
	return context.WithValue(ctx, actorKey{}, a)
}

// NormalizeEmail is the only email normaliser in the codebase.
func NormalizeEmail(s string) (string, error) {
	e := strings.ToLower(strings.TrimSpace(s))
	a, err := mail.ParseAddress(e)
	if err != nil || a.Address != e || strings.Count(e, "@") != 1 || strings.HasPrefix(e, "@") || strings.HasSuffix(e, "@") {
		return "", httpx.Validation(httpx.FieldError{Field: "email", Message: "must be a valid email address"})
	}
	return e, nil
}

// NewToken returns a 256-bit random token (43 URL-safe chars) and its raw SHA-256 hash.
func NewToken() (token string, hash []byte, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", nil, err
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, hashToken(token), nil
}

func hashToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

// validToken reports whether s is shaped like a NewToken result; malformed input is treated as unknown without querying.
func validToken(s string) bool {
	if len(s) != 43 {
		return false
	}
	b, err := base64.RawURLEncoding.DecodeString(s)
	return err == nil && len(b) == 32 && base64.RawURLEncoding.EncodeToString(b) == s
}
