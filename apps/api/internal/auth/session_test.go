package auth

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/testdb"
)

func sqlExec(t *testing.T, db *sql.DB, q string, args ...any) {
	t.Helper()
	err := pg.WithPlatformTx(context.Background(), db, func(tx *sql.Tx) error {
		_, err := tx.Exec(q, args...)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func sqlCount(t *testing.T, db *sql.DB, q string, args ...any) (n int) {
	t.Helper()
	err := pg.WithPlatformTx(context.Background(), db, func(tx *sql.Tx) error {
		return tx.QueryRow(q, args...).Scan(&n)
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func isUnauth(err error) bool {
	var he *httpx.Error
	return errors.As(err, &he) && he.Code == "unauthenticated"
}

func TestSessionLifecycle(t *testing.T) {
	db := testdb.Open(t)
	fx := testdb.Seed(t, db)
	ctx := context.Background()
	s := &SessionStore{DB: db}

	tok, err := s.Create(ctx, fx.AdminA)
	if err != nil {
		t.Fatal(err)
	}
	a, slid, err := s.Lookup(ctx, tok)
	if err != nil || slid || a.UserID != fx.AdminA || a.Email != "admin@a.edu" {
		t.Fatalf("lookup: %+v slid=%v err=%v", a, slid, err)
	}
	if n := sqlCount(t, db, `select count(*) from sessions where token_hash = $1`, []byte(tok)); n != 0 {
		t.Fatal("plaintext token stored")
	}
	if n := sqlCount(t, db, `select count(*) from sessions where token_hash = $1`, hashToken(tok)); n != 1 {
		t.Fatal("hashed token not stored")
	}

	if err := s.Revoke(ctx, tok); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Lookup(ctx, tok); !isUnauth(err) {
		t.Fatalf("after revoke: %v", err)
	}
	if _, _, err := s.Lookup(ctx, "garbage"); !isUnauth(err) {
		t.Fatalf("garbage: %v", err)
	}
}

func TestSessionExpiryAndSlide(t *testing.T) {
	db := testdb.Open(t)
	fx := testdb.Seed(t, db)
	ctx := context.Background()
	s := &SessionStore{DB: db}

	// past 30 days: unauthenticated and the row is deleted
	tok, _ := s.Create(ctx, fx.AdminA)
	sqlExec(t, db, `update sessions set expires_at = now() - interval '1 second' where token_hash = $1`, hashToken(tok))
	if _, _, err := s.Lookup(ctx, tok); !isUnauth(err) {
		t.Fatalf("expired: %v", err)
	}
	if n := sqlCount(t, db, `select count(*) from sessions where token_hash = $1`, hashToken(tok)); n != 0 {
		t.Fatal("expired row not deleted")
	}

	// seen 2 days ago: slides
	tok, _ = s.Create(ctx, fx.AdminA)
	sqlExec(t, db, `update sessions set last_seen_at = now() - interval '2 days', expires_at = now() + interval '1 day' where token_hash = $1`, hashToken(tok))
	_, slid, err := s.Lookup(ctx, tok)
	if err != nil || !slid {
		t.Fatalf("want slid, got %v %v", slid, err)
	}
	if n := sqlCount(t, db, `select count(*) from sessions where token_hash = $1 and expires_at > now() + interval '29 days 23 hours'`, hashToken(tok)); n != 1 {
		t.Fatal("expires_at not extended to ~30d")
	}

	// seen 1 hour ago: no slide
	tok, _ = s.Create(ctx, fx.AdminA)
	sqlExec(t, db, `update sessions set last_seen_at = now() - interval '1 hour' where token_hash = $1`, hashToken(tok))
	if _, slid, err := s.Lookup(ctx, tok); err != nil || slid {
		t.Fatalf("want no slide, got %v %v", slid, err)
	}
}

func TestAuthenticateAndRequireSession(t *testing.T) {
	db := testdb.Open(t)
	fx := testdb.Seed(t, db)
	ctx := context.Background()
	s := &SessionStore{DB: db}
	tok, _ := s.Create(ctx, fx.AdminA)

	// no cookie
	r := httptest.NewRequest("GET", "/", nil)
	if _, err := Authenticate(httptest.NewRecorder(), r, s); !isUnauth(err) {
		t.Fatalf("no cookie: %v", err)
	}

	// valid cookie, slide re-sets the cookie
	sqlExec(t, db, `update sessions set last_seen_at = now() - interval '2 days' where token_hash = $1`, hashToken(tok))
	r = httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: SessionCookie, Value: tok})
	w := httptest.NewRecorder()
	a, err := Authenticate(w, r, s)
	if err != nil || a.UserID != fx.AdminA {
		t.Fatalf("auth: %+v %v", a, err)
	}
	c := w.Result().Cookies()
	if len(c) != 1 || c[0].Name != SessionCookie || c[0].MaxAge != 2592000 || !c[0].HttpOnly || !c[0].Secure || c[0].SameSite != http.SameSiteLaxMode || c[0].Path != "/" {
		t.Fatalf("slide cookie: %+v", c)
	}

	// middleware: 401 envelope without cookie, actor in ctx with
	var got Actor
	h := RequireSession(s)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = FromContext(r.Context())
	}))
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 401 {
		t.Fatalf("status %d", w.Code)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || got.Email != "admin@a.edu" {
		t.Fatalf("status %d actor %+v", w.Code, got)
	}

	// clear cookie
	w = httptest.NewRecorder()
	clearSessionCookie(w)
	if c := w.Result().Cookies(); len(c) != 1 || c[0].MaxAge >= 0 || c[0].Name != SessionCookie {
		t.Fatalf("clear: %+v", c)
	}
}
