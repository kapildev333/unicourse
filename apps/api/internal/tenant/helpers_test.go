package tenant

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/auth"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/testdb"
)

// env is a real router + log/recover middleware over httptest, as main wires it.
type env struct {
	T    *testing.T
	DB   *sql.DB
	Fx   testdb.Fixture
	Deps Deps
	Srv  *httptest.Server
	Log  *syncBuf // request log lines (JSON) and mail lines
}

type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// newServer opens the test database, seeds it, lets reg register routes, and serves them.
func newServer(t *testing.T, reg func(r *httpx.Router, d Deps)) *env {
	t.Helper()
	db := testdb.Open(t)
	fx := testdb.Seed(t, db)
	logs := &syncBuf{}
	l := slog.New(slog.NewJSONHandler(logs, nil))
	d := Deps{DB: db, Sessions: &auth.SessionStore{DB: db}, Mailer: auth.LogMailer{L: l}, BaseURL: "http://localhost"}
	r := httpx.NewRouter()
	reg(r, d)
	srv := httptest.NewServer(httpx.RequestIDMiddleware(httpx.LogMiddleware(l, httpx.Recover(r))))
	t.Cleanup(srv.Close)
	return &env{T: t, DB: db, Fx: fx, Deps: d, Srv: srv, Log: logs}
}

// user creates a user with a membership in uni (status active) and returns its id and a session cookie.
// deptIDs are inserted into department_admins.
func (e *env) user(uni, email, role, status string, deptIDs ...string) (userID string, cookie *http.Cookie) {
	e.T.Helper()
	ctx := context.Background()
	must := func(err error) {
		e.T.Helper()
		if err != nil {
			e.T.Fatal(err)
		}
	}
	err := pg.WithPlatformTx(ctx, e.DB, func(tx *sql.Tx) error {
		return tx.QueryRow(`select id from users where email = $1`, email).Scan(&userID)
	})
	if err != nil {
		must(pg.WithPlatformTx(ctx, e.DB, func(tx *sql.Tx) error {
			return tx.QueryRow(`insert into users (email) values ($1) returning id`, email).Scan(&userID)
		}))
	}
	must(pg.WithTx(ctx, e.DB, uni, func(tx *sql.Tx) error {
		if _, err := tx.Exec(`insert into memberships (university_id, user_id, role, status) values ($1, $2, $3, $4)`, uni, userID, role, status); err != nil {
			return err
		}
		for _, d := range deptIDs {
			if _, err := tx.Exec(`insert into department_admins (university_id, department_id, user_id) values ($1, $2, $3)`, uni, d, userID); err != nil {
				return err
			}
		}
		return nil
	}))
	tok, err := e.Deps.Sessions.Create(ctx, userID)
	must(err)
	return userID, &http.Cookie{Name: auth.SessionCookie, Value: tok}
}

// do sends a request (body is JSON-encoded unless nil) and returns status and raw body.
func (e *env) do(cookie *http.Cookie, method, path string, body any) (int, []byte) {
	e.T.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, e.Srv.URL+path, rd)
	if err != nil {
		e.T.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.T.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	return res.StatusCode, raw
}

// errBody decodes the error envelope.
func errBody(raw []byte) (code string, details []httpx.FieldError) {
	var e struct {
		Error struct {
			Code    string
			Details []httpx.FieldError
		}
	}
	json.Unmarshal(raw, &e)
	return e.Error.Code, e.Error.Details
}

func TestHelpers(t *testing.T) {
	if !IsUUID("00000000-0000-4000-8000-000000000001") || IsUUID("nope") || IsUUID("00000000-0000-4000-8000-00000000000G") {
		t.Fatal("IsUUID")
	}
	for _, in := range []string{`[]`, `null`} {
		if ids, err := ScanIDs([]byte(in)); err != nil || ids == nil || len(ids) != 0 {
			t.Fatalf("ScanIDs(%s) = %#v, %v", in, ids, err)
		}
	}
	if ids, _ := ScanIDs([]byte(`["a","b"]`)); len(ids) != 2 {
		t.Fatal("ScanIDs values")
	}
	if v, fe := TrimLen("name", "  hi ", 5); v != "hi" || fe != nil {
		t.Fatal("TrimLen ok")
	}
	if _, fe := TrimLen("name", "  ", 5); fe == nil || fe.Message != "is required" || fe.Field != "name" {
		t.Fatal("TrimLen empty")
	}
	if _, fe := TrimLen("name", "a\u0000b", 5); fe == nil || fe.Field != "name" || fe.Message != "must not contain NUL" {
		t.Fatalf("TrimLen NUL: %v", fe)
	}
	if _, fe := TrimLen("name", strings.Repeat("é", 6), 5); fe == nil || fe.Message != "must be at most 5 characters" {
		t.Fatalf("TrimLen long: %v", fe)
	}
	if e, ok := NoFields().(*httpx.Error); !ok || e.Code != "validation_failed" || len(e.Details) != 1 || e.Details[0].Field != "" {
		t.Fatal("NoFields")
	}
	r := httptest.NewRequest("GET", "/x?limit=2", nil)
	p, after, afterID, err := Page(r)
	if err != nil || p.Limit != 2 || after != nil || afterID != nil {
		t.Fatal("page no cursor")
	}
	cur := httpx.NextCursor(testTime, "00000000-0000-4000-8000-000000000001")
	p, after, afterID, err = Page(httptest.NewRequest("GET", "/x?cursor="+cur, nil))
	if err != nil || !testTime.Equal(after.(time.Time)) || afterID != "00000000-0000-4000-8000-000000000001" {
		t.Fatalf("page cursor: %v %v %v", after, afterID, err)
	}
	if _, _, _, err := Page(httptest.NewRequest("GET", "/x?limit=x", nil)); err == nil {
		t.Fatal("page bad limit")
	}
}

var testTime = time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
