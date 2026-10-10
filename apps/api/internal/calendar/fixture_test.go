package calendar

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kapildev333/unicourse/apps/api/internal/auth"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
	"github.com/kapildev333/unicourse/apps/api/internal/testdb"
)

// env is the real router over httptest, seeded with university A (America/New_York), a section S1 (CS101/001,
// term TermA) with an instructor, a TA and two enrolled students, a second section S2 (CS102) and a member
// who is in no section.
type env struct {
	T   *testing.T
	DB  *sql.DB
	Fx  testdb.Fixture
	Srv *httptest.Server
	D   tenant.Deps

	S1, S2                             string
	Admin, Inst, TA, Stud, Stud2, Lone *http.Cookie
	UserID                             map[string]string
	Logs                               *bytes.Buffer
}

func newEnv(t *testing.T) *env {
	t.Helper()
	db := testdb.Open(t)
	fx := testdb.Seed(t, db)
	d := tenant.Deps{DB: db, Sessions: &auth.SessionStore{DB: db}, BaseURL: "http://app.test"}
	r := httpx.NewRouter()
	Register(r, d)
	logs := &bytes.Buffer{}
	srv := httptest.NewServer(httpx.RequestIDMiddleware(httpx.LogMiddleware(slog.New(slog.NewJSONHandler(logs, nil)), httpx.Recover(r))))
	t.Cleanup(srv.Close)
	e := &env{T: t, DB: db, Fx: fx, Srv: srv, D: d, UserID: map[string]string{"admin@a.edu": fx.AdminA}, Logs: logs}
	e.Admin = e.cookie(fx.AdminA)
	e.Inst = e.user("inst@a.edu")
	e.TA = e.user("ta@a.edu")
	e.Stud = e.user("stud@a.edu")
	e.Stud2 = e.user("stud2@a.edu")
	e.Lone = e.user("lone@a.edu")
	e.tx(func(tx *sql.Tx) error {
		var course1, course2 string
		for i, c := range []*string{&course1, &course2} {
			code, title := "CS101", "Intro to CS"
			if i == 1 {
				code, title = "CS102", "Data Structures"
			}
			if err := tx.QueryRow(`insert into courses (university_id, department_id, code, title) values ($1,$2,$3,$4) returning id`, fx.UniversityA, fx.DeptA, code, title).Scan(c); err != nil {
				return err
			}
		}
		if err := tx.QueryRow(`insert into sections (university_id, course_id, term_id, number) values ($1,$2,$3,'001') returning id`, fx.UniversityA, course1, fx.TermA).Scan(&e.S1); err != nil {
			return err
		}
		if err := tx.QueryRow(`insert into sections (university_id, course_id, term_id, number) values ($1,$2,$3,'001') returning id`, fx.UniversityA, course2, fx.TermA).Scan(&e.S2); err != nil {
			return err
		}
		for _, s := range [][2]string{{"inst@a.edu", "instructor"}, {"ta@a.edu", "ta"}} {
			if _, err := tx.Exec(`insert into section_staff (university_id, section_id, user_id, role) values ($1,$2,$3,$4)`, fx.UniversityA, e.S1, e.UserID[s[0]], s[1]); err != nil {
				return err
			}
		}
		_, err := tx.Exec(`insert into enrollments (university_id, section_id, user_id) select $1, $2, id from users where email in ('stud@a.edu', 'stud2@a.edu')`, fx.UniversityA, e.S1)
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

func (e *env) user(email string) *http.Cookie {
	e.T.Helper()
	var id string
	if err := pg.WithPlatformTx(context.Background(), e.DB, func(tx *sql.Tx) error {
		return tx.QueryRow(`insert into users (email, display_name) values ($1, $1) returning id`, email).Scan(&id)
	}); err != nil {
		e.T.Fatal(err)
	}
	e.tx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`insert into memberships (university_id, user_id, role, status) values ($1,$2,'member','active')`, e.Fx.UniversityA, id)
		return err
	})
	e.UserID[email] = id
	return e.cookie(id)
}

func (e *env) tx(fn func(tx *sql.Tx) error) {
	e.T.Helper()
	if err := pg.WithTx(context.Background(), e.DB, e.Fx.UniversityA, fn); err != nil {
		e.T.Fatal(err)
	}
}

// work inserts a quiz or assignment of section whose window is given as SQL intervals from now() (D30),
// e.g. opens "-1 day", closes "2 days". extra is "time_limit_minutes" / "late_until" assignments for quizzes
// ("30" minutes) and assignments (late interval, or "").
func (e *env) work(section, kind, name, opens, closes, extra string) (id string) {
	e.T.Helper()
	var limit, late any
	if kind == "quiz" && extra != "" {
		limit = extra
	}
	if kind == "assignment" && extra != "" {
		late = extra
	}
	e.tx(func(tx *sql.Tx) error {
		return tx.QueryRow(`insert into assessments (university_id, section_id, kind, name, max_points, weight, opens_at, closes_at, time_limit_minutes, late_until)
			values ($1,$2,$3,$4,10,5, now() + $5::interval, now() + $6::interval, $7::int, now() + $8::interval) returning id`,
			e.Fx.UniversityA, section, kind, name, opens, closes, limit, late).Scan(&id)
	})
	return id
}

func (e *env) manual(section, name string) {
	e.T.Helper()
	e.tx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`insert into assessments (university_id, section_id, kind, name, max_points, weight) values ($1,$2,'manual',$3,10,5)`, e.Fx.UniversityA, section, name)
		return err
	})
}

// attempt starts (started ago, an interval) a quiz attempt for email; submitted also submits it.
func (e *env) attempt(email, quiz, started string, submitted bool) {
	e.T.Helper()
	e.tx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`insert into attempts (university_id, assessment_id, user_id, started_at, submitted_at, score)
			values ($1,$2,$3, now() - $4::interval, case when $5 then now() end, case when $5 then 0 end)`, e.Fx.UniversityA, quiz, e.UserID[email], started, submitted)
		return err
	})
}

func (e *env) submit(email, assignment string) {
	e.T.Helper()
	e.tx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`insert into submissions (university_id, assessment_id, user_id) values ($1,$2,$3)`, e.Fx.UniversityA, assignment, e.UserID[email])
		return err
	})
}

func (e *env) call(c *http.Cookie, method, path string, body any) (int, []byte) {
	e.T.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, e.Srv.URL+"/api/v1/universities/a"+path, rd)
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
