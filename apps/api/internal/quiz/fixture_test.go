package quiz

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
	Admin, DaIn, DaOut, Inst, TA, Stud, Stud2, Other, BAdmin *http.Cookie
	UserID                                                   map[string]string // email -> id
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
	e.BAdmin = e.cookie(fx.AdminB)
	e.Admin = e.cookie(fx.AdminA)
	e.UserID["admin@a.edu"] = fx.AdminA
	e.DaIn = e.user("dain@a.edu", "dept_admin", fx.DeptA)
	e.DaOut = e.user("daout@a.edu", "dept_admin", e.DeptA2)
	e.Inst = e.user("inst@a.edu", "member")
	e.TA = e.user("ta@a.edu", "member")
	e.Stud = e.user("stud@a.edu", "member")
	e.Stud2 = e.user("stud2@a.edu", "member")
	e.Other = e.user("other@a.edu", "member")
	e.tx(fx.UniversityA, func(tx *sql.Tx) error {
		for _, s := range [][2]string{{"inst@a.edu", "instructor"}, {"ta@a.edu", "ta"}} {
			if _, err := tx.Exec(`insert into section_staff (university_id, section_id, user_id, role) values ($1,$2,$3,$4)`, fx.UniversityA, e.SectionA, e.UserID[s[0]], s[1]); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(`select seed_us_grade_scale($1)`, fx.UniversityA); err != nil {
			return err
		}
		_, err := tx.Exec(`insert into enrollments (university_id, section_id, user_id) select $1, $2, id from users where email in ('stud@a.edu', 'stud2@a.edu')`, fx.UniversityA, e.SectionA)
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

// quiz inserts a quiz of SectionA by SQL; opens and closes are SQL intervals from now() ("1 hour", "-2 hours"), D30.
func (e *env) quiz(opens, closes string) (id string) {
	e.T.Helper()
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`insert into assessments (university_id, section_id, kind, name, max_points, weight, position, opens_at, closes_at)
			values ($1, $2, 'quiz', 'Quiz ' || gen_random_uuid(), 1, 10, (select count(*) from assessments), now() + $3::interval, now() + $4::interval) returning id`,
			e.Fx.UniversityA, e.SectionA, opens, closes).Scan(&id)
	})
	return id
}

// putQuestions replaces the questions through the API as the admin and fails the test unless it is a 200.
func (e *env) putQuestions(id string, items any) {
	e.T.Helper()
	if st, b := e.raw(e.Admin, "PUT", e.qpath(id), map[string]any{"items": items}); st != 200 {
		e.T.Fatalf("put questions: %d %s", st, b)
	}
}

// closeQuiz moves the window into the past in one statement (the window check holds).
func (e *env) closeQuiz(id string) {
	e.T.Helper()
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`update assessments set opens_at = now() - interval '2 hours', closes_at = now() - interval '1 hour' where id = $1`, id)
		return err
	})
}

// attempt inserts an attempt row of the user by SQL; submitted makes it a submitted one (score 0).
func (e *env) attempt(assessmentID, email string, submitted bool) {
	e.T.Helper()
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`insert into attempts (university_id, assessment_id, user_id, submitted_at, score) values ($1, $2, $3, case when $4 then now() end, case when $4 then 0 end)`,
			e.Fx.UniversityA, assessmentID, e.UserID[email], submitted)
		return err
	})
}

// qpath is the questions URL of a quiz of SectionA.
func (e *env) qpath(id string) string {
	return "/sections/" + e.SectionA + "/assessments/" + id + "/questions"
}

// raw is call with the body as a string.
func (e *env) raw(c *http.Cookie, method, path string, body any) (int, string) {
	e.T.Helper()
	st, b := e.call(c, method, path, body)
	return st, string(b)
}

// as returns the session cookie of a role name.
func (e *env) as(role string) *http.Cookie {
	return map[string]*http.Cookie{"admin": e.Admin, "dain": e.DaIn, "daout": e.DaOut, "inst": e.Inst, "ta": e.TA, "stud": e.Stud,
		"stud2": e.Stud2, "other": e.Other, "badmin": e.BAdmin}[role]
}
