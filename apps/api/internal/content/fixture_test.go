package content

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/kapildev333/unicourse/apps/api/internal/auth"
	"github.com/kapildev333/unicourse/apps/api/internal/blob"
	"github.com/kapildev333/unicourse/apps/api/internal/course"
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

	DeptA2, TermA2, CourseA, SectionA, SectionA2 string // S1 = CS101 / TermA / 001; S2 = CS101 / TermA2 / 001, no members
	TermName                                     string // name of TermA
	DeptB, TermB, CourseB                        string // university B
	// session cookies by role
	Admin, DaIn, DaOut, Inst, TA, Stud, Stud2, Other, BAdm *http.Cookie
	UserID                                                 map[string]string // email -> id
}

func newEnv(t *testing.T) *env { return newEnvLimit(t, 0) }

// newEnvLimit is newEnv with the upload limit set (0 is the default 100 MB).
func newEnvLimit(t *testing.T, max int64) *env {
	t.Helper()
	db := testdb.Open(t)
	fx := testdb.Seed(t, db)
	d := tenant.Deps{DB: db, Sessions: &auth.SessionStore{DB: db}, BaseURL: "http://localhost", Files: blob.Dir{Root: t.TempDir(), MaxBytes: max}}
	r := httpx.NewRouter()
	course.Register(r, d)
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
	e.tx(fx.UniversityA, func(tx *sql.Tx) error {
		if err := tx.QueryRow(`select name from terms where id = $1`, fx.TermA).Scan(&e.TermName); err != nil {
			return err
		}
		return tx.QueryRow(`insert into sections (university_id, course_id, term_id, number) values ($1,$2,$3,'001') returning id`, fx.UniversityA, e.CourseA, e.TermA2).Scan(&e.SectionA2)
	})
	e.Admin = e.cookie(fx.AdminA)
	e.UserID["admin@a.edu"] = fx.AdminA
	e.DaIn = e.user("dain@a.edu", "dept_admin", fx.DeptA)
	e.DaOut = e.user("daout@a.edu", "dept_admin", e.DeptA2)
	e.Inst = e.user("inst@a.edu", "member")
	e.TA = e.user("ta@a.edu", "member")
	e.Stud = e.user("stud@a.edu", "member")
	e.Stud2 = e.user("stud2@a.edu", "member")
	e.Other = e.user("other@a.edu", "member")
	e.BAdm = e.userIn(fx.UniversityB, "badm@b.edu", "admin")
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
	return e.userIn(e.Fx.UniversityA, email, role, deptIDs...)
}

// userIn is user for any university.
func (e *env) userIn(uni, email, role string, deptIDs ...string) *http.Cookie {
	e.T.Helper()
	var id string
	if err := pg.WithPlatformTx(context.Background(), e.DB, func(tx *sql.Tx) error {
		return tx.QueryRow(`insert into users (email, display_name) values ($1, $2) returning id`, email, "N "+email).Scan(&id)
	}); err != nil {
		e.T.Fatal(err)
	}
	e.tx(uni, func(tx *sql.Tx) error {
		if _, err := tx.Exec(`insert into memberships (university_id, user_id, role, status) values ($1,$2,$3,'active')`, uni, id, role); err != nil {
			return err
		}
		for _, d := range deptIDs {
			if _, err := tx.Exec(`insert into department_admins (university_id, department_id, user_id) values ($1,$2,$3)`, uni, d, id); err != nil {
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

// as returns the cookie of a column of the design 6 matrix: admin, dain, daout, inst, ta, stud, other, badm.
func (e *env) as(role string) *http.Cookie {
	return map[string]*http.Cookie{"admin": e.Admin, "dain": e.DaIn, "daout": e.DaOut, "inst": e.Inst, "ta": e.TA, "stud": e.Stud, "other": e.Other, "badm": e.BAdm}[role]
}

// roles is the column order of the design 6 matrix.
var roles = []string{"admin", "dain", "daout", "inst", "ta", "stud", "other", "badm"}

type mailRow struct {
	To, Subject, Body string
	Urgent            bool
}

// mails returns the queued mail to one recipient ("" = all), oldest first.
func (e *env) mails(to string) []mailRow {
	e.T.Helper()
	var out []mailRow
	if err := pg.WithPlatformTx(context.Background(), e.DB, func(tx *sql.Tx) error {
		rows, err := tx.Query(`select recipient, subject, body, urgent from outbox_mails where $1 = '' or recipient = $1 order by created_at, id`, to)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var m mailRow
			if err := rows.Scan(&m.To, &m.Subject, &m.Body, &m.Urgent); err != nil {
				return err
			}
			out = append(out, m)
		}
		return rows.Err()
	}); err != nil {
		e.T.Fatal(err)
	}
	return out
}

// announcementRow inserts an announcement of a section by SQL (author: Inst) and returns its id.
func (e *env) announcement(sectionID, title string) (id string) {
	e.T.Helper()
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`insert into announcements (university_id, section_id, author_user_id, title, body) values ($1, $2, $3, $4, 'Body') returning id`,
			e.Fx.UniversityA, sectionID, e.UserID["inst@a.edu"], title).Scan(&id)
	})
	return id
}

// SectionB creates a section in university B and returns its id.
func (e *env) SectionB() (id string) {
	e.T.Helper()
	e.tx(e.Fx.UniversityB, func(tx *sql.Tx) error {
		return tx.QueryRow(`insert into sections (university_id, course_id, term_id, number) values ($1,$2,$3,'001') returning id`, e.Fx.UniversityB, e.CourseB, e.TermB).Scan(&id)
	})
	return id
}

// week inserts a week of SectionA by SQL at the next position and returns its id.
func (e *env) week(title string) (id string) {
	e.T.Helper()
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`insert into weeks (university_id, section_id, title, position)
			values ($1, $2, $3, (select count(*) + 1 from weeks where section_id = $2)) returning id`, e.Fx.UniversityA, e.SectionA, title).Scan(&id)
	})
	return id
}

// material inserts a material row of a week by SQL (no blob unless the test puts one) at the next position.
func (e *env) material(weekID, title string) (id string) {
	e.T.Helper()
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`insert into materials (university_id, section_id, week_id, title, file_name, content_type, size_bytes, sha256, position, uploaded_by_user_id)
			values ($1, $2, $3, $4, 't.pdf', 'application/pdf', 1, repeat('a', 64), (select count(*) + 1 from materials where week_id = $3), $5) returning id`,
			e.Fx.UniversityA, e.SectionA, weekID, title, e.UserID["inst@a.edu"]).Scan(&id)
	})
	return id
}

// uploadParts posts a multipart body that fn writes through io.Pipe, so nothing is held in memory.
func (e *env) uploadParts(c *http.Cookie, path string, fn func(mw *multipart.Writer)) (int, []byte) {
	e.T.Helper()
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		fn(mw)
		pw.CloseWithError(mw.Close())
	}()
	req, _ := http.NewRequest("POST", e.Srv.URL+root+path, pr)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if c != nil {
		req.AddCookie(c)
	}
	res, err := http.DefaultClient.Do(req)
	pr.CloseWithError(io.ErrClosedPipe) // unblock the writer if the server answered early
	if err != nil {
		e.T.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	return res.StatusCode, raw
}

// upload posts one file (and a title part unless title is "") to a week of SectionA.
func (e *env) upload(c *http.Cookie, weekID, fileName string, content io.Reader, title string) (int, []byte) {
	e.T.Helper()
	return e.uploadParts(c, "/sections/"+e.SectionA+"/weeks/"+weekID+"/materials", func(mw *multipart.Writer) {
		if title != "" {
			mw.WriteField("title", title)
		}
		w, err := mw.CreateFormFile("file", fileName)
		if err == nil {
			_, _ = io.Copy(w, content)
		}
	})
}

// get requests a path with extra headers and returns the response and its body.
func (e *env) get(c *http.Cookie, path string, hdr map[string]string) (*http.Response, []byte) {
	e.T.Helper()
	req, _ := http.NewRequest("GET", e.Srv.URL+root+path, nil)
	if c != nil {
		req.AddCookie(c)
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.T.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	return res, raw
}

func (e *env) blobPath(id string) string { return filepath.Join(e.D.Files.Root, e.Fx.UniversityA, id) }

// tmpFiles lists the files in <root>/<university>/.tmp.
func (e *env) tmpFiles() []string {
	ents, _ := os.ReadDir(filepath.Join(e.D.Files.Root, e.Fx.UniversityA, ".tmp"))
	var out []string
	for _, f := range ents {
		out = append(out, f.Name())
	}
	return out
}

func (e *env) count(q string, args ...any) (n int) {
	e.T.Helper()
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error { return tx.QueryRow(q, args...).Scan(&n) })
	return
}
