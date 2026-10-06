package authz_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/testdb"
)

const randomID = "00000000-0000-4000-8000-000000000001"

func TestPure(t *testing.T) {
	s := authz.Scope{Role: authz.DeptAdmin, DeptIDs: []string{"d1"}}
	if authz.IsAdmin(s) || !authz.CanManageDepartment(s, "d1") || authz.CanManageDepartment(s, "d2") ||
		!authz.CanInvite(s, authz.Member) || authz.CanInvite(s, authz.DeptAdmin) || authz.CanInvite(authz.Scope{Role: authz.Member}, authz.Member) {
		t.Fatal("pure predicates")
	}
	if _, ok := authz.ScopeFromContext(context.Background()); ok {
		t.Fatal("scope in empty ctx")
	}
	if got, ok := authz.ScopeFromContext(authz.WithScope(context.Background(), s)); !ok || got.Role != authz.DeptAdmin {
		t.Fatal("scope round trip")
	}
}

func TestTruthTable(t *testing.T) {
	db := testdb.Open(t)
	fx := testdb.Seed(t, db)
	ctx := context.Background()
	uni := fx.UniversityA
	var d2, section, sectionB string
	users := map[string]string{}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(pg.WithPlatformTx(ctx, db, func(tx *sql.Tx) error {
		for _, n := range []string{"din", "dout", "inst", "ta", "student", "other", "stale"} {
			var id string
			if err := tx.QueryRow(`insert into users (email) values ($1) returning id`, n+"@a.edu").Scan(&id); err != nil {
				return err
			}
			users[n] = id
		}
		return nil
	}))
	must(pg.WithTx(ctx, db, uni, func(tx *sql.Tx) error {
		if err := tx.QueryRow(`insert into departments (university_id, name, code) values ($1, 'Math', 'MA') returning id`, uni).Scan(&d2); err != nil {
			return err
		}
		var course string
		if err := tx.QueryRow(`insert into courses (university_id, department_id, code, title) values ($1, $2, 'CS101', 'Intro') returning id`, uni, fx.DeptA).Scan(&course); err != nil {
			return err
		}
		if err := tx.QueryRow(`insert into sections (university_id, course_id, term_id, number) values ($1, $2, $3, '1') returning id`, uni, course, fx.TermA).Scan(&section); err != nil {
			return err
		}
		for _, r := range [][2]string{{"inst", "instructor"}, {"ta", "ta"}} {
			if _, err := tx.Exec(`insert into section_staff (university_id, section_id, user_id, role) values ($1, $2, $3, $4)`, uni, section, users[r[0]], r[1]); err != nil {
				return err
			}
		}
		// stale department_admins row for a plain member
		if _, err := tx.Exec(`insert into department_admins (university_id, department_id, user_id) values ($1, $2, $3)`, uni, fx.DeptA, users["stale"]); err != nil {
			return err
		}
		_, err := tx.Exec(`insert into enrollments (university_id, section_id, user_id) values ($1, $2, $3)`, uni, section, users["student"])
		return err
	}))
	// a section of university B, invisible under A's tx
	must(pg.WithTx(ctx, db, fx.UniversityB, func(tx *sql.Tx) error {
		var dept, course, term string
		b := fx.UniversityB
		if err := tx.QueryRow(`insert into departments (university_id, name, code) values ($1, 'X', 'X') returning id`, b).Scan(&dept); err != nil {
			return err
		}
		if err := tx.QueryRow(`insert into courses (university_id, department_id, code, title) values ($1, $2, 'X1', 'X') returning id`, b, dept).Scan(&course); err != nil {
			return err
		}
		if err := tx.QueryRow(`insert into terms (university_id, name, starts_on, ends_on) values ($1, 'T', '2026-01-01', '2026-02-01') returning id`, b).Scan(&term); err != nil {
			return err
		}
		if err := tx.QueryRow(`insert into sections (university_id, course_id, term_id, number) values ($1, $2, $3, '1') returning id`, b, course, term).Scan(&sectionB); err != nil {
			return err
		}
		// the same users are staff/student there, so the false results under A depend on RLS alone
		if _, err := tx.Exec(`insert into section_staff (university_id, section_id, user_id, role) values ($1, $2, $3, 'instructor')`, b, sectionB, users["inst"]); err != nil {
			return err
		}
		_, err := tx.Exec(`insert into enrollments (university_id, section_id, user_id) values ($1, $2, $3)`, b, sectionB, users["student"])
		return err
	}))

	actors := []struct {
		name string
		s    authz.Scope
	}{
		{"admin", authz.Scope{UniversityID: uni, UserID: fx.AdminA, Role: authz.Admin}},
		{"dept_admin-in", authz.Scope{UniversityID: uni, UserID: users["din"], Role: authz.DeptAdmin, DeptIDs: []string{fx.DeptA}}},
		{"dept_admin-out", authz.Scope{UniversityID: uni, UserID: users["dout"], Role: authz.DeptAdmin, DeptIDs: []string{d2}}},
		{"instructor", authz.Scope{UniversityID: uni, UserID: users["inst"], Role: authz.Member}},
		{"ta", authz.Scope{UniversityID: uni, UserID: users["ta"], Role: authz.Member}},
		{"student", authz.Scope{UniversityID: uni, UserID: users["student"], Role: authz.Member}},
		{"unrelated", authz.Scope{UniversityID: uni, UserID: users["other"], Role: authz.Member}},
		// a member whose scope carries department ids (stale rows): every manage row must be false
		{"member-stale-depts", authz.Scope{UniversityID: uni, UserID: users["stale"], Role: authz.Member, DeptIDs: []string{fx.DeptA}}},
	}
	type sectionFn func(context.Context, *sql.Tx, authz.Scope, string) (bool, error)
	// columns follow actors
	rows := []struct {
		name string
		pure func(authz.Scope) bool
		tx   sectionFn
		want [8]bool
	}{
		{name: "IsAdmin", pure: authz.IsAdmin, want: [8]bool{true}},
		{name: "CanManageDepartment", pure: func(s authz.Scope) bool { return authz.CanManageDepartment(s, fx.DeptA) }, want: [8]bool{true, true}},
		{name: "CanInvite(member)", pure: func(s authz.Scope) bool { return authz.CanInvite(s, authz.Member) }, want: [8]bool{true, true, true}},
		{name: "CanInvite(dept_admin)", pure: func(s authz.Scope) bool { return authz.CanInvite(s, authz.DeptAdmin) }, want: [8]bool{true}},
		{name: "CanInvite(admin)", pure: func(s authz.Scope) bool { return authz.CanInvite(s, authz.Admin) }, want: [8]bool{true}},
		{name: "CanManageSection", tx: authz.CanManageSection, want: [8]bool{true, true}},
		{name: "CanManageTAs", tx: authz.CanManageTAs, want: [8]bool{true, true, false, true}},
		{name: "CanReadSection", tx: authz.CanReadSection, want: [8]bool{true, true, false, true, true, true}},
		{name: "CanReadRoster", tx: authz.CanReadRoster, want: [8]bool{true, true, false, true, true}},
	}
	must(pg.WithTx(ctx, db, uni, func(tx *sql.Tx) error {
		for _, row := range rows {
			for i, a := range actors {
				var got bool
				if row.pure != nil {
					got = row.pure(a.s)
				} else {
					var err error
					if got, err = row.tx(ctx, tx, a.s, section); err != nil {
						t.Errorf("%s/%s: %v", row.name, a.name, err)
					}
					// missing section and another university's section: false, no error
					for _, id := range []string{randomID, sectionB} {
						g, err := row.tx(ctx, tx, a.s, id)
						if err != nil || (g && a.s.Role != authz.Admin) {
							t.Errorf("%s/%s on missing %s: %v, %v", row.name, a.name, id, g, err)
						}
					}
				}
				if got != row.want[i] {
					t.Errorf("%s/%s = %v, want %v", row.name, a.name, got, row.want[i])
				}
			}
		}
		return nil
	}))
}
