package tenant

import (
	"database/sql"
	"encoding/json"
	"reflect"
	"testing"
)

const termPath = "/api/v1/universities/a/terms"

func TestTerms(t *testing.T) {
	e := newServer(t, regTenant)
	admin := e.adminCookie()
	_, member := e.user(e.Fx.UniversityA, "m@a.edu", "member", "active")

	st, raw := e.do(admin, "POST", termPath, map[string]any{"name": "Spring 2027", "startsOn": "2027-01-10", "endsOn": "2027-05-20"})
	if st != 201 || !reflect.DeepEqual(keys(t, raw), []string{"endsOn", "id", "name", "startsOn"}) {
		t.Fatalf("create: %d %s", st, raw)
	}
	id := idOf(t, raw)
	if n, _, a := e.auditRow("term.create", id); n != 1 || !a.Valid {
		t.Fatal("create audit")
	}

	// member: list yes, write no
	st, raw = e.do(member, "GET", termPath, nil)
	var l struct{ Items []Term }
	json.Unmarshal(raw, &l)
	if st != 200 || len(l.Items) != 2 || l.Items[1].StartsOn != "2027-01-10" {
		t.Fatalf("member list: %d %s", st, raw)
	}
	if st, _ = e.do(member, "POST", termPath, map[string]any{"name": "X", "startsOn": "2027-01-10", "endsOn": "2027-05-20"}); st != 403 {
		t.Fatalf("member create %d", st)
	}
	if st, _ = e.do(member, "DELETE", termPath+"/"+id, nil); st != 403 {
		t.Fatalf("member delete %d", st)
	}

	// validation
	_, raw = e.do(admin, "POST", termPath, map[string]any{"name": "X", "startsOn": "2027-01-10", "endsOn": "2027-01-10"})
	if _, d := errBody(raw); len(d) != 1 || d[0].Field != "endsOn" || d[0].Message != "must be after startsOn" {
		t.Fatalf("equal dates: %s", raw)
	}
	_, raw = e.do(admin, "POST", termPath, map[string]any{"name": "X", "startsOn": "2027-02-01", "endsOn": "2027-01-10"})
	if _, d := errBody(raw); len(d) != 1 || d[0].Field != "endsOn" {
		t.Fatalf("reversed: %s", raw)
	}
	for _, bad := range []string{"01/10/2027", "2027-13-01", "2027-1-1"} {
		st, raw = e.do(admin, "POST", termPath, map[string]any{"name": "X", "startsOn": bad, "endsOn": "2028-01-01"})
		if _, d := errBody(raw); st != 400 || len(d) != 1 || d[0].Field != "startsOn" || d[0].Message != "must be a date YYYY-MM-DD" {
			t.Fatalf("bad date %s: %d %s", bad, st, raw)
		}
	}
	if st, _ = e.do(admin, "POST", termPath, map[string]any{"name": repeat("n", 101), "startsOn": "2027-01-10", "endsOn": "2027-05-20"}); st != 400 {
		t.Fatalf("long name %d", st)
	}

	// duplicate name
	st, raw = e.do(admin, "POST", termPath, map[string]any{"name": "Spring 2027", "startsOn": "2028-01-10", "endsOn": "2028-05-20"})
	if c, _ := errBody(raw); st != 409 || c != "conflict" {
		t.Fatalf("dup: %d %s", st, raw)
	}
	var env struct{ Error struct{ Message string } }
	json.Unmarshal(raw, &env)
	if env.Error.Message != "term name already exists" {
		t.Fatal(env.Error.Message)
	}

	// patch: only endsOn is checked against the stored startsOn
	if st, _ = e.do(admin, "PATCH", termPath+"/"+id, map[string]any{"endsOn": "2027-01-10"}); st != 400 {
		t.Fatalf("patch endsOn <= stored startsOn: %d", st)
	}
	st, raw = e.do(admin, "PATCH", termPath+"/"+id, map[string]any{"endsOn": "2027-06-01"})
	var got Term
	json.Unmarshal(raw, &got)
	if st != 200 || got.EndsOn != "2027-06-01" || got.StartsOn != "2027-01-10" || got.Name != "Spring 2027" {
		t.Fatalf("patch: %d %s", st, raw)
	}
	if n, b, a := e.auditRow("term.update", id); n != 1 || !b.Valid || !a.Valid {
		t.Fatal("update audit")
	}
	if st, _ = e.do(admin, "PATCH", termPath+"/"+id, map[string]any{}); st != 400 {
		t.Fatalf("empty patch %d", st)
	}
	if st, _ = e.do(admin, "PATCH", termPath+"/"+id, map[string]any{"name": "Fall 2026"}); st != 409 {
		t.Fatalf("patch to dup name %d", st)
	}

	// 404s
	for _, p := range []string{"00000000-0000-4000-8000-000000000001", "nope"} {
		if st, _ = e.do(admin, "PATCH", termPath+"/"+p, map[string]any{"name": "X"}); st != 404 {
			t.Fatalf("patch %s: %d", p, st)
		}
		if st, _ = e.do(admin, "DELETE", termPath+"/"+p, nil); st != 404 {
			t.Fatalf("delete %s: %d", p, st)
		}
	}

	// delete: in use -> 409; free -> 204 + audit before
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		var course string
		if err := tx.QueryRow(`insert into courses (university_id, department_id, code, title) values ($1,$2,'C1','T') returning id`, e.Fx.UniversityA, e.Fx.DeptA).Scan(&course); err != nil {
			return err
		}
		_, err := tx.Exec(`insert into sections (university_id, course_id, term_id, number) values ($1,$2,$3,'001')`, e.Fx.UniversityA, course, e.Fx.TermA)
		return err
	})
	if st, _ = e.do(admin, "DELETE", termPath+"/"+e.Fx.TermA, nil); st != 409 {
		t.Fatalf("delete in use %d", st)
	}
	if st, _ = e.do(admin, "DELETE", termPath+"/"+id, nil); st != 204 {
		t.Fatalf("delete %d", st)
	}
	if n, b, _ := e.auditRow("term.delete", id); n != 1 || !b.Valid {
		t.Fatal("delete audit")
	}
}

func TestTermsPagination(t *testing.T) {
	e := newServer(t, regTenant)
	admin := e.adminCookie()
	for _, n := range []string{"T1", "T2"} { // plus seeded Fall 2026: 3 rows
		if st, raw := e.do(admin, "POST", termPath, map[string]any{"name": n, "startsOn": "2027-01-10", "endsOn": "2027-05-20"}); st != 201 {
			t.Fatal(st, string(raw))
		}
	}
	var p struct {
		Items      []Term
		NextCursor string
	}
	_, raw := e.do(admin, "GET", termPath+"?limit=2", nil)
	json.Unmarshal(raw, &p)
	if len(p.Items) != 2 || p.NextCursor == "" {
		t.Fatalf("page1 %s", raw)
	}
	_, raw = e.do(admin, "GET", termPath+"?limit=2&cursor="+p.NextCursor, nil)
	var p2 struct {
		Items      []Term
		NextCursor string
	}
	json.Unmarshal(raw, &p2)
	if len(p2.Items) != 1 || p2.Items[0].Name != "T2" || p2.NextCursor != "" || json.Valid(raw) && !reflect.DeepEqual(keys(t, raw), []string{"items"}) {
		t.Fatalf("page2 %s", raw)
	}
}

func TestTermsSameCreatedAtPaging(t *testing.T) {
	e := newServer(t, regTenant)
	admin := e.adminCookie()
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`insert into terms (university_id, name, starts_on, ends_on) select $1, 'T'||g, '2027-01-01', '2027-06-01' from generate_series(1,4) g`, e.Fx.UniversityA)
		return err
	})
	seen := map[string]int{}
	cursor := ""
	for i := 0; i < 10; i++ {
		_, raw := e.do(admin, "GET", termPath+"?limit=1&cursor="+cursor, nil)
		var p struct {
			Items      []Term
			NextCursor string
		}
		json.Unmarshal(raw, &p)
		for _, d := range p.Items {
			seen[d.ID]++
		}
		if cursor = p.NextCursor; cursor == "" {
			break
		}
	}
	if len(seen) != 5 {
		t.Fatalf("saw %d distinct rows: %v", len(seen), seen)
	}
	for id, n := range seen {
		if n != 1 {
			t.Fatalf("%s seen %d times", id, n)
		}
	}
}

func TestTermDateRange(t *testing.T) {
	e := newServer(t, regTenant)
	admin := e.adminCookie()
	for _, b := range []string{
		`{"name":"Z1","startsOn":"2026-01-01","endsOn":"0001-01-01"}`,
		`{"name":"Z2","startsOn":"0001-01-01","endsOn":"0001-01-01"}`,
		`{"name":"Z3","startsOn":"0000-01-01","endsOn":"2026-01-01"}`,
	} {
		if st, raw := e.do(admin, "POST", termPath, json.RawMessage(b)); st != 400 {
			t.Fatalf("%s: %d %s", b, st, raw)
		}
	}
	st, raw := e.do(admin, "PATCH", termPath+"/"+e.Fx.TermA, json.RawMessage(`{"endsOn":"0001-01-01"}`))
	if _, d := errBody(raw); st != 400 || len(d) == 0 || d[0].Field != "endsOn" {
		t.Fatalf("patch: %d %s", st, raw)
	}
}

func TestTermPatchNull(t *testing.T) {
	e := newServer(t, regTenant)
	admin := e.adminCookie()
	for _, f := range []string{"name", "startsOn", "endsOn"} {
		st, raw := e.do(admin, "PATCH", termPath+"/"+e.Fx.TermA, json.RawMessage(`{"`+f+`":null}`))
		if _, d := errBody(raw); st != 400 || len(d) != 1 || d[0].Field != f || d[0].Message != "must be a string" {
			t.Fatalf("%s null: %d %s", f, st, raw)
		}
	}
}

func TestListTermsQ(t *testing.T) {
	e := newServer(t, regTenant)
	admin := e.adminCookie()
	for _, n := range []string{"Spring 2027", "Spring 2028", "Summer 2027"} {
		if st, raw := e.do(admin, "POST", termPath, map[string]any{"name": n, "startsOn": "2027-01-10", "endsOn": "2027-05-20"}); st != 201 {
			t.Fatal(st, string(raw))
		}
	}
	list := func(q string) (int, []string, string) {
		st, raw := e.do(admin, "GET", termPath+q, nil)
		var p struct {
			Items      []Term
			NextCursor string
		}
		json.Unmarshal(raw, &p)
		var out []string
		for _, t := range p.Items {
			out = append(out, t.Name)
		}
		return st, out, p.NextCursor
	}
	for q, want := range map[string][]string{
		"?q=spr": {"Spring 2027", "Spring 2028"}, "?q=2027": {"Spring 2027", "Summer 2027"},
		"?q=2027-01": nil, "?q=%25": nil, "?q=_": nil,
	} {
		if _, got, _ := list(q); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: %v, want %v", q, got, want)
		}
	}
	if _, got, _ := list("?q="); len(got) != 4 {
		t.Errorf("blank: %v", got)
	}
	if st, _, _ := list("?q=" + repeat("a", 101)); st != 400 {
		t.Errorf("long q: %d", st)
	}
	_, p1, cur := list("?q=spr&limit=1")
	_, p2, cur2 := list("?q=spr&limit=1&cursor=" + cur)
	if !reflect.DeepEqual(p1, []string{"Spring 2027"}) || !reflect.DeepEqual(p2, []string{"Spring 2028"}) || cur2 != "" {
		t.Fatalf("paging %v %v %q", p1, p2, cur2)
	}
}
