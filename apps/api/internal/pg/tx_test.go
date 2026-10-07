package pg_test

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/testdb"
)

var ctx = context.Background()

type ids struct{ dept, term, course, section, user, inv string }

// rest are the insert statements for the remaining tenant tables; {U} is the university id.
var rest = map[string]string{
	"memberships":           `insert into memberships (university_id, user_id, role, status) values ('{U}', '{USR}', 'member', 'active')`,
	"department_admins":     `insert into department_admins (university_id, department_id, user_id) values ('{U}', '{D}', '{USR}')`,
	"section_staff":         `insert into section_staff (university_id, section_id, user_id, role) values ('{U}', '{S}', '{USR}', 'instructor')`,
	"enrollments":           `insert into enrollments (university_id, section_id, user_id) values ('{U}', '{S}', '{USR}')`,
	"invitations":           `insert into invitations (university_id, email, role, token_hash, expires_at) values ('{U}', 'x@b.edu', 'member', sha256(gen_random_uuid()::text::bytea), now() + interval '1 day')`,
	"pending_section_staff": `insert into pending_section_staff (university_id, invitation_id, section_id, role) values ('{U}', '{INV}', '{S}', 'ta')`,
	"pending_enrollments":   `insert into pending_enrollments (university_id, invitation_id, section_id) values ('{U}', '{INV}', '{S}')`,
	"section_meetings":      `insert into section_meetings (university_id, section_id, weekday, starts_at, ends_at) values ('{U}', '{S}', 1, '10:00', '11:00')`,
	"audit_log":             `insert into audit_log (university_id, action, entity_type, entity_id) values ('{U}', 'create', 'section', '{S}')`,
}

func sub(q, uni string, i ids) string {
	return strings.NewReplacer("{U}", uni, "{D}", i.dept, "{S}", i.section, "{USR}", i.user, "{INV}", i.inv).Replace(q)
}

// populate fills every tenant table for uni inside an open WithTx(uni) transaction.
func populate(t *testing.T, tx *sql.Tx, uni, user string) ids {
	t.Helper()
	i := ids{user: user}
	q := func(dst *string, query string) {
		if err := tx.QueryRow(query).Scan(dst); err != nil {
			t.Fatal(query, err)
		}
	}
	q(&i.dept, `insert into departments (university_id, name, code) values ('`+uni+`', 'D', 'D1') returning id`)
	q(&i.term, `insert into terms (university_id, name, starts_on, ends_on) values ('`+uni+`', 'T', '2026-01-01', '2026-06-01') returning id`)
	q(&i.course, `insert into courses (university_id, department_id, code, title) values ('`+uni+`', '`+i.dept+`', 'C1', 'C') returning id`)
	q(&i.section, `insert into sections (university_id, course_id, term_id, number) values ('`+uni+`', '`+i.course+`', '`+i.term+`', '001') returning id`)
	q(&i.inv, sub(rest["invitations"], uni, i)+` returning id`)
	for name, stmt := range rest {
		if name == "invitations" {
			continue
		}
		if _, err := tx.Exec(sub(stmt, uni, i)); err != nil {
			t.Fatal(name, err)
		}
	}
	return i
}

func count(t *testing.T, tx *sql.Tx, table string) (n int) {
	t.Helper()
	if err := tx.QueryRow(`select count(*) from ` + table).Scan(&n); err != nil {
		t.Fatal(table, err)
	}
	return n
}

func newUser(t *testing.T, db *sql.DB, email string) (id string) {
	t.Helper()
	err := pg.WithPlatformTx(ctx, db, func(tx *sql.Tx) error {
		return tx.QueryRow(`insert into users (email) values ($1) returning id`, email).Scan(&id)
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestRoleDoesNotBypassRLS(t *testing.T) {
	db := testdb.Open(t) // fails loudly itself; assert directly too
	var bypass, super bool
	db.QueryRow(`select rolbypassrls, rolsuper from pg_roles where rolname = current_user`).Scan(&bypass, &super)
	if bypass || super {
		t.Fatal("role bypasses RLS")
	}
}

func TestWithTxIsolatesUniversities(t *testing.T) {
	db := testdb.Open(t)
	f := testdb.Seed(t, db)
	user := newUser(t, db, "u@b.edu")

	var a ids
	if err := pg.WithTx(ctx, db, f.UniversityA, func(tx *sql.Tx) error { a = populate(t, tx, f.UniversityA, user); return nil }); err != nil {
		t.Fatal(err)
	}
	// every tenant table: inserting a row for B while scoped to A is rejected by the policy
	attempts := map[string]string{
		"departments": `insert into departments (university_id, name, code) values ('{U}', 'D', 'ZZ')`,
		"terms":       `insert into terms (university_id, name, starts_on, ends_on) values ('{U}', 'T', '2026-01-01', '2026-06-01')`,
		"courses":     `insert into courses (university_id, department_id, code, title) values ('{U}', '{D}', 'ZZ', 'C')`,
		"sections":    `insert into sections (university_id, course_id, term_id, number) values ('{U}', (select id from courses limit 1), (select id from terms limit 1), '002')`,
	}
	for k, v := range rest {
		attempts[k] = v
	}
	if len(attempts) != len(tenantTables) {
		t.Fatalf("%d attempts for %d tenant tables", len(attempts), len(tenantTables))
	}
	for name, stmt := range attempts {
		err := pg.WithTx(ctx, db, f.UniversityA, func(tx *sql.Tx) error {
			_, err := tx.Exec(sub(stmt, f.UniversityB, a))
			return err
		})
		var pe *pgconn.PgError
		if !errors.As(err, &pe) || pe.Code != "42501" {
			t.Errorf("%s: cross-tenant insert error = %v, want 42501", name, err)
		}
	}
	// B sees nothing; A sees its rows
	pg.WithTx(ctx, db, f.UniversityB, func(tx *sql.Tx) error {
		for _, tb := range tenantTables {
			if tb == "memberships" {
				continue // seed gave B an admin membership
			}
			if n := count(t, tx, tb); n != 0 && tb != "memberships" {
				t.Errorf("B sees %d rows of %s", n, tb)
			}
		}
		if n := count(t, tx, "memberships"); n != 1 {
			t.Errorf("B sees %d memberships, want its own 1", n)
		}
		return nil
	})
	pg.WithTx(ctx, db, f.UniversityA, func(tx *sql.Tx) error {
		for _, tb := range tenantTables {
			if n := count(t, tx, tb); n == 0 {
				t.Errorf("A sees no rows of %s", tb)
			}
		}
		return nil
	})
	// no tenant setting at all: nothing visible, nothing writable
	pg.WithPlatformTx(ctx, db, func(tx *sql.Tx) error {
		for _, tb := range tenantTables {
			if n := count(t, tx, tb); n != 0 {
				t.Errorf("platform tx sees %d rows of %s", n, tb)
			}
		}
		return nil
	})
	if err := pg.WithTx(ctx, db, "", func(*sql.Tx) error { return nil }); err == nil {
		t.Error("empty university id accepted")
	}
}

func TestUserInTwoUniversities(t *testing.T) {
	db := testdb.Open(t)
	f := testdb.Seed(t, db)
	u := newUser(t, db, "u@b.edu")
	other := newUser(t, db, "v@b.edu")
	for _, uni := range []string{f.UniversityA, f.UniversityB} {
		pg.WithTx(ctx, db, uni, func(tx *sql.Tx) error { populate(t, tx, uni, u); return nil })
	}
	pg.WithTx(ctx, db, f.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`insert into memberships (university_id, user_id, role, status) values ($1, $2, 'member', 'active')`, f.UniversityA, other)
		if err != nil {
			t.Fatal(err)
		}
		_, err = tx.Exec(`insert into invitations (university_id, email, role, token_hash, expires_at) values ($1, 'other@b.edu', 'member', sha256(gen_random_uuid()::text::bytea), now())`, f.UniversityA)
		if err != nil {
			t.Fatal(err)
		}
		return nil
	})
	// WithTx(A): only A's membership of u
	pg.WithTx(ctx, db, f.UniversityA, func(tx *sql.Tx) error {
		var n int
		tx.QueryRow(`select count(*) from memberships where user_id = $1`, u).Scan(&n)
		if n != 1 {
			t.Errorf("WithTx(A) sees %d memberships of u, want 1", n)
		}
		return nil
	})
	// WithUserTx(u): own memberships and department_admins everywhere, no other tenant data
	pg.WithUserTx(ctx, db, u, "", func(tx *sql.Tx) error {
		if n := count(t, tx, "memberships"); n != 2 {
			t.Errorf("memberships = %d, want 2 (own, A and B only)", n)
		}
		if n := count(t, tx, "department_admins"); n != 2 {
			t.Errorf("department_admins = %d, want 2", n)
		}
		for _, tb := range tenantTables {
			if tb == "memberships" || tb == "department_admins" {
				continue
			}
			if n := count(t, tx, tb); n != 0 {
				t.Errorf("user tx sees %d rows of %s", n, tb)
			}
		}
		_, err := tx.Exec(`insert into memberships (university_id, user_id, role, status) values ($1, $2, 'admin', 'active')`, f.UniversityA, other)
		var pe *pgconn.PgError
		if !errors.As(err, &pe) || pe.Code != "42501" {
			t.Errorf("membership insert in user tx: %v, want 42501", err)
		}
		return nil
	})
	// WithUserTx("", email): only invitations with that exact email
	pg.WithUserTx(ctx, db, "", "x@b.edu", func(tx *sql.Tx) error {
		if n := count(t, tx, "invitations"); n != 2 {
			t.Errorf("invitations = %d, want 2", n)
		}
		if n := count(t, tx, "memberships"); n != 0 {
			t.Errorf("email-only tx sees %d memberships", n)
		}
		return nil
	})
}

func TestRollbackAndPanic(t *testing.T) {
	db := testdb.Open(t)
	f := testdb.Seed(t, db)
	boom := errors.New("boom")
	insert := func(tx *sql.Tx) {
		if _, err := tx.Exec(`insert into departments (university_id, name, code) values ($1, 'X', 'X')`, f.UniversityA); err != nil {
			t.Fatal(err)
		}
	}
	check := func(want int) {
		t.Helper()
		pg.WithTx(ctx, db, f.UniversityA, func(tx *sql.Tx) error {
			var n int
			tx.QueryRow(`select count(*) from departments where code = 'X'`).Scan(&n)
			if n != want {
				t.Errorf("departments X = %d, want %d", n, want)
			}
			return nil
		})
	}
	if err := pg.WithTx(ctx, db, f.UniversityA, func(tx *sql.Tx) error { insert(tx); return boom }); !errors.Is(err, boom) {
		t.Fatal(err)
	}
	check(0)
	func() {
		defer func() {
			if r := recover(); r != "kaboom" {
				t.Errorf("recovered %v, want re-raised panic", r)
			}
		}()
		pg.WithTx(ctx, db, f.UniversityA, func(tx *sql.Tx) error { insert(tx); panic("kaboom") })
	}()
	check(0)
	pg.WithTx(ctx, db, f.UniversityA, func(tx *sql.Tx) error { insert(tx); return nil })
	check(1)
}

// A row of A must not reference a parent row of B, even by a valid id: composite FKs give 23503.
func TestCrossTenantReferencesRejected(t *testing.T) {
	db := testdb.Open(t)
	f := testdb.Seed(t, db)
	user := newUser(t, db, "u@b.edu")
	var b ids
	pg.WithTx(ctx, db, f.UniversityB, func(tx *sql.Tx) error { b = populate(t, tx, f.UniversityB, user); return nil })
	cases := map[string]string{
		"courses->departments":               `insert into courses (university_id, department_id, code, title) values ('{U}', '{D}', 'X', 'X')`,
		"sections->courses":                  `insert into sections (university_id, course_id, term_id, number) values ('{U}', '` + b.course + `', '` + f.TermA + `', '9')`,
		"sections->terms":                    `insert into sections (university_id, course_id, term_id, number) values ('{U}', (select id from courses limit 1), '` + b.term + `', '9')`,
		"department_admins->departments":     rest["department_admins"],
		"section_staff->sections":            rest["section_staff"],
		"enrollments->sections":              rest["enrollments"],
		"section_meetings->sections":         rest["section_meetings"],
		"pending_section_staff->sections":    `insert into pending_section_staff (university_id, invitation_id, section_id, role) values ('{U}', (select id from invitations limit 1), '{S}', 'ta')`,
		"pending_section_staff->both":        `insert into pending_section_staff (university_id, invitation_id, section_id, role) values ('{U}', '{INV}', '{S}', 'ta')`,
		"pending_enrollments->sections":      `insert into pending_enrollments (university_id, invitation_id, section_id) values ('{U}', (select id from invitations limit 1), '{S}')`,
		"pending_enrollments->both":          rest["pending_enrollments"],
		"pending_enrollments->invitations":   `insert into pending_enrollments (university_id, invitation_id, section_id) values ('{U}', '{INV}', (select id from sections limit 1))`,
		"pending_section_staff->invitations": `insert into pending_section_staff (university_id, invitation_id, section_id, role) values ('{U}', '{INV}', (select id from sections limit 1), 'ta')`,
	}
	// A-side parents so the "limit 1" subselects resolve
	pg.WithTx(ctx, db, f.UniversityA, func(tx *sql.Tx) error {
		for _, q := range []string{
			`insert into courses (university_id, department_id, code, title) values ('{U}', '` + f.DeptA + `', 'A1', 'A')`,
			`insert into sections (university_id, course_id, term_id, number) select '{U}', id, '` + f.TermA + `', '1' from courses`,
			`insert into invitations (university_id, email, role, token_hash, expires_at) values ('{U}', 'a@a.edu', 'member', sha256('a'::bytea), now())`,
		} {
			if _, err := tx.Exec(sub(q, f.UniversityA, b)); err != nil {
				t.Fatal(err)
			}
		}
		return nil
	})
	// b.user already holds the same (parent, user) pairs in B: the error must still be the FK's, not a unique's
	for name, stmt := range cases {
		err := pg.WithTx(ctx, db, f.UniversityA, func(tx *sql.Tx) error {
			_, err := tx.Exec(sub(stmt, f.UniversityA, b))
			return pg.MapError(err)
		})
		if !errors.Is(err, pg.ErrNotFound) {
			t.Errorf("%s: %v, want ErrNotFound", name, err)
		}
	}
}

func TestSelfPoliciesDoNotWidenTenantReads(t *testing.T) {
	db := testdb.Open(t)
	f := testdb.Seed(t, db)
	u := newUser(t, db, "u@b.edu")
	for _, uni := range []string{f.UniversityA, f.UniversityB} {
		pg.WithTx(ctx, db, uni, func(tx *sql.Tx) error { populate(t, tx, uni, u); return nil })
	}
	err := pg.WithUserTx(ctx, db, u, "x@b.edu", func(tx *sql.Tx) error {
		if n := count(t, tx, "memberships"); n != 2 {
			t.Fatalf("user tx memberships = %d, want 2", n)
		}
		if err := pg.SetUniversity(ctx, tx, f.UniversityA); err != nil {
			t.Fatal(err)
		}
		for _, tb := range []string{"memberships", "department_admins", "invitations"} {
			var n int
			tx.QueryRow(`select count(*) from `+tb+` where university_id <> $1`, f.UniversityA).Scan(&n)
			if n != 0 {
				t.Errorf("%s with user and university A shows %d rows of other universities", tb, n)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdatedAtTrigger(t *testing.T) {
	db := testdb.Open(t)
	f := testdb.Seed(t, db)
	for _, c := range []struct{ table, id, set string }{
		{"departments", f.DeptA, "name = 'Renamed'"},
		{"users", f.AdminA, "display_name = 'Renamed'"},
	} {
		run := func() error {
			if c.table == "users" {
				return pg.WithPlatformTx(ctx, db, func(tx *sql.Tx) error {
					_, err := tx.Exec(`update users set `+c.set+` where id = $1`, c.id)
					return err
				})
			}
			return pg.WithTx(ctx, db, f.UniversityA, func(tx *sql.Tx) error {
				_, err := tx.Exec(`update `+c.table+` set `+c.set+` where id = $1`, c.id)
				return err
			})
		}
		if err := run(); err != nil { // a later transaction than the seed's
			t.Fatal(err)
		}
		var ok bool
		pg.WithTx(ctx, db, f.UniversityA, func(tx *sql.Tx) error {
			return tx.QueryRow(`select updated_at > created_at from `+c.table+` where id = $1`, c.id).Scan(&ok)
		})
		if !ok {
			t.Errorf("%s: updated_at not advanced by update", c.table)
		}
	}
}

func TestEveryUpdatedAtTableHasTrigger(t *testing.T) {
	db := testdb.Open(t)
	rows, err := db.Query(`select c.table_name from information_schema.columns c
		where c.table_schema = 'public' and c.column_name = 'updated_at'
		and not exists (select 1 from pg_trigger g where g.tgrelid = format('public.%I', c.table_name)::regclass
			and not g.tgisinternal and g.tgfoid = 'set_updated_at'::regproc)`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		rows.Scan(&name)
		t.Errorf("%s has updated_at but no set_updated_at trigger", name)
	}
}
