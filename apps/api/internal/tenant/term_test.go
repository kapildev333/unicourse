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
