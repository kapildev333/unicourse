package content

import (
	"database/sql"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

type annJSON struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Emailed   bool   `json:"emailed"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	Author    struct{ UserID, DisplayName, Email string }
}

func (e *env) post(c any, title string, extra map[string]any) (int, []byte) {
	body := map[string]any{"title": title, "body": "Body of " + title}
	for k, v := range extra {
		body[k] = v
	}
	return e.call(e.as(c.(string)), "POST", "/sections/"+e.SectionA+"/announcements", body)
}

func TestAnnouncementCRUD(t *testing.T) {
	e := newEnv(t)
	st, raw := e.post("inst", "Hello", nil)
	var a annJSON
	if st != 201 || json.Unmarshal(raw, &a) != nil || !a.Emailed || a.Author.UserID != e.UserID["inst@a.edu"] || a.Author.DisplayName != "N inst@a.edu" || a.Author.Email != "inst@a.edu" || a.CreatedAt != a.UpdatedAt {
		t.Fatalf("create: %d %s", st, raw)
	}
	p := "/sections/" + e.SectionA + "/announcements/" + a.ID
	time.Sleep(10 * time.Millisecond)
	st, raw = e.call(e.TA, "PATCH", p, map[string]any{"title": "  New  "})
	var u annJSON
	if st != 200 || json.Unmarshal(raw, &u) != nil || u.Title != "New" || u.Body != a.Body || u.Author.UserID != a.Author.UserID || u.UpdatedAt <= a.UpdatedAt || u.CreatedAt != a.CreatedAt {
		t.Fatalf("patch: %d %s", st, raw)
	}
	if st, raw = e.call(e.TA, "DELETE", p, nil); st != 204 {
		t.Fatalf("delete: %d %s", st, raw)
	}
	st, raw = e.call(e.Stud, "GET", "/sections/"+e.SectionA+"/announcements", nil)
	if st != 200 || !strings.Contains(string(raw), `"items":[]`) {
		t.Fatalf("list: %d %s", st, raw)
	}
}

func TestAnnouncementOrderAndPages(t *testing.T) {
	e := newEnv(t)
	var ids []string
	for i := 0; i < 5; i++ {
		ids = append(ids, e.announcement(e.SectionA, string(rune('A'+i))))
		time.Sleep(5 * time.Millisecond)
	}
	list := func(q string) (items []annJSON, next string) {
		st, raw := e.call(e.Stud, "GET", "/sections/"+e.SectionA+"/announcements"+q, nil)
		var out struct {
			Items      []annJSON
			NextCursor string
		}
		if st != 200 || json.Unmarshal(raw, &out) != nil {
			t.Fatalf("list %s: %d %s", q, st, raw)
		}
		return out.Items, out.NextCursor
	}
	all, next := list("")
	if len(all) != 5 || next != "" {
		t.Fatalf("all: %d %q", len(all), next)
	}
	var got []string
	for i, a := range all {
		if a.ID != ids[4-i] {
			t.Fatalf("order at %d", i)
		}
	}
	q := "?limit=2"
	for pages := 0; ; pages++ {
		items, nx := list(q)
		for _, a := range items {
			got = append(got, a.ID)
		}
		if nx == "" {
			if pages != 2 || len(items) != 1 {
				t.Fatalf("pages = %d, last = %d", pages, len(items))
			}
			break
		}
		if len(items) != 2 {
			t.Fatalf("page of %d", len(items))
		}
		q = "?limit=2&cursor=" + nx
	}
	if !reflect.DeepEqual(got, []string{ids[4], ids[3], ids[2], ids[1], ids[0]}) {
		t.Fatalf("paged = %v", got)
	}
	if st, raw := e.call(e.Stud, "GET", "/sections/"+e.SectionA+"/announcements?cursor=bad", nil); st != 400 {
		t.Fatalf("bad cursor: %d %s", st, raw)
	} else if _, f := errDetails(raw); !reflect.DeepEqual(f, []string{"cursor"}) {
		t.Fatalf("bad cursor fields %v", f)
	}
}

func TestAnnouncementValidation(t *testing.T) {
	e := newEnv(t)
	path := "/sections/" + e.SectionA + "/announcements"
	fields := func(method, p string, body any) (int, []string) {
		st, raw := e.call(e.Inst, method, p, body)
		_, f := errDetails(raw)
		return st, f
	}
	for _, title := range []string{"", "   ", strings.Repeat("a", 201), "a\nb", "a\tb"} {
		if st, f := fields("POST", path, map[string]any{"title": title, "body": "b"}); st != 400 || !reflect.DeepEqual(f, []string{"title"}) {
			t.Errorf("title %q: %d %v", title, st, f)
		}
	}
	for _, body := range []string{"", strings.Repeat("a", 20001)} {
		if st, f := fields("POST", path, map[string]any{"title": "t", "body": body}); st != 400 || !reflect.DeepEqual(f, []string{"body"}) {
			t.Errorf("body len %d: %d %v", len(body), st, f)
		}
	}
	if st, f := fields("POST", path, map[string]any{"title": "", "body": ""}); st != 400 || !reflect.DeepEqual(f, []string{"title", "body"}) {
		t.Errorf("both: %d %v", st, f)
	}
	if st, f := fields("POST", path, map[string]any{"title": "t", "body": "b", "email": nil}); st != 400 || !reflect.DeepEqual(f, []string{"email"}) {
		t.Errorf("email null: %d %v", st, f)
	}
	if st, _ := fields("POST", path, map[string]any{"title": "t", "body": "b", "x": 1}); st != 400 {
		t.Errorf("unknown key: %d", st)
	}
	if st, _ := fields("POST", path, map[string]any{"title": strings.Repeat("é", 200), "body": "line 1\n\nline 3", "email": false}); st != 201 {
		t.Errorf("max title: %d", st)
	}
	id := e.announcement(e.SectionA, "T")
	p := path + "/" + id
	if st, f := fields("PATCH", p, map[string]any{}); st != 400 || !reflect.DeepEqual(f, []string{""}) {
		t.Errorf("{}: %d %v", st, f)
	}
	if st, raw := e.call(e.Inst, "PATCH", p, map[string]any{"title": nil}); st != 400 || !strings.Contains(string(raw), "must not be null") {
		t.Errorf("null: %d %s", st, raw)
	}
	if st, f := fields("PATCH", p, map[string]any{"title": "", "body": ""}); st != 400 || !reflect.DeepEqual(f, []string{"title", "body"}) {
		t.Errorf("patch both: %d %v", st, f)
	}
	other := e.announcement(e.SectionA2, "Other")
	for _, bad := range []string{other, "x", "00000000-0000-0000-0000-000000000000"} {
		for _, m := range []string{"PATCH", "DELETE"} {
			if st, _ := fields(m, path+"/"+bad, map[string]any{"title": "x"}); st != 404 {
				t.Errorf("%s %s: %d", m, bad, st)
			}
		}
	}
}

func TestAnnouncementMail(t *testing.T) {
	e := newEnv(t)
	e.user("stud3@a.edu", "member") // a member who is not on the section gets nothing
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		var inv string
		if err := tx.QueryRow(`insert into invitations (university_id, email, role, token_hash, expires_at) values ($1, 'invitee@a.edu', 'member', sha256('i'::bytea), now() + interval '1 day') returning id`, e.Fx.UniversityA).Scan(&inv); err != nil {
			return err
		}
		if _, err := tx.Exec(`insert into pending_enrollments (university_id, invitation_id, section_id) values ($1, $2, $3)`, e.Fx.UniversityA, inv, e.SectionA); err != nil {
			return err
		}
		// stud is both enrolled and staff: mailed once
		_, err := tx.Exec(`insert into section_staff (university_id, section_id, user_id, role) values ($1, $2, $3, 'ta')`, e.Fx.UniversityA, e.SectionA, e.UserID["stud@a.edu"])
		return err
	})
	st, raw := e.post("inst", "Midterm moved", map[string]any{"body": "Room 101"})
	if st != 201 {
		t.Fatalf("%d %s", st, raw)
	}
	all := e.mails("")
	var to []string
	for _, m := range all {
		to = append(to, m.To)
	}
	slices.Sort(to)
	if !reflect.DeepEqual(to, []string{"stud2@a.edu", "stud@a.edu", "ta@a.edu"}) {
		t.Fatalf("recipients %v", to)
	}
	m := e.mails("stud@a.edu")[0]
	want := "Room 101\n\nPosted by N inst@a.edu in CS101 Intro to Programming, " + e.TermName + "\nhttp://localhost/a/sections/" + e.SectionA + "/announcements\n"
	if m.Subject != "[CS101-001] Midterm moved" || m.Body != want || m.Urgent {
		t.Fatalf("mail %+v\nwant body %q", m, want)
	}
}

func TestAnnouncementMailSkipsRemovedMember(t *testing.T) {
	e := newEnv(t)
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`delete from memberships where user_id = $1`, e.UserID["stud@a.edu"])
		return err
	})
	if st, raw := e.post("inst", "Hi", nil); st != 201 {
		t.Fatalf("%d %s", st, raw)
	}
	if n := len(e.mails("stud@a.edu")); n != 0 {
		t.Fatalf("removed member got %d mails", n)
	}
	if n := len(e.mails("stud2@a.edu")); n != 1 {
		t.Fatalf("active student got %d mails", n)
	}
}

func TestAnnouncementMailNotSent(t *testing.T) {
	e := newEnv(t)
	st, raw := e.post("inst", "Quiet", map[string]any{"email": false})
	if st != 201 || jmap(t, raw)["emailed"] != false || len(e.mails("")) != 0 {
		t.Fatalf("email false: %d %s %d mails", st, raw, len(e.mails("")))
	}
	st, raw = e.post("inst", "Loud", nil)
	if st != 201 || jmap(t, raw)["emailed"] != true {
		t.Fatalf("%d %s", st, raw)
	}
	n := len(e.mails(""))
	id := jmap(t, raw)["id"].(string)
	p := "/sections/" + e.SectionA + "/announcements/" + id
	if st, _ := e.call(e.Inst, "PATCH", p, map[string]any{"body": "changed"}); st != 200 {
		t.Fatal(st)
	}
	if st, _ := e.call(e.Inst, "DELETE", p, nil); st != 204 || len(e.mails("")) != n {
		t.Fatalf("delete: %d, mails %d != %d", st, len(e.mails("")), n)
	}
	// an author without a display name is named by email
	e.DB.Exec(`update users set display_name = '' where email = 'inst@a.edu'`)
	e.post("inst", "Nameless", nil)
	ms := e.mails("stud@a.edu")
	if last := ms[len(ms)-1]; !strings.Contains(last.Body, "Posted by inst@a.edu in") {
		t.Fatalf("body %q", last.Body)
	}
	// only the author on the section: nobody to mail
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`delete from section_staff where section_id = $1 and role = 'ta'`, e.SectionA)
		if err == nil {
			_, err = tx.Exec(`delete from enrollments where section_id = $1`, e.SectionA)
		}
		return err
	})
	n = len(e.mails(""))
	st, raw = e.post("inst", "Alone", nil)
	if st != 201 || jmap(t, raw)["emailed"] != false || len(e.mails("")) != n {
		t.Fatalf("alone: %d %s", st, raw)
	}
}

func TestAnnouncementAudit(t *testing.T) {
	e := newEnv(t)
	_, raw := e.post("inst", "T1", nil)
	id := jmap(t, raw)["id"].(string)
	p := "/sections/" + e.SectionA + "/announcements/" + id
	e.call(e.Inst, "PATCH", p, map[string]any{"title": "T2"})
	e.call(e.Inst, "DELETE", p, nil)
	row := func(action string) (n int, before, after sql.NullString) {
		e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
			return tx.QueryRow(`select count(*), max(before::text), max(after::text) from audit_log where action = $1 and entity_type = 'announcement' and entity_id = $2 and actor_user_id = $3`,
				action, id, e.UserID["inst@a.edu"]).Scan(&n, &before, &after)
		})
		return
	}
	if n, b, a := row("announcement.create"); n != 1 || b.Valid || !strings.Contains(a.String, `"title": "T1"`) || !strings.Contains(a.String, `"emailed": true`) || strings.Contains(a.String, "@") {
		t.Errorf("create: %d %v %v", n, b, a)
	}
	if n, b, a := row("announcement.update"); n != 1 || !strings.Contains(b.String, `"title": "T1"`) || !strings.Contains(a.String, `"title": "T2"`) {
		t.Errorf("update: %d %v %v", n, b, a)
	}
	if n, b, a := row("announcement.delete"); n != 1 || !strings.Contains(b.String, `"title": "T2"`) || a.Valid {
		t.Errorf("delete: %d %v %v", n, b, a)
	}
}

func TestAnnouncementAuthz(t *testing.T) {
	e := newEnv(t)
	sec := "/sections/" + e.SectionA + "/announcements"
	fresh := func() string { return sec + "/" + e.announcement(e.SectionA, "T") }
	type row struct {
		name, method string
		path         func() string
		body         any
		want         [8]int
	}
	same := func(s string) func() string { return func() string { return s } }
	rows := []row{
		{"GET", "GET", same(sec), nil, [8]int{200, 200, 403, 200, 200, 200, 403, 403}},
		{"POST", "POST", same(sec), map[string]any{"title": "t", "body": "b"}, [8]int{201, 201, 403, 201, 201, 403, 403, 403}},
		{"PATCH", "PATCH", fresh, map[string]any{"title": "x"}, [8]int{200, 200, 403, 200, 200, 403, 403, 403}},
		{"DELETE", "DELETE", fresh, nil, [8]int{204, 204, 403, 204, 204, 403, 403, 403}},
	}
	for _, r := range rows {
		for i, role := range roles {
			if st, raw := e.call(e.as(role), r.method, r.path(), r.body); st != r.want[i] {
				t.Errorf("%s as %s: %d, want %d (%s)", r.name, role, st, r.want[i], raw)
			}
		}
	}
	// a student who is not enrolled in the second section cannot read it
	if st, _ := e.call(e.Stud, "GET", "/sections/"+e.SectionA2+"/announcements", nil); st != 403 {
		t.Errorf("stud on A2: %d", st)
	}
	if st, _ := e.call(e.Admin, "GET", "/sections/00000000-0000-0000-0000-000000000000/announcements", nil); st != 404 {
		t.Errorf("unknown section: %d", st)
	}
	if st, _ := e.call(e.Admin, "GET", "/sections/"+e.SectionB()+"/announcements", nil); st != 404 {
		t.Errorf("B's section: %d", st)
	}
}
