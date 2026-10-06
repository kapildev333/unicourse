package tenant

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"reflect"
	"sort"
	"testing"

	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
)

func regTenant(r *httpx.Router, d Deps) { Register(r, d) }

// adminCookie returns a session cookie for the seeded admin of university a.
func (e *env) adminCookie() *http.Cookie {
	e.T.Helper()
	tok, err := e.Deps.Sessions.Create(context.Background(), e.Fx.AdminA)
	if err != nil {
		e.T.Fatal(err)
	}
	return &http.Cookie{Name: "unicourse_session", Value: tok}
}

func keys(t *testing.T, raw []byte) []string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func (e *env) tx(uni string, fn func(tx *sql.Tx) error) {
	e.T.Helper()
	if err := pg.WithTx(context.Background(), e.DB, uni, fn); err != nil {
		e.T.Fatal(err)
	}
}

// auditRow returns the number of audit rows for action+entity and the before/after json of one.
func (e *env) auditRow(action, id string) (n int, before, after sql.NullString) {
	e.T.Helper()
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select count(*), max(before::text), max(after::text) from audit_log where action = $1 and entity_id = $2`, action, id).Scan(&n, &before, &after)
	})
	return
}

func idOf(t *testing.T, raw []byte) string {
	t.Helper()
	var m struct{ ID string }
	if err := json.Unmarshal(raw, &m); err != nil || m.ID == "" {
		t.Fatalf("no id in %s", raw)
	}
	return m.ID
}

const deptPath = "/api/v1/universities/a/departments"

func TestDepartments(t *testing.T) {
	e := newServer(t, regTenant)
	admin := e.adminCookie()
	_, deptAdmin := e.user(e.Fx.UniversityA, "da@a.edu", "dept_admin", "active", e.Fx.DeptA)
	_, member := e.user(e.Fx.UniversityA, "m@a.edu", "member", "active")

	st, raw := e.do(admin, "POST", deptPath, map[string]any{"name": "Math", "code": "MATH"})
	if st != 201 || !reflect.DeepEqual(keys(t, raw), []string{"code", "id", "name"}) {
		t.Fatalf("create: %d %s", st, raw)
	}
	id := idOf(t, raw)
	if n, _, after := e.auditRow("department.create", id); n != 1 || !after.Valid {
		t.Fatalf("create audit n=%d after=%v", n, after)
	}

	if st, raw = e.do(admin, "POST", deptPath, map[string]any{"name": "Other", "code": "MATH"}); st != 409 {
		t.Fatalf("dup: %d %s", st, raw)
	} else if c, _ := errBody(raw); c != "conflict" {
		t.Fatalf("dup code %s", c)
	}
	for who, c := range map[string]*http.Cookie{"dept_admin": deptAdmin, "member": member} {
		if st, _ = e.do(c, "POST", deptPath, map[string]any{"name": "X", "code": "XX"}); st != 403 {
			t.Fatalf("%s create: %d", who, st)
		}
		if st, _ = e.do(c, "PATCH", deptPath+"/"+id, map[string]any{"name": "X"}); st != 403 {
			t.Fatalf("%s patch: %d", who, st)
		}
		if st, _ = e.do(c, "DELETE", deptPath+"/"+id, nil); st != 403 {
			t.Fatalf("%s delete: %d", who, st)
		}
	}
	if st, _ = e.do(member, "GET", deptPath, nil); st != 200 {
		t.Fatalf("member list: %d", st)
	}

	// validation
	for name, b := range map[string]map[string]any{
		"lowercase code": {"name": "X", "code": "cs"},
		"long name":      {"name": repeat("n", 101), "code": "XX"},
		"empty name":     {"name": "  ", "code": "XX"},
		"missing code":   {"name": "X"},
	} {
		st, raw := e.do(admin, "POST", deptPath, b)
		c, d := errBody(raw)
		if st != 400 || c != "validation_failed" || len(d) != 1 {
			t.Fatalf("%s: %d %s", name, st, raw)
		}
	}
	_, raw = e.do(admin, "POST", deptPath, map[string]any{"name": "X", "code": "cs"})
	if _, d := errBody(raw); d[0].Field != "code" || d[0].Message != "must be 2 to 10 uppercase letters or digits" {
		t.Fatalf("code detail %+v", d)
	}

	// patch
	st, raw = e.do(admin, "PATCH", deptPath+"/"+id, map[string]any{"name": "Mathematics"})
	var got Department
	json.Unmarshal(raw, &got)
	if st != 200 || got.Name != "Mathematics" || got.Code != "MATH" {
		t.Fatalf("patch: %d %s", st, raw)
	}
	if n, b, a := e.auditRow("department.update", id); n != 1 || !b.Valid || !a.Valid {
		t.Fatal("update audit")
	}
	if st, _ = e.do(admin, "PATCH", deptPath+"/"+id, map[string]any{}); st != 400 {
		t.Fatalf("empty patch %d", st)
	}
	if st, _ = e.do(admin, "PATCH", deptPath+"/"+id, map[string]any{"code": "CS"}); st != 409 {
		t.Fatalf("patch to dup code %d", st)
	}

	// 404s
	unknown := "00000000-0000-4000-8000-000000000001"
	for _, p := range []string{unknown, "nope"} {
		for _, m := range []string{"PATCH", "DELETE"} {
			if st, _ = e.do(admin, m, deptPath+"/"+p, map[string]any{"name": "X"}); st != 404 {
				t.Fatalf("%s %s: %d", m, p, st)
			}
		}
	}
	var deptB string
	e.tx(e.Fx.UniversityB, func(tx *sql.Tx) error {
		return tx.QueryRow(`insert into departments (university_id, name, code) values ($1,'B','BB') returning id`, e.Fx.UniversityB).Scan(&deptB)
	})
	for _, m := range []string{"PATCH", "DELETE"} {
		if st, _ = e.do(admin, m, deptPath+"/"+deptB, map[string]any{"name": "X"}); st != 404 {
			t.Fatalf("cross-university %s: %d", m, st)
		}
	}

	// delete: in use, then trims invitations, audit before
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`insert into courses (university_id, department_id, code, title) values ($1,$2,'C1','T')`, e.Fx.UniversityA, e.Fx.DeptA)
		return err
	})
	if st, raw = e.do(admin, "DELETE", deptPath+"/"+e.Fx.DeptA, nil); st != 409 {
		t.Fatalf("delete in use: %d %s", st, raw)
	}
	var inv string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`insert into invitations (university_id, email, role, department_ids, token_hash, expires_at)
			values ($1,'i@a.edu','dept_admin',array[$2::uuid,$3::uuid],'\x01', now() + interval '1 day') returning id`, e.Fx.UniversityA, id, e.Fx.DeptA).Scan(&inv)
	})
	if st, _ = e.do(admin, "DELETE", deptPath+"/"+id, nil); st != 204 {
		t.Fatalf("delete: %d", st)
	}
	var ids string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select to_json(department_ids)::text from invitations where id = $1`, inv).Scan(&ids)
	})
	if ids != `["`+e.Fx.DeptA+`"]` {
		t.Fatalf("invitation ids %s", ids)
	}
	if n, b, _ := e.auditRow("department.delete", id); n != 1 || !b.Valid {
		t.Fatal("delete audit")
	}
	if st, _ = e.do(admin, "DELETE", deptPath+"/"+id, nil); st != 404 {
		t.Fatalf("second delete %d", st)
	}
}

func repeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}

func TestDepartmentsPagination(t *testing.T) {
	e := newServer(t, regTenant)
	admin := e.adminCookie()
	for _, c := range []string{"AA", "BB", "CC"} { // plus the seeded CS: 4 rows
		if st, raw := e.do(admin, "POST", deptPath, map[string]any{"name": c, "code": c}); st != 201 {
			t.Fatal(st, string(raw))
		}
	}
	type pg struct {
		Items      []Department
		NextCursor string
	}
	fetch := func(q string) (pg, []byte) {
		st, raw := e.do(admin, "GET", deptPath+q, nil)
		var p pg
		if st != 200 || json.Unmarshal(raw, &p) != nil {
			t.Fatalf("list %s: %d %s", q, st, raw)
		}
		return p, raw
	}
	p1, _ := fetch("?limit=2")
	if len(p1.Items) != 2 || p1.NextCursor == "" {
		t.Fatalf("page1 %+v", p1)
	}
	// delete the row the cursor points at: neither skip nor repeat
	if st, _ := e.do(admin, "DELETE", deptPath+"/"+p1.Items[1].ID, nil); st != 204 {
		t.Fatal(st)
	}
	p2, raw := fetch("?limit=2&cursor=" + p1.NextCursor)
	if len(p2.Items) != 2 || p2.Items[0].Code != "BB" && p2.Items[0].Code != "CC" || p2.NextCursor != "" {
		t.Fatalf("page2 %s", raw)
	}
	if p2.Items[0].ID == p1.Items[0].ID || p2.Items[1].ID == p1.Items[0].ID {
		t.Fatal("repeat")
	}
}

func TestDepartmentWriteRollsBackAudit(t *testing.T) {
	e := newServer(t, regTenant)
	admin := e.adminCookie()
	// a failing write (duplicate) leaves no audit row
	e.do(admin, "POST", deptPath, map[string]any{"name": "Dup", "code": "CS"})
	var n int
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select count(*) from audit_log`).Scan(&n)
	})
	if n != 0 {
		t.Fatalf("audit rows after failed write: %d", n)
	}
}
