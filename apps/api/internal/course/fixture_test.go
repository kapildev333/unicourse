package course

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kapildev333/unicourse/apps/api/internal/auth"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
	"github.com/kapildev333/unicourse/apps/api/internal/testdb"
)

const root = "/api/v1/universities/a"

// env is the real router + tenant.Wrap over httptest, seeded with the fixture of design §11.
type env struct {
	T   *testing.T
	DB  *sql.DB
	Fx  testdb.Fixture
	Srv *httptest.Server
	D   tenant.Deps

	DeptA2, TermA2, CourseA, SectionA string // S1 = CS101 / TermA / 001
	DeptB, TermB, CourseB             string // university B
	// session cookies by role
	Admin, DaIn, DaOut, Inst, TA, Stud, Other *http.Cookie
	UserID                                    map[string]string // email -> id
}

func newEnv(t *testing.T) *env {
	t.Helper()
	db := testdb.Open(t)
	fx := testdb.Seed(t, db)
	d := tenant.Deps{DB: db, Sessions: &auth.SessionStore{DB: db}, BaseURL: "http://localhost"}
	r := httpx.NewRouter()
	Register(r, d)
	srv := httptest.NewServer(httpx.RequestIDMiddleware(httpx.Recover(r)))
	t.Cleanup(srv.Close)
	e := &env{T: t, DB: db, Fx: fx, Srv: srv, D: d, UserID: map[string]string{}}
	e.tx(fx.UniversityA, func(tx *sql.Tx) error {
		if err := tx.QueryRow(`insert into departments (university_id, name, code) values ($1,'Math','MATH') returning id`, fx.UniversityA).Scan(&e.DeptA2); err != nil {
			return err
		}
		if err := tx.QueryRow(`insert into terms (university_id, name, starts_on, ends_on) values ($1,'Spring 2027','2027-01-15','2027-05-20') returning id`, fx.UniversityA).Scan(&e.TermA2); err != nil {
			return err
		}
		return tx.QueryRow(`insert into courses (university_id, department_id, code, title) values ($1,$2,'CS101','Intro to Programming') returning id`, fx.UniversityA, fx.DeptA).Scan(&e.CourseA)
	})
	e.tx(fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`insert into sections (university_id, course_id, term_id, number) values ($1,$2,$3,'001') returning id`, fx.UniversityA, e.CourseA, fx.TermA).Scan(&e.SectionA)
	})
	e.tx(fx.UniversityB, func(tx *sql.Tx) error {
		if err := tx.QueryRow(`insert into departments (university_id, name, code) values ($1,'Physics','PHY') returning id`, fx.UniversityB).Scan(&e.DeptB); err != nil {
			return err
		}
		if err := tx.QueryRow(`insert into terms (university_id, name, starts_on, ends_on) values ($1,'Fall 2026','2026-09-01','2026-12-15') returning id`, fx.UniversityB).Scan(&e.TermB); err != nil {
			return err
		}
		return tx.QueryRow(`insert into courses (university_id, department_id, code, title) values ($1,$2,'PHY1','Physics') returning id`, fx.UniversityB, e.DeptB).Scan(&e.CourseB)
	})
	e.Admin = e.cookie(fx.AdminA)
	e.UserID["admin@a.edu"] = fx.AdminA
	e.DaIn = e.user("dain@a.edu", "dept_admin", fx.DeptA)
	e.DaOut = e.user("daout@a.edu", "dept_admin", e.DeptA2)
	e.Inst = e.user("inst@a.edu", "member")
	e.TA = e.user("ta@a.edu", "member")
	e.Stud = e.user("stud@a.edu", "member")
	e.Other = e.user("other@a.edu", "member")
	e.tx(fx.UniversityA, func(tx *sql.Tx) error {
		for _, s := range [][2]string{{"inst@a.edu", "instructor"}, {"ta@a.edu", "ta"}} {
			if _, err := tx.Exec(`insert into section_staff (university_id, section_id, user_id, role) values ($1,$2,$3,$4)`, fx.UniversityA, e.SectionA, e.UserID[s[0]], s[1]); err != nil {
				return err
			}
		}
		_, err := tx.Exec(`insert into enrollments (university_id, section_id, user_id) values ($1,$2,$3)`, fx.UniversityA, e.SectionA, e.UserID["stud@a.edu"])
		return err
	})
	return e
}

func (e *env) cookie(userID string) *http.Cookie {
	e.T.Helper()
	tok, err := e.D.Sessions.Create(context.Background(), userID)
	if err != nil {
		e.T.Fatal(err)
	}
	return &http.Cookie{Name: auth.SessionCookie, Value: tok}
}

// user creates an active member of A with role (dept_admin rows for deptIDs) and returns a session.
func (e *env) user(email, role string, deptIDs ...string) *http.Cookie {
	e.T.Helper()
	var id string
	if err := pg.WithPlatformTx(context.Background(), e.DB, func(tx *sql.Tx) error {
		return tx.QueryRow(`insert into users (email, display_name) values ($1, $2) returning id`, email, "N "+email).Scan(&id)
	}); err != nil {
		e.T.Fatal(err)
	}
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		if _, err := tx.Exec(`insert into memberships (university_id, user_id, role, status) values ($1,$2,$3,'active')`, e.Fx.UniversityA, id, role); err != nil {
			return err
		}
		for _, d := range deptIDs {
			if _, err := tx.Exec(`insert into department_admins (university_id, department_id, user_id) values ($1,$2,$3)`, e.Fx.UniversityA, d, id); err != nil {
				return err
			}
		}
		return nil
	})
	e.UserID[email] = id
	return e.cookie(id)
}

func (e *env) tx(uni string, fn func(tx *sql.Tx) error) {
	e.T.Helper()
	if err := pg.WithTx(context.Background(), e.DB, uni, fn); err != nil {
		e.T.Fatal(err)
	}
}

func (e *env) call(c *http.Cookie, method, path string, body any) (int, []byte) {
	e.T.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, e.Srv.URL+root+path, rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c != nil {
		req.AddCookie(c)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.T.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	return res.StatusCode, raw
}

// audit returns the count of audit rows for action+entity and one row's before/after json.
func (e *env) audit(action, id string) (n int, before, after sql.NullString) {
	e.T.Helper()
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select count(*), max(before::text), max(after::text) from audit_log where action = $1 and entity_id = $2`, action, id).Scan(&n, &before, &after)
	})
	return
}

func jmap(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("bad json %s", raw)
	}
	return m
}

func errDetails(raw []byte) (code string, fields []string) {
	var b struct {
		Error struct {
			Code    string
			Details []httpx.FieldError
		}
	}
	json.Unmarshal(raw, &b)
	for _, d := range b.Error.Details {
		fields = append(fields, d.Field)
	}
	return b.Error.Code, fields
}
