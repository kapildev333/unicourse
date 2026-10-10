package notification

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/auth"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
	"github.com/kapildev333/unicourse/apps/api/internal/testdb"
)

type env struct {
	T   *testing.T
	DB  *sql.DB
	Fx  testdb.Fixture
	Srv *httptest.Server
	D   tenant.Deps

	Section, Announcement, Assessment string
	Stud, Other                       *http.Cookie
	UserID                            map[string]string
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
	e.Stud = e.user(fx.UniversityA, "stud@a.edu")
	e.Other = e.user(fx.UniversityA, "other@a.edu")
	e.tx(fx.UniversityA, func(tx *sql.Tx) error {
		var course string
		if err := tx.QueryRow(`insert into courses (university_id, department_id, code, title) values ($1,$2,'CS101','Intro to CS') returning id`, fx.UniversityA, fx.DeptA).Scan(&course); err != nil {
			return err
		}
		if err := tx.QueryRow(`insert into sections (university_id, course_id, term_id, number) values ($1,$2,$3,'001') returning id`, fx.UniversityA, course, fx.TermA).Scan(&e.Section); err != nil {
			return err
		}
		if err := tx.QueryRow(`insert into announcements (university_id, section_id, author_user_id, title, body) values ($1,$2,$3,'Midterm moved','B') returning id`, fx.UniversityA, e.Section, fx.AdminA).Scan(&e.Announcement); err != nil {
			return err
		}
		return tx.QueryRow(`insert into assessments (university_id, section_id, kind, name, max_points, weight, opens_at, closes_at)
			values ($1,$2,'quiz','Quiz 1',1,5, now(), now() + interval '1 day') returning id`, fx.UniversityA, e.Section).Scan(&e.Assessment)
	})
	return e
}

func (e *env) user(uni, email string) *http.Cookie {
	e.T.Helper()
	var id string
	if err := pg.WithPlatformTx(context.Background(), e.DB, func(tx *sql.Tx) error {
		return tx.QueryRow(`insert into users (email, display_name) values ($1, $1) on conflict (email) do update set email = excluded.email returning id`, email).Scan(&id)
	}); err != nil {
		e.T.Fatal(err)
	}
	e.tx(uni, func(tx *sql.Tx) error {
		_, err := tx.Exec(`insert into memberships (university_id, user_id, role, status) values ($1,$2,'member','active')`, uni, id)
		return err
	})
	e.UserID[email] = id
	tok, err := e.D.Sessions.Create(context.Background(), id)
	if err != nil {
		e.T.Fatal(err)
	}
	return &http.Cookie{Name: auth.SessionCookie, Value: tok}
}

func (e *env) tx(uni string, fn func(tx *sql.Tx) error) {
	e.T.Helper()
	if err := pg.WithTx(context.Background(), e.DB, uni, fn); err != nil {
		e.T.Fatal(err)
	}
}

// note inserts a notification of the user created `ago` ago and returns its id.
func (e *env) note(email, kind string, ago time.Duration) (id string) {
	e.T.Helper()
	var ann, ass any
	switch kind {
	case "announcement":
		ann = e.Announcement
	case "assessment_created", "scores_published":
		ass = e.Assessment
	}
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`insert into notifications (university_id, user_id, kind, section_id, announcement_id, assessment_id, created_at)
			values ($1,$2,$3,$4,$5,$6, now() - $7 * interval '1 second') returning id`,
			e.Fx.UniversityA, e.UserID[email], kind, e.Section, ann, ass, int(ago.Seconds())).Scan(&id)
	})
	return id
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
	req.AddCookie(c)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.T.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	return res.StatusCode, raw
}

type listOut struct {
	Items      []Notification
	NextCursor string
}

func (e *env) list(c *http.Cookie, q string) listOut {
	e.T.Helper()
	st, raw := e.call(c, "GET", "/me/notifications"+q, nil)
	var out listOut
	if st != 200 || json.Unmarshal(raw, &out) != nil {
		e.T.Fatalf("list: %d %s", st, raw)
	}
	return out
}

func (e *env) count(c *http.Cookie) int {
	e.T.Helper()
	st, raw := e.call(c, "GET", "/me/notifications/unread-count", nil)
	var out struct{ Count int }
	if st != 200 || json.Unmarshal(raw, &out) != nil {
		e.T.Fatalf("count: %d %s", st, raw)
	}
	return out.Count
}

func TestListShape(t *testing.T) {
	e := newEnv(t)
	e.note("stud@a.edu", "announcement", 30)
	e.note("stud@a.edu", "scores_published", 20)
	e.note("stud@a.edu", "grades_finalized", 10)
	st, raw := e.call(e.Stud, "GET", "/me/notifications", nil)
	if st != 200 || strings.Contains(string(raw), "null") {
		t.Fatalf("%d %s", st, raw)
	}
	items := e.list(e.Stud, "").Items
	if len(items) != 3 || items[0].Kind != "grades_finalized" || items[0].Subject != "" || items[0].Assessment != nil || items[0].AnnouncementID != "" ||
		items[0].Course.Code != "CS101" || items[0].Course.Title != "Intro to CS" || items[0].Section.Number != "001" || items[0].Section.ID != e.Section {
		t.Fatalf("grades_finalized: %+v", items[0])
	}
	if a := items[1]; a.Kind != "scores_published" || a.Subject != "Quiz 1" || a.Assessment == nil || a.Assessment.ID != e.Assessment || a.Assessment.Kind != "quiz" || a.ReadAt != nil {
		t.Fatalf("scores_published: %+v", a)
	}
	if a := items[2]; a.Kind != "announcement" || a.Subject != "Midterm moved" || a.AnnouncementID != e.Announcement || a.Assessment != nil {
		t.Fatalf("announcement: %+v", a)
	}
	// the title is read live
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`update announcements set title = 'Renamed'`)
		return err
	})
	if got := e.list(e.Stud, "").Items[2].Subject; got != "Renamed" {
		t.Fatalf("subject %q", got)
	}
}

func TestListNewestFirstAndPaged(t *testing.T) {
	e := newEnv(t)
	a := e.note("stud@a.edu", "announcement", 30)
	b := e.note("stud@a.edu", "announcement", 20)
	c := e.note("stud@a.edu", "announcement", 10)
	p1 := e.list(e.Stud, "?limit=2")
	if len(p1.Items) != 2 || p1.Items[0].ID != c || p1.Items[1].ID != b || p1.NextCursor == "" {
		t.Fatalf("page 1: %+v", p1)
	}
	p2 := e.list(e.Stud, "?limit=2&cursor="+p1.NextCursor)
	if len(p2.Items) != 1 || p2.Items[0].ID != a || p2.NextCursor != "" {
		t.Fatalf("page 2: %+v", p2)
	}
}

func TestListOwnOnly(t *testing.T) {
	e := newEnv(t)
	e.note("stud@a.edu", "announcement", 10)
	e.note("other@a.edu", "announcement", 10)
	// the same user is also a member of B and has a row there
	cb := e.user(e.Fx.UniversityB, "stud@a.edu") // second membership for the same email
	_ = cb
	if n := len(e.list(e.Stud, "").Items); n != 1 {
		t.Fatalf("stud sees %d", n)
	}
	if n := len(e.list(e.Other, "").Items); n != 1 {
		t.Fatalf("other sees %d", n)
	}
	if got := e.count(e.Stud); got != 1 {
		t.Fatalf("count %d", got)
	}
	// someone who is not a member of A is refused
	st, _ := e.call(&http.Cookie{Name: auth.SessionCookie, Value: "bogus"}, "GET", "/me/notifications", nil)
	if st != 401 {
		t.Fatalf("anonymous %d", st)
	}
}

func TestUnreadCount(t *testing.T) {
	e := newEnv(t)
	if e.count(e.Stud) != 0 {
		t.Fatal("not zero")
	}
	id := e.note("stud@a.edu", "announcement", 10)
	e.note("stud@a.edu", "grades_finalized", 10)
	e.note("other@a.edu", "grades_finalized", 10)
	if e.count(e.Stud) != 2 {
		t.Fatal("want 2")
	}
	e.call(e.Stud, "PATCH", "/me/notifications/"+id, map[string]any{"read": true})
	if e.count(e.Stud) != 1 || e.count(e.Other) != 1 {
		t.Fatal("want 1 and 1")
	}
}

func TestMarkRead(t *testing.T) {
	e := newEnv(t)
	id := e.note("stud@a.edu", "announcement", 10)
	foreign := e.note("other@a.edu", "announcement", 10)
	p := "/me/notifications/"
	st, raw := e.call(e.Stud, "PATCH", p+id, map[string]any{"read": true})
	var n Notification
	if st != 200 || json.Unmarshal(raw, &n) != nil || n.ReadAt == nil || n.ID != id {
		t.Fatalf("read: %d %s", st, raw)
	}
	first := *n.ReadAt
	time.Sleep(10 * time.Millisecond)
	st, raw = e.call(e.Stud, "PATCH", p+id, map[string]any{"read": true})
	var again Notification
	if st != 200 || json.Unmarshal(raw, &again) != nil || again.ReadAt == nil || !again.ReadAt.Equal(first) {
		t.Fatalf("repeat changed readAt: %d %s", st, raw)
	}
	st, raw = e.call(e.Stud, "PATCH", p+id, map[string]any{"read": false})
	var un Notification
	if st != 200 || json.Unmarshal(raw, &un) != nil || un.ReadAt != nil || strings.Contains(string(raw), "readAt") {
		t.Fatalf("unread: %d %s", st, raw)
	}
	if st, _ := e.call(e.Stud, "PATCH", p+foreign, map[string]any{"read": true}); st != 404 {
		t.Fatalf("another user's row: %d", st)
	}
	if st, _ := e.call(e.Stud, "PATCH", p+"nope", map[string]any{"read": true}); st != 404 {
		t.Fatalf("bad uuid: %d", st)
	}
	if st, _ := e.call(e.Stud, "PATCH", p+"00000000-0000-0000-0000-000000000000", map[string]any{"read": true}); st != 404 {
		t.Fatalf("missing: %d", st)
	}
	if st, _ := e.call(e.Stud, "PATCH", p+id, map[string]any{"read": nil}); st != 400 {
		t.Fatalf("null: %d", st)
	}
	if st, _ := e.call(e.Stud, "PATCH", p+id, map[string]any{}); st != 400 {
		t.Fatalf("empty: %d", st)
	}
	if st, _ := e.call(e.Stud, "PATCH", p+id, map[string]any{"bogus": 1}); st != 400 {
		t.Fatalf("unknown field: %d", st)
	}
	// the other user's row is untouched
	if e.count(e.Other) != 1 {
		t.Fatal("foreign row was changed")
	}
}

func TestMarkAllRead(t *testing.T) {
	e := newEnv(t)
	e.note("stud@a.edu", "announcement", 10)
	e.note("stud@a.edu", "grades_finalized", 10)
	e.note("other@a.edu", "grades_finalized", 10)
	if st, raw := e.call(e.Stud, "PATCH", "/me/notifications", map[string]any{"read": false}); st != 400 || !strings.Contains(string(raw), "must be true") {
		t.Fatalf("false: %d %s", st, raw)
	}
	if st, raw := e.call(e.Stud, "PATCH", "/me/notifications", map[string]any{"read": true}); st != 204 {
		t.Fatalf("%d %s", st, raw)
	}
	if e.count(e.Stud) != 0 || e.count(e.Other) != 1 {
		t.Fatal("mark all touched the wrong rows")
	}
	if st, _ := e.call(e.Stud, "PATCH", "/me/notifications", map[string]any{"read": true}); st != 204 {
		t.Fatalf("repeat: %d", st)
	}
}

func TestSettings(t *testing.T) {
	e := newEnv(t)
	st, raw := e.call(e.Stud, "GET", "/me/notification-settings", nil)
	if st != 200 || strings.TrimSpace(string(raw)) != `{"announcementMail":true}` {
		t.Fatalf("default: %d %s", st, raw)
	}
	st, raw = e.call(e.Stud, "PATCH", "/me/notification-settings", map[string]any{"announcementMail": false})
	if st != 200 || strings.TrimSpace(string(raw)) != `{"announcementMail":false}` {
		t.Fatalf("patch: %d %s", st, raw)
	}
	e.call(e.Stud, "PATCH", "/me/notification-settings", map[string]any{"announcementMail": false}) // unchanged: no second audit row
	var n int
	var before, after string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select count(*), coalesce(max(before::text), ''), coalesce(max(after::text), '') from audit_log where action = 'membership.update' and actor_user_id = $1`, e.UserID["stud@a.edu"]).Scan(&n, &before, &after)
	})
	if n != 1 || !strings.Contains(strings.ReplaceAll(before, " ", ""), `"announcementMail":true`) || !strings.Contains(strings.ReplaceAll(after, " ", ""), `"announcementMail":false`) {
		t.Fatalf("audit %d %s -> %s", n, before, after)
	}
	if st, raw = e.call(e.Other, "GET", "/me/notification-settings", nil); !strings.Contains(string(raw), "true") {
		t.Fatalf("other changed: %d %s", st, raw)
	}
	if st, _ = e.call(e.Stud, "PATCH", "/me/notification-settings", map[string]any{"announcementMail": nil}); st != 400 {
		t.Fatalf("null: %d", st)
	}
	if st, _ = e.call(e.Stud, "PATCH", "/me/notification-settings", map[string]any{}); st != 400 {
		t.Fatalf("empty: %d", st)
	}
	if st, _ = e.call(e.Stud, "PATCH", "/me/notification-settings", map[string]any{"x": 1}); st != 400 {
		t.Fatalf("unknown: %d", st)
	}
}
