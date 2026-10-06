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

func TestCourseTruthTable(t *testing.T) {
	e := newEnv(t)
	body := func() map[string]any { return map[string]any{"departmentId": e.Fx.DeptA, "code": "X1", "title": "T"} }
	n := 0
	for _, c := range []struct {
		who                 string
		list, read, writeIn int
	}{
		{"admin", 200, 0, 201}, {"dain", 200, 0, 201}, {"daout", 200, 0, 403},
		{"inst", 403, 0, 403}, {"ta", 403, 0, 403}, {"stud", 403, 0, 403}, {"other", 403, 0, 403},
	} {
		ck := map[string]*http.Cookie{"admin": e.Admin, "dain": e.DaIn, "daout": e.DaOut, "inst": e.Inst, "ta": e.TA, "stud": e.Stud, "other": e.Other}[c.who]
		if st, raw := e.call(ck, "GET", "/courses", nil); st != c.list {
			t.Errorf("%s list: %d %s", c.who, st, raw)
		}
		n++
		b := body()
		b["code"] = "X" + string(rune('A'+n))
		if st, raw := e.call(ck, "POST", "/courses", b); st != c.writeIn {
			t.Errorf("%s create: %d %s", c.who, st, raw)
		}
		wantW := map[int]int{201: 200, 403: 403}[c.writeIn]
		if st, raw := e.call(ck, "PATCH", "/courses/"+e.CourseA, map[string]any{"title": "New " + c.who}); st != wantW {
			t.Errorf("%s patch: %d %s", c.who, st, raw)
		}
	}
	// unauthenticated and malformed ids
	if st, _ := e.call(nil, "GET", "/courses", nil); st != 401 {
		t.Errorf("anon list %d", st)
	}
	for _, m := range []string{"PATCH", "DELETE"} {
		for _, id := range []string{"nope", "00000000-0000-0000-0000-000000000000", e.CourseB} {
			if st, raw := e.call(e.Admin, m, "/courses/"+id, map[string]any{"title": "x"}); st != 404 {
				t.Errorf("%s %s: %d %s", m, id, st, raw)
			}
		}
	}
	// 404 before 403 for a plain member on a malformed id
	if st, _ := e.call(e.Other, "DELETE", "/courses/nope", nil); st != 404 {
		t.Errorf("member malformed id: %d", st)
	}
}

func TestCourseCreate(t *testing.T) {
	e := newEnv(t)
	st, raw := e.call(e.DaIn, "POST", "/courses", map[string]any{"departmentId": e.Fx.DeptA, "code": "  CS201 ", "title": " Data Structures "})
	m := jmap(t, raw)
	if st != 201 || len(m) != 4 || m["code"] != "CS201" || m["title"] != "Data Structures" || m["departmentId"] != e.Fx.DeptA {
		t.Fatalf("create: %d %s", st, raw)
	}
	id, _ := m["id"].(string)
	if n, _, after := e.audit("course.create", id); n != 1 || !after.Valid || !strings.Contains(after.String, "CS201") {
		t.Fatalf("audit %d %v", n, after)
	}
	if st, _ := e.call(e.Admin, "POST", "/courses", map[string]any{"departmentId": e.Fx.DeptA, "code": "CS201", "title": "x"}); st != 409 {
		t.Fatalf("dup: %d", st)
	}
	// validation: all errors at once, body order; permission (403) is not reached before validation
	st, raw = e.call(e.Admin, "POST", "/courses", map[string]any{"departmentId": "zz", "code": strings.Repeat("c", 21), "title": strings.Repeat("t", 201)})
	if _, f := errDetails(raw); st != 400 || !reflect.DeepEqual(f, []string{"departmentId", "code", "title"}) {
		t.Fatalf("validation: %d %s", st, raw)
	}
	st, raw = e.call(e.Admin, "POST", "/courses", map[string]any{})
	if _, f := errDetails(raw); st != 400 || !reflect.DeepEqual(f, []string{"departmentId", "code", "title"}) {
		t.Fatalf("empty: %d %s", st, raw)
	}
	if st, _ := e.call(e.Admin, "POST", "/courses", map[string]any{"departmentId": e.Fx.DeptA, "code": strings.Repeat("c", 20), "title": strings.Repeat("é", 200)}); st != 201 {
		t.Fatalf("max lengths: %d", st)
	}
	// department of another university (composite FK) and unknown department -> 404
	for _, d := range []string{e.DeptB, "00000000-0000-0000-0000-000000000000"} {
		if st, raw := e.call(e.Admin, "POST", "/courses", map[string]any{"departmentId": d, "code": "Z", "title": "z"}); st != 404 {
			t.Fatalf("dept %s: %d %s", d, st, raw)
		}
	}
}

func TestCourseListFiltersAndPaging(t *testing.T) {
	e := newEnv(t)
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		// five courses sharing one created_at: the keyset must break ties by id
		_, err := tx.Exec(`insert into courses (university_id, department_id, code, title, created_at)
			select $1, $2, 'M' || g, 'Math ' || g, '2026-01-01T00:00:00Z' from generate_series(1,5) g`, e.Fx.UniversityA, e.DeptA2)
		return err
	})
	var seen []string
	cursor := ""
	for i := 0; i < 10; i++ {
		st, raw := e.call(e.DaOut, "GET", "/courses?limit=2"+cursor, nil)
		var l struct {
			Items      []map[string]any
			NextCursor string
		}
		if st != 200 || json.Unmarshal(raw, &l) != nil {
			t.Fatalf("page: %d %s", st, raw)
		}
		for _, it := range l.Items {
			seen = append(seen, it["id"].(string))
		}
		if l.NextCursor == "" {
			break
		}
		cursor = "&cursor=" + url.QueryEscape(l.NextCursor)
	}
	uniq := map[string]bool{}
	for _, s := range seen {
		uniq[s] = true
	}
	if len(seen) != 6 || len(uniq) != 6 {
		t.Fatalf("paging saw %d rows, %d distinct (want 6)", len(seen), len(uniq))
	}
	st, raw := e.call(e.Admin, "GET", "/courses?departmentId="+e.DeptA2, nil)
	var l struct{ Items []map[string]any }
	json.Unmarshal(raw, &l)
	if st != 200 || len(l.Items) != 5 || l.Items[0]["departmentId"] != e.DeptA2 {
		t.Fatalf("filter: %d %s", st, raw)
	}
	if st, raw := e.call(e.Admin, "GET", "/courses?departmentId=zz", nil); st != 400 {
		t.Fatalf("bad filter: %d %s", st, raw)
	}
	// courses of university B never show up
	st, raw = e.call(e.Admin, "GET", "/courses", nil)
	if strings.Contains(string(raw), "PHY1") {
		t.Fatalf("leak: %s", raw)
	}
}

func TestCourseUpdateDelete(t *testing.T) {
	e := newEnv(t)
	st, raw := e.call(e.DaIn, "PATCH", "/courses/"+e.CourseA, map[string]any{"title": "Renamed"})
	m := jmap(t, raw)
	if st != 200 || m["title"] != "Renamed" || m["code"] != "CS101" || m["departmentId"] != e.Fx.DeptA {
		t.Fatalf("partial patch: %d %s", st, raw)
	}
	if n, b, a := e.audit("course.update", e.CourseA); n != 1 || !strings.Contains(b.String, "Intro") || !strings.Contains(a.String, "Renamed") {
		t.Fatalf("audit %d %v %v", n, b, a)
	}
	if st, raw := e.call(e.DaIn, "PATCH", "/courses/"+e.CourseA, map[string]any{}); st != 400 {
		t.Fatalf("empty patch: %d %s", st, raw)
	}
	if st, _ := e.call(e.DaIn, "PATCH", "/courses/"+e.CourseA, map[string]any{"code": ""}); st != 400 {
		t.Fatalf("blank code: %d", st)
	}
	// dept_admin may not move a course into a department they do not manage
	if st, _ := e.call(e.DaIn, "PATCH", "/courses/"+e.CourseA, map[string]any{"departmentId": e.DeptA2}); st != 403 {
		t.Fatalf("move: %d", st)
	}
	if st, _ := e.call(e.Admin, "PATCH", "/courses/"+e.CourseA, map[string]any{"departmentId": e.DeptA2}); st != 200 {
		t.Fatalf("admin move: %d", st)
	}
	if st, raw := e.call(e.Admin, "PATCH", "/courses/"+e.CourseA, map[string]any{"departmentId": e.DeptB}); st != 404 {
		t.Fatalf("foreign dept: %d %s", st, raw)
	}
	// delete: 409 while it has sections, 204 + audit before once empty
	if st, raw := e.call(e.Admin, "DELETE", "/courses/"+e.CourseA, nil); st != 409 {
		t.Fatalf("delete with sections: %d %s", st, raw)
	}
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`delete from sections`)
		return err
	})
	if st, _ := e.call(e.DaIn, "DELETE", "/courses/"+e.CourseA, nil); st != 403 {
		t.Fatalf("daIn delete after move: %d", st)
	}
	if st, raw := e.call(e.DaOut, "DELETE", "/courses/"+e.CourseA, nil); st != 204 || len(raw) != 0 {
		t.Fatalf("delete: %d %s", st, raw)
	}
	if n, b, _ := e.audit("course.delete", e.CourseA); n != 1 || !strings.Contains(b.String, "CS101") {
		t.Fatalf("delete audit %d %v", n, b)
	}
	if st, _ := e.call(e.Admin, "DELETE", "/courses/"+e.CourseA, nil); st != 404 {
		t.Fatalf("second delete: %d", st)
	}
}
