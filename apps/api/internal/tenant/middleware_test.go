package tenant

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
)

type probeBody struct {
	Scope       authz.Scope `json:"scope"`
	CtxUni      string      `json:"ctxUni"`
	CtxScopeOK  bool        `json:"ctxScopeOk"`
	Memberships int         `json:"memberships"`
}

// probeEnv serves /probe (reports scope + visible rows), /fail (inserts a row, then errors) and
// /create (inserts a row, returns 201).
func probeEnv(t *testing.T) *env {
	return newServer(t, func(r *httpx.Router, d Deps) {
		r.Handle("GET", "/api/v1/universities/{slug}/probe", Wrap(d, func(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
			var n int
			if err := tx.QueryRow(`select count(*) from memberships where user_id = $1`, s.UserID).Scan(&n); err != nil {
				return 0, nil, err
			}
			cs, ok := authz.ScopeFromContext(r.Context())
			return 200, probeBody{Scope: s, CtxUni: UniversityID(r.Context()), CtxScopeOK: ok && cs.UserID == s.UserID, Memberships: n}, nil
		}))
		insert := func(r *http.Request, tx *sql.Tx, s authz.Scope) error {
			_, err := tx.Exec(`insert into departments (university_id, name, code) values ($1, 'Probe', 'PRB')`, s.UniversityID)
			return err
		}
		r.Handle("POST", "/api/v1/universities/{slug}/fail", Wrap(d, func(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
			if err := insert(r, tx, s); err != nil {
				return 0, nil, err
			}
			return 201, map[string]string{"leak": "must not be written"}, errors.New("boom")
		}))
		r.Handle("POST", "/api/v1/universities/{slug}/create", Wrap(d, func(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
			return 201, map[string]string{"ok": "yes"}, insert(r, tx, s)
		}))
		// swallows a failed statement (aborting the tx) and still claims success
		r.Handle("POST", "/api/v1/universities/{slug}/swallow", Wrap(d, func(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
			insert(r, tx, s)
			insert(r, tx, s) // duplicate code: tx aborted, error ignored
			return 201, map[string]string{"ok": "yes"}, nil
		}))
		r.Handle("DELETE", "/api/v1/universities/{slug}/nobody", Wrap(d, func(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
			return 204, nil, nil
		}))
	})
}

func (e *env) probe(c *http.Cookie, slug string) (int, probeBody, []byte) {
	e.T.Helper()
	st, raw := e.do(c, "GET", "/api/v1/universities/"+slug+"/probe", nil)
	var b probeBody
	if st == 200 {
		if err := json.Unmarshal(raw, &b); err != nil {
			e.T.Fatal(err, string(raw))
		}
	}
	return st, b, raw
}

func (e *env) deptCount(code string) (n int) {
	e.T.Helper()
	err := pg.WithTx(context.Background(), e.DB, e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select count(*) from departments where code = $1`, code).Scan(&n)
	})
	if err != nil {
		e.T.Fatal(err)
	}
	return n
}

func TestWrapAuthFailures(t *testing.T) {
	e := probeEnv(t)
	st, raw, _ := func() (int, []byte, int) {
		s, r := e.do(nil, "GET", "/api/v1/universities/a/probe", nil)
		return s, r, 0
	}()
	if code, _ := errBody(raw); st != 401 || code != "unauthenticated" {
		t.Fatalf("no session: %d %s", st, raw)
	}
	// anonymous callers cannot probe slugs: unknown slug without a session is still 401
	if st, _ := e.do(nil, "GET", "/api/v1/universities/zzz/probe", nil); st != 401 {
		t.Fatalf("anonymous unknown slug: %d", st)
	}
	_, c := e.user(e.Fx.UniversityA, "m@a.edu", "member", "active")
	if st, _, raw := e.probe(c, "zzz"); st != 404 {
		t.Fatalf("unknown slug: %d %s", st, raw)
	}
	if st, _, raw := e.probe(c, "b"); st != 403 {
		t.Fatalf("member of A on B: %d %s", st, raw)
	} else if code, _ := errBody(raw); code != "forbidden" {
		t.Fatalf("code %q", code)
	}
	_, inv := e.user(e.Fx.UniversityA, "inv@a.edu", "admin", "invited")
	if st, _, raw := e.probe(inv, "a"); st != 403 {
		t.Fatalf("invited membership: %d %s", st, raw)
	}
}

func TestWrapScopeIsolation(t *testing.T) {
	e := probeEnv(t)
	uid, c := e.user(e.Fx.UniversityA, "both@a.edu", "member", "active")
	// same user, admin of B
	if err := pg.WithTx(context.Background(), e.DB, e.Fx.UniversityB, func(tx *sql.Tx) error {
		_, err := tx.Exec(`insert into memberships (university_id, user_id, role, status) values ($1, $2, 'admin', 'active')`, e.Fx.UniversityB, uid)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	st, b, raw := e.probe(c, "a")
	if st != 200 || b.Scope.UniversityID != e.Fx.UniversityA || b.Scope.Role != authz.Member || b.Scope.UserID != uid ||
		b.CtxUni != e.Fx.UniversityA || !b.CtxScopeOK || b.Memberships != 1 {
		t.Fatalf("A: %d %s", st, raw)
	}
	st, b, raw = e.probe(c, "b")
	if st != 200 || b.Scope.UniversityID != e.Fx.UniversityB || b.Scope.Role != authz.Admin || b.Memberships != 1 {
		t.Fatalf("B: %d %s", st, raw)
	}
	if !strings.Contains(e.Log.String(), `"universityId":"`+e.Fx.UniversityA+`"`) {
		t.Fatalf("request log lacks universityId: %s", e.Log.String())
	}
	if !strings.Contains(e.Log.String(), `"userId":"`+uid+`"`) {
		t.Fatal("request log lacks userId")
	}
}

func TestWrapDeptIDs(t *testing.T) {
	e := probeEnv(t)
	_, c := e.user(e.Fx.UniversityA, "da@a.edu", "dept_admin", "active", e.Fx.DeptA)
	st, b, raw := e.probe(c, "a")
	if st != 200 || b.Scope.Role != authz.DeptAdmin || len(b.Scope.DeptIDs) != 1 || b.Scope.DeptIDs[0] != e.Fx.DeptA {
		t.Fatalf("%d %s", st, raw)
	}
	_, c = e.user(e.Fx.UniversityA, "mem@a.edu", "member", "active")
	if _, b, _ = e.probe(c, "a"); len(b.Scope.DeptIDs) != 0 {
		t.Fatalf("member DeptIDs %v", b.Scope.DeptIDs)
	}
	// a plain member with stale department_admins rows still gets no DeptIDs
	_, c = e.user(e.Fx.UniversityA, "stale@a.edu", "member", "active", e.Fx.DeptA)
	st, b, raw = e.probe(c, "a")
	if st != 200 || b.Scope.Role != authz.Member || len(b.Scope.DeptIDs) != 0 || strings.Contains(string(raw), e.Fx.DeptA) {
		t.Fatalf("stale member: %d %s", st, raw)
	}
}

func TestWrapCommitAndRollback(t *testing.T) {
	e := probeEnv(t)
	c := func() *http.Cookie { _, c := e.user(e.Fx.UniversityA, "adm2@a.edu", "admin", "active"); return c }()
	st, raw := e.do(c, "POST", "/api/v1/universities/a/fail", nil)
	if code, _ := errBody(raw); st != 500 || code == "" || strings.Contains(string(raw), "leak") {
		t.Fatalf("fail: %d %s", st, raw)
	}
	if n := e.deptCount("PRB"); n != 0 {
		t.Fatalf("handler error left %d rows", n)
	}
	st, raw = e.do(c, "POST", "/api/v1/universities/a/create", nil)
	if st != 201 || !strings.Contains(string(raw), `"ok":"yes"`) {
		t.Fatalf("create: %d %s", st, raw)
	}
	if n := e.deptCount("PRB"); n != 1 {
		t.Fatalf("committed rows %d", n)
	}
	// a failing commit-time statement (duplicate code) is an error response, never the success body
	st, raw = e.do(c, "POST", "/api/v1/universities/a/create", nil)
	if st == 201 || strings.Contains(string(raw), `"ok"`) {
		t.Fatalf("duplicate: %d %s", st, raw)
	}
	if st, raw = e.do(c, "DELETE", "/api/v1/universities/a/nobody", nil); st != 204 || len(raw) != 0 {
		t.Fatalf("204: %d %q", st, raw)
	}
}

func TestWrapResponseOnlyAfterCommit(t *testing.T) {
	e := probeEnv(t)
	_, c := e.user(e.Fx.UniversityA, "adm3@a.edu", "admin", "active")
	st, raw := e.do(c, "POST", "/api/v1/universities/a/swallow", nil)
	if st != 500 || strings.Contains(string(raw), `"ok"`) {
		t.Fatalf("swallow: %d %s", st, raw)
	}
	if n := e.deptCount("PRB"); n != 0 {
		t.Fatalf("aborted tx left %d rows", n)
	}
}

func TestWrapRaw(t *testing.T) {
	e := newServer(t, func(r *httpx.Router, d Deps) {
		r.Handle("GET", "/api/v1/universities/{slug}/raw", Wrap(d, func(*http.Request, *sql.Tx, authz.Scope) (int, any, error) {
			return 200, Raw{ContentType: "text/csv; charset=utf-8", Filename: "gradebook.csv", Body: []byte("a,b\n1,2\n")}, nil
		}))
	})
	_, c := e.user(e.Fx.UniversityA, "m@a.edu", "member", "active")
	req, _ := http.NewRequest("GET", e.Srv.URL+"/api/v1/universities/a/raw", nil)
	req.AddCookie(c)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 || string(b) != "a,b\n1,2\n" || res.Header.Get("Content-Type") != "text/csv; charset=utf-8" ||
		res.Header.Get("Content-Disposition") != `attachment; filename="gradebook.csv"` {
		t.Fatalf("%d %q %v", res.StatusCode, b, res.Header)
	}
}
