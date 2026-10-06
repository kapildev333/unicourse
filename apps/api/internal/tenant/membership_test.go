package tenant

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
)

const pathA = "/api/v1/universities/a"

func (e *env) memberID(uni, userID string) (id string) {
	e.T.Helper()
	e.tx(uni, func(tx *sql.Tx) error {
		return tx.QueryRow(`select id from memberships where user_id = $1`, userID).Scan(&id)
	})
	return
}

func (e *env) activeAdmins() (n int) {
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select count(*) from memberships where role = 'admin' and status = 'active'`).Scan(&n)
	})
	return
}

func (e *env) deptAdminRows(userID string) (n int) {
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select count(*) from department_admins where user_id = $1`, userID).Scan(&n)
	})
	return
}

func TestMembershipList(t *testing.T) {
	e := newServer(t, regTenant)
	uni := e.Fx.UniversityA
	da, daCookie := e.user(uni, "da@a.edu", "dept_admin", "active", e.Fx.DeptA)
	e.user(uni, "m@a.edu", "member", "active")
	adm := e.adminCookie()

	st, raw := e.do(adm, "GET", pathA+"/memberships?limit=2", nil)
	var l struct {
		Items      []Membership
		NextCursor string
	}
	if st != 200 || json.Unmarshal(raw, &l) != nil || len(l.Items) != 2 || l.NextCursor == "" {
		t.Fatalf("page1: %d %s", st, raw)
	}
	st, raw = e.do(adm, "GET", pathA+"/memberships?limit=2&cursor="+l.NextCursor, nil)
	var l2 struct{ Items []Membership }
	if st != 200 || json.Unmarshal(raw, &l2) != nil || len(l2.Items) != 1 || l2.Items[0].ID == l.Items[0].ID || l2.Items[0].ID == l.Items[1].ID {
		t.Fatalf("page2: %d %s", st, raw)
	}
	st, raw = e.do(adm, "GET", pathA+"/memberships", nil)
	var all struct{ Items []map[string]any }
	json.Unmarshal(raw, &all)
	found := false
	for _, m := range all.Items {
		for _, k := range []string{"id", "userId", "email", "displayName", "role", "status", "departmentIds"} {
			if _, ok := m[k]; !ok {
				t.Fatalf("missing %s in %v", k, m)
			}
		}
		if m["userId"] == da {
			found = true
			if ids := m["departmentIds"].([]any); len(ids) != 1 || ids[0] != e.Fx.DeptA {
				t.Fatalf("departmentIds: %v", m)
			}
		}
	}
	if st != 200 || len(all.Items) != 3 || !found {
		t.Fatalf("list: %d %s", st, raw)
	}
	if st, _ := e.do(daCookie, "GET", pathA+"/memberships", nil); st != 403 {
		t.Fatalf("dept_admin list: %d", st)
	}
}

func TestMembershipUpdate(t *testing.T) {
	e := newServer(t, regTenant)
	uni := e.Fx.UniversityA
	u, _ := e.user(uni, "u@a.edu", "member", "active")
	mid := e.memberID(uni, u)
	adm := e.adminCookie()
	path := pathA + "/memberships/" + mid

	st, raw := e.do(adm, "PATCH", path, map[string]any{"role": "dept_admin", "departmentIds": []string{e.Fx.DeptA, e.Fx.DeptA}})
	var m Membership
	if st != 200 || json.Unmarshal(raw, &m) != nil || m.Role != "dept_admin" || len(m.DepartmentIDs) != 1 || e.deptAdminRows(u) != 1 {
		t.Fatalf("promote: %d %s", st, raw)
	}
	if n, b, a := e.auditRow("membership.update", mid); n != 1 || !b.Valid || !a.Valid {
		t.Fatalf("audit %d %v %v", n, b, a)
	}
	// replacing the ids with a second department
	var d2 string
	e.tx(uni, func(tx *sql.Tx) error {
		return tx.QueryRow(`insert into departments (university_id, name, code) values ($1,'Math','MATH') returning id`, uni).Scan(&d2)
	})
	st, raw = e.do(adm, "PATCH", path, map[string]any{"role": "dept_admin", "departmentIds": []string{d2}})
	json.Unmarshal(raw, &m)
	if st != 200 || len(m.DepartmentIDs) != 1 || m.DepartmentIDs[0] != d2 || e.deptAdminRows(u) != 1 {
		t.Fatalf("replace: %d %s", st, raw)
	}
	// validation and not found
	for _, c := range []struct {
		body  map[string]any
		field string
	}{
		{map[string]any{"role": "dept_admin"}, "departmentIds"},
		{map[string]any{"role": "member", "departmentIds": []string{d2}}, "departmentIds"},
		{map[string]any{"role": "boss"}, "role"},
		{map[string]any{}, "role"},
		{map[string]any{"role": "dept_admin", "departmentIds": []string{"x"}}, "departmentIds"},
	} {
		st, raw := e.do(adm, "PATCH", path, c.body)
		if code, d := errBody(raw); st != 400 || code != "validation_failed" || d[0].Field != c.field {
			t.Fatalf("%v: %d %s", c.body, st, raw)
		}
	}
	if st, _ := e.do(adm, "PATCH", path, map[string]any{"role": "dept_admin", "departmentIds": []string{"00000000-0000-4000-8000-000000000001"}}); st != 404 {
		t.Fatalf("unknown dept: %d", st)
	}
	// demotion clears
	st, raw = e.do(adm, "PATCH", path, map[string]any{"role": "member"})
	json.Unmarshal(raw, &m)
	if st != 200 || m.Role != "member" || len(m.DepartmentIDs) != 0 || e.deptAdminRows(u) != 0 {
		t.Fatalf("demote: %d %s", st, raw)
	}
	// membership of university B is invisible
	bm := e.memberID(e.Fx.UniversityB, e.Fx.AdminB)
	if st, _ := e.do(adm, "PATCH", pathA+"/memberships/"+bm, map[string]any{"role": "member"}); st != 404 {
		t.Fatalf("B membership: %d", st)
	}
	if st, _ := e.do(adm, "DELETE", pathA+"/memberships/"+bm, nil); st != 404 {
		t.Fatalf("B membership delete: %d", st)
	}
	_, mc := e.user(uni, "m2@a.edu", "member", "active")
	if st, _ := e.do(mc, "PATCH", path, map[string]any{"role": "admin"}); st != 403 {
		t.Fatalf("member patch: %d", st)
	}
}

func TestMembershipLastAdmin(t *testing.T) {
	e := newServer(t, regTenant)
	uni := e.Fx.UniversityA
	adm := e.adminCookie()
	self := pathA + "/memberships/" + e.memberID(uni, e.Fx.AdminA)

	st, raw := e.do(adm, "PATCH", self, map[string]any{"role": "member"})
	if msg := string(raw); st != 409 || !strings.Contains(msg, "university must keep one admin") {
		t.Fatalf("demote only admin: %d %s", st, raw)
	}
	if st, _ := e.do(adm, "DELETE", self, nil); st != 409 {
		t.Fatalf("delete only admin: %d", st)
	}
	a2, _ := e.user(uni, "a2@a.edu", "admin", "active")
	if st, raw := e.do(adm, "PATCH", self, map[string]any{"role": "member"}); st != 200 {
		t.Fatalf("self-demote with another admin: %d %s", st, raw)
	}
	// a2 is now the only admin; the old session is a member
	if e.activeAdmins() != 1 {
		t.Fatal("admins")
	}
	_ = a2
}

func TestMembershipDeleteAndRace(t *testing.T) {
	e := newServer(t, regTenant)
	uni := e.Fx.UniversityA
	adm := e.adminCookie()
	u, _ := e.user(uni, "u@a.edu", "dept_admin", "active", e.Fx.DeptA)
	mid := e.memberID(uni, u)
	if st, _ := e.do(adm, "DELETE", pathA+"/memberships/"+mid, nil); st != 204 {
		t.Fatalf("delete: %d", st)
	}
	if n, b, _ := e.auditRow("membership.delete", mid); n != 1 || !b.Valid || e.deptAdminRows(u) != 0 {
		t.Fatalf("audit/department_admins: %d %v", n, b)
	}
	if st, _ := e.do(adm, "DELETE", pathA+"/memberships/"+mid, nil); st != 404 {
		t.Fatalf("again: %d", st)
	}

	// race: A1 and A2 demote each other
	a2, c2 := e.user(uni, "a2@a.edu", "admin", "active")
	m1, m2 := e.memberID(uni, e.Fx.AdminA), e.memberID(uni, a2)
	var wg sync.WaitGroup
	codes := make([]int, 2)
	for i, c := range []struct {
		ck *http.Cookie
		id string
	}{{adm, m2}, {c2, m1}} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes[i], _ = e.do(c.ck, "PATCH", pathA+"/memberships/"+c.id, map[string]any{"role": "member"})
		}()
	}
	wg.Wait()
	if e.activeAdmins() != 1 {
		t.Fatalf("admins after race: %d (codes %v)", e.activeAdmins(), codes)
	}
}
