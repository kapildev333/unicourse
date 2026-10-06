package course

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestSectionTruthTable(t *testing.T) {
	e := newEnv(t)
	n := 0
	for _, c := range []struct {
		who                    string
		ck                     *http.Cookie
		list, get, create, mut int
	}{
		{"admin", e.Admin, 200, 200, 201, 200},
		{"dain", e.DaIn, 200, 200, 201, 200},
		{"daout", e.DaOut, 200, 200, 403, 403}, // reads the whole university (design §14), writes only own departments
		{"inst", e.Inst, 403, 200, 403, 403},
		{"ta", e.TA, 403, 200, 403, 403},
		{"stud", e.Stud, 403, 200, 403, 403},
		{"other", e.Other, 403, 403, 403, 403},
	} {
		n++
		if st, raw := e.call(c.ck, "GET", "/sections", nil); st != c.list {
			t.Errorf("%s list: %d %s", c.who, st, raw)
		}
		if st, raw := e.call(c.ck, "GET", "/sections/"+e.SectionA, nil); st != c.get {
			t.Errorf("%s get: %d %s", c.who, st, raw)
		}
		if st, raw := e.call(c.ck, "POST", "/sections", map[string]any{"courseId": e.CourseA, "termId": e.TermA2, "number": "N" + string(rune('a'+n))}); st != c.create {
			t.Errorf("%s create: %d %s", c.who, st, raw)
		}
		if st, raw := e.call(c.ck, "PATCH", "/sections/"+e.SectionA, map[string]any{"number": "P" + string(rune('a'+n))}); st != c.mut {
			t.Errorf("%s patch: %d %s", c.who, st, raw)
		}
	}
	// the admin-created sections are in a section nobody but staff of S1 can read
	st, raw := e.call(e.Admin, "POST", "/sections", map[string]any{"courseId": e.CourseA, "termId": e.TermA2, "number": "ZZ"})
	other2 := jmap(t, raw)["id"].(string)
	if st != 201 {
		t.Fatal(st)
	}
	if st, _ := e.call(e.Inst, "GET", "/sections/"+other2, nil); st != 403 {
		t.Errorf("instructor reads other section: %d", st)
	}
	if st, _ := e.call(e.DaOut, "DELETE", "/sections/"+e.SectionA, nil); st != 403 {
		t.Errorf("daout delete: %d", st)
	}
	if st, _ := e.call(e.Inst, "DELETE", "/sections/"+e.SectionA, nil); st != 403 {
		t.Errorf("inst delete: %d", st)
	}
	// 404 first: malformed, unknown, foreign ids; and before 403 for a plain member
	var foreign string
	e.tx(e.Fx.UniversityB, func(tx *sql.Tx) error {
		if err := tx.QueryRow(`insert into terms (university_id, name, starts_on, ends_on) values ($1,'T2','2026-01-01','2026-02-01') returning id`, e.Fx.UniversityB).Scan(&foreign); err != nil {
			return err
		}
		return tx.QueryRow(`insert into sections (university_id, course_id, term_id, number) values ($1,$2,$3,'1') returning id`, e.Fx.UniversityB, e.CourseB, foreign).Scan(&foreign)
	})
	for _, id := range []string{"nope", "00000000-0000-0000-0000-000000000000", foreign} {
		for _, m := range []string{"GET", "PATCH", "DELETE"} {
			if st, raw := e.call(e.Other, m, "/sections/"+id, map[string]any{"number": "1"}); st != 404 {
				t.Errorf("%s %s: %d %s", m, id, st, raw)
			}
		}
	}
}

func TestSectionCreateAndShape(t *testing.T) {
	e := newEnv(t)
	st, raw := e.call(e.DaIn, "POST", "/sections", map[string]any{"courseId": e.CourseA, "termId": e.TermA2, "number": " 002 "})
	var got Section
	if st != 201 || json.Unmarshal(raw, &got) != nil {
		t.Fatalf("create: %d %s", st, raw)
	}
	m := jmap(t, raw)
	if len(m) != 8 || got.Number != "002" || got.Course.Code != "CS101" || got.Course.DepartmentID != e.Fx.DeptA ||
		got.Term.Name != "Spring 2027" || got.Term.StartsOn != "2027-01-15" || got.Term.EndsOn != "2027-05-20" {
		t.Fatalf("shape: %s", raw)
	}
	if !strings.Contains(string(raw), `"staff":[]`) || !strings.Contains(string(raw), `"pendingStaff":[]`) {
		t.Fatalf("arrays must be present and empty: %s", raw)
	}
	if n, _, after := e.audit("section.create", got.ID); n != 1 || !after.Valid || !strings.Contains(after.String, "002") {
		t.Fatalf("audit %d %v", n, after)
	}
	if st, _ := e.call(e.Admin, "POST", "/sections", map[string]any{"courseId": e.CourseA, "termId": e.TermA2, "number": "002"}); st != 409 {
		t.Fatalf("dup: %d", st)
	}
	// same number in another term is fine
	if st, _ := e.call(e.Admin, "POST", "/sections", map[string]any{"courseId": e.CourseA, "termId": e.Fx.TermA, "number": "002"}); st != 201 {
		t.Fatalf("same number other term: %d", st)
	}
	zero := "00000000-0000-0000-0000-000000000000"
	for _, b := range []map[string]any{
		{"courseId": e.CourseA, "termId": zero, "number": "1"},     // unknown term
		{"courseId": zero, "termId": e.TermA2, "number": "1"},      // unknown course
		{"courseId": e.CourseB, "termId": e.TermA2, "number": "1"}, // course of university B
		{"courseId": e.CourseA, "termId": e.TermB, "number": "1"},  // term of university B
	} {
		if st, raw := e.call(e.Admin, "POST", "/sections", b); st != 404 {
			t.Errorf("%v: %d %s", b, st, raw)
		}
	}
	st, raw = e.call(e.Admin, "POST", "/sections", map[string]any{"courseId": "x", "number": strings.Repeat("9", 21)})
	if _, f := errDetails(raw); st != 400 || !reflect.DeepEqual(f, []string{"courseId", "termId", "number"}) {
		t.Fatalf("validation: %d %s", st, raw)
	}
	if st, _ := e.call(e.Admin, "POST", "/sections", map[string]any{"courseId": e.CourseA, "termId": e.TermA2, "number": strings.Repeat("9", 20)}); st != 201 {
		t.Fatalf("max length: %d", st)
	}
}

func TestSectionListEmbedsStaff(t *testing.T) {
	e := newEnv(t)
	e.user("zinst@a.edu", "member")
	e.user("a-ta@a.edu", "member")
	var s2 string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		if err := tx.QueryRow(`insert into sections (university_id, course_id, term_id, number) values ($1,$2,$3,'002') returning id`, e.Fx.UniversityA, e.CourseA, e.TermA2).Scan(&s2); err != nil {
			return err
		}
		for _, u := range []struct{ email, role string }{{"zinst@a.edu", "instructor"}, {"a-ta@a.edu", "ta"}} {
			if _, err := tx.Exec(`insert into section_staff (university_id, section_id, user_id, role) values ($1,$2,$3,$4)`, e.Fx.UniversityA, e.SectionA, e.UserID[u.email], u.role); err != nil {
				return err
			}
		}
		// live, expired and accepted invitations, each with a pending staff row on S1
		for _, inv := range []struct{ email, role, expires, accepted string }{
			{"live@a.edu", "instructor", "now() + interval '1 day'", "null"},
			{"old@a.edu", "ta", "now() - interval '1 day'", "null"},
			{"done@a.edu", "ta", "now() + interval '1 day'", "now()"},
			{"alive-ta@a.edu", "ta", "now() + interval '1 day'", "null"},
		} {
			if _, err := tx.Exec(`with i as (insert into invitations (university_id, email, role, token_hash, expires_at, accepted_at)
				values ($1, $2, 'member', decode(md5($2), 'hex'), `+inv.expires+`, `+inv.accepted+`) returning id)
				insert into pending_section_staff (university_id, invitation_id, section_id, role) select $1, id, $3, $4 from i`,
				e.Fx.UniversityA, inv.email, e.SectionA, inv.role); err != nil {
				return err
			}
		}
		return nil
	})
	var l struct{ Items []Section }
	st, raw := e.call(e.DaOut, "GET", "/sections?termId="+e.Fx.TermA+"&courseId="+e.CourseA, nil)
	if st != 200 || json.Unmarshal(raw, &l) != nil || len(l.Items) != 1 {
		t.Fatalf("list: %d %s", st, raw)
	}
	var staff, pending []string
	for _, s := range l.Items[0].Staff {
		staff = append(staff, s.Role+":"+s.Email)
	}
	for _, p := range l.Items[0].PendingStaff {
		pending = append(pending, p.Role+":"+p.Email)
	}
	if want := []string{"instructor:inst@a.edu", "instructor:zinst@a.edu", "ta:a-ta@a.edu", "ta:ta@a.edu"}; !reflect.DeepEqual(staff, want) {
		t.Errorf("staff order %v", staff)
	}
	if want := []string{"instructor:live@a.edu", "ta:alive-ta@a.edu"}; !reflect.DeepEqual(pending, want) {
		t.Errorf("pending (expired and accepted must be hidden) %v", pending)
	}
	if l.Items[0].Staff[0].DisplayName != "N inst@a.edu" || l.Items[0].Staff[0].UserID != e.UserID["inst@a.edu"] {
		t.Errorf("staff fields %+v", l.Items[0].Staff[0])
	}
	// get returns the same embedded rows
	st, raw = e.call(e.Stud, "GET", "/sections/"+e.SectionA, nil)
	var one Section
	if json.Unmarshal(raw, &one); st != 200 || len(one.Staff) != 4 || len(one.PendingStaff) != 2 {
		t.Fatalf("get: %d %s", st, raw)
	}
	// filters
	for q, want := range map[string]int{"termId=" + e.TermA2: 1, "courseId=" + e.CourseA: 2, "termId=" + e.TermA2 + "&courseId=" + e.CourseB: 0, "": 2} {
		st, raw := e.call(e.Admin, "GET", "/sections?"+q, nil)
		var l struct{ Items []Section }
		if json.Unmarshal(raw, &l); st != 200 || len(l.Items) != want {
			t.Errorf("filter %q: %d %s", q, st, raw)
		}
	}
	if st, raw := e.call(e.Admin, "GET", "/sections?termId=x&courseId=y", nil); st != 400 {
		t.Errorf("bad filters: %d %s", st, raw)
	} else if _, f := errDetails(raw); !reflect.DeepEqual(f, []string{"termId", "courseId"}) {
		t.Errorf("bad filter fields %v", f)
	}
	// no N+1: one statement serves a list of any size (staff/pending are correlated subqueries
	// inside the single select, see sectionCols); here we check keyset paging over rows sharing created_at.
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`insert into sections (university_id, course_id, term_id, number, created_at)
			select $1, $2, $3, 'P' || g, '2026-01-01T00:00:00Z' from generate_series(1,5) g`, e.Fx.UniversityA, e.CourseA, e.TermA2)
		return err
	})
	var ids []string
	cursor := ""
	for i := 0; i < 10; i++ {
		var p struct {
			Items      []Section
			NextCursor string
		}
		st, raw := e.call(e.Admin, "GET", "/sections?limit=2"+cursor, nil)
		if st != 200 || json.Unmarshal(raw, &p) != nil {
			t.Fatalf("page: %d %s", st, raw)
		}
		for _, s := range p.Items {
			ids = append(ids, s.ID)
		}
		if p.NextCursor == "" {
			break
		}
		cursor = "&cursor=" + url.QueryEscape(p.NextCursor)
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			t.Errorf("section %s returned twice", id)
		}
		seen[id] = true
	}
	if len(ids) != 7 {
		t.Fatalf("paging returned %d sections, want 7", len(ids))
	}
}

func TestSectionUpdateDelete(t *testing.T) {
	e := newEnv(t)
	st, raw := e.call(e.DaIn, "PATCH", "/sections/"+e.SectionA, map[string]any{"number": " 010 "})
	var got Section
	if json.Unmarshal(raw, &got); st != 200 || got.Number != "010" || got.Course.Code != "CS101" || len(got.Staff) != 2 {
		t.Fatalf("patch: %d %s", st, raw)
	}
	if n, b, a := e.audit("section.update", e.SectionA); n != 1 || !strings.Contains(b.String, `"001"`) || !strings.Contains(a.String, `"010"`) {
		t.Fatalf("audit %d %v %v", n, b, a)
	}
	for _, b := range []map[string]any{{}, {"number": ""}, {"number": strings.Repeat("1", 21)}} {
		if st, _ := e.call(e.Admin, "PATCH", "/sections/"+e.SectionA, b); st != 400 {
			t.Errorf("patch %v: %d", b, st)
		}
	}
	// courseId/termId are not updatable
	if st, _ := e.call(e.Admin, "PATCH", "/sections/"+e.SectionA, map[string]any{"termId": e.TermA2}); st != 400 {
		t.Errorf("termId in patch: %d", st)
	}
	var s2 string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`insert into sections (university_id, course_id, term_id, number) values ($1,$2,$3,'020') returning id`, e.Fx.UniversityA, e.CourseA, e.Fx.TermA).Scan(&s2)
	})
	if st, _ := e.call(e.Admin, "PATCH", "/sections/"+s2, map[string]any{"number": "010"}); st != 409 {
		t.Errorf("number conflict: %d", st)
	}
	// delete cascades staff and enrollments (never 409) and audits before
	if st, raw := e.call(e.DaIn, "DELETE", "/sections/"+e.SectionA, nil); st != 204 || len(raw) != 0 {
		t.Fatalf("delete: %d %s", st, raw)
	}
	var staff, enr int
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		tx.QueryRow(`select count(*) from section_staff where section_id=$1`, e.SectionA).Scan(&staff)
		return tx.QueryRow(`select count(*) from enrollments where section_id=$1`, e.SectionA).Scan(&enr)
	})
	if staff != 0 || enr != 0 {
		t.Errorf("cascade left staff=%d enrollments=%d", staff, enr)
	}
	if n, b, _ := e.audit("section.delete", e.SectionA); n != 1 || !strings.Contains(b.String, `"010"`) {
		t.Errorf("delete audit %d %v", n, b)
	}
	if st, _ := e.call(e.Admin, "GET", "/sections/"+e.SectionA, nil); st != 404 {
		t.Errorf("get deleted: %d", st)
	}
}

func TestSectionNULAndPatchNull(t *testing.T) {
	e := newEnv(t)
	for name, body := range map[string]map[string]any{"NUL": {"number": "\u0000"}, "null": {"number": nil}} {
		st, raw := e.call(e.Admin, "PATCH", "/sections/"+e.SectionA, body)
		if _, f := errDetails(raw); st != 400 || !reflect.DeepEqual(f, []string{"number"}) {
			t.Errorf("%s: %d %s", name, st, raw)
		}
	}
	st, raw := e.call(e.Admin, "POST", "/sections", map[string]any{"courseId": e.CourseA, "termId": e.TermA2, "number": "1\u0000"})
	if _, f := errDetails(raw); st != 400 || !reflect.DeepEqual(f, []string{"number"}) {
		t.Errorf("create NUL: %d %s", st, raw)
	}
}
