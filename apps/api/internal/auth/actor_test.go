package auth

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
)

func TestNormalizeEmail(t *testing.T) {
	got, err := NormalizeEmail("  Alice@CS.Example.EDU ")
	if err != nil || got != "alice@cs.example.edu" {
		t.Fatalf("got %q, %v", got, err)
	}
	for _, bad := range []string{"", "a", "a@", "@b", "a@b@c", "Alice <a@b.edu>", strings.Repeat("a", 250) + "@b.edu"} {
		_, err := NormalizeEmail(bad)
		var he *httpx.Error
		if !errors.As(err, &he) || he.Code != "validation_failed" || len(he.Details) != 1 || he.Details[0].Field != "email" {
			t.Errorf("%q: want validation_failed on email, got %v", bad, err)
		}
	}
}

func TestNewToken(t *testing.T) {
	a, ha, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	b, _, _ := NewToken()
	if len(a) != 43 || a == b {
		t.Fatalf("token %q / %q", a, b)
	}
	if !bytes.Equal(hashToken(a), ha) || len(ha) != 32 {
		t.Fatal("hash mismatch")
	}
	if !validToken(a) || validToken("short") || validToken(a[:42]+"!") {
		t.Fatal("validToken")
	}
}

func TestActorContext(t *testing.T) {
	if _, ok := FromContext(t.Context()); ok {
		t.Fatal("unexpected actor")
	}
	a, ok := FromContext(withActor(t.Context(), Actor{UserID: "u", Email: "e"}))
	if !ok || a.UserID != "u" {
		t.Fatal("actor not round-tripped")
	}
}
