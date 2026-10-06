// Package testdb gives integration tests a migrated, empty database.
package testdb

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/kapildev333/unicourse/apps/api/internal/pg"
)

// lockKey serialises DB test packages (go test runs packages in parallel against one database):
// Open holds a session advisory lock until the test's cleanup finishes.
const lockKey = 7263551

// Open connects to UNICOURSE_TEST_DATABASE_URL (skips when unset), fails loudly when the role is
// a superuser or bypasses RLS, migrates, and truncates every table before and after the test.
func Open(t *testing.T) *sql.DB {
	t.Helper()
	url := os.Getenv("UNICOURSE_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("UNICOURSE_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	db, err := pg.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lock.ExecContext(ctx, `select pg_advisory_lock($1)`, lockKey); err != nil {
		t.Fatal(err)
	}
	var unsafe bool
	if err := db.QueryRowContext(ctx, `select rolsuper or rolbypassrls from pg_roles where rolname = current_user`).Scan(&unsafe); err != nil {
		t.Fatal(err)
	}
	if unsafe {
		t.Fatal("test database role is a superuser or has BYPASSRLS: row level security would not apply; connect as role unicourse")
	}
	if err := pg.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	truncate(t, db)
	t.Cleanup(func() {
		truncate(t, db)
		lock.ExecContext(ctx, `select pg_advisory_unlock($1)`, lockKey)
		lock.Close()
		db.Close()
	})
	return db
}

func truncate(t *testing.T, db *sql.DB) {
	t.Helper()
	err := pg.WithPlatformTx(context.Background(), db, func(tx *sql.Tx) error {
		var tables string
		if err := tx.QueryRow(`select coalesce(string_agg(quote_ident(tablename), ','), '') from pg_tables where schemaname = 'public' and tablename <> 'schema_migrations'`).Scan(&tables); err != nil || tables == "" {
			return err
		}
		_, err := tx.Exec(`truncate ` + tables + ` restart identity cascade`)
		return err
	})
	if err != nil {
		t.Fatal("truncate:", err)
	}
}

type Fixture struct{ UniversityA, UniversityB, AdminA, AdminB, DeptA, TermA string }

// Seed inserts universities a and b (a.edu, b.edu), an active admin in each, and a department and
// a term in A.
func Seed(t *testing.T, db *sql.DB) Fixture {
	t.Helper()
	ctx := context.Background()
	var f Fixture
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal("seed:", err)
		}
	}
	must(pg.WithPlatformTx(ctx, db, func(tx *sql.Tx) error {
		for _, u := range []struct {
			slug string
			uni  *string
			adm  *string
		}{{"a", &f.UniversityA, &f.AdminA}, {"b", &f.UniversityB, &f.AdminB}} {
			if err := tx.QueryRow(`insert into universities (name, slug, allowed_email_domains, timezone) values ($1, $2, $3, 'America/New_York') returning id`,
				"University "+u.slug, u.slug, "{"+u.slug+".edu}").Scan(u.uni); err != nil {
				return err
			}
			if err := tx.QueryRow(`insert into users (email, display_name) values ($1, 'Admin') returning id`, "admin@"+u.slug+".edu").Scan(u.adm); err != nil {
				return err
			}
		}
		return nil
	}))
	for _, u := range [][2]string{{f.UniversityA, f.AdminA}, {f.UniversityB, f.AdminB}} {
		must(pg.WithTx(ctx, db, u[0], func(tx *sql.Tx) error {
			_, err := tx.Exec(`insert into memberships (university_id, user_id, role, status) values ($1, $2, 'admin', 'active')`, u[0], u[1])
			return err
		}))
	}
	must(pg.WithTx(ctx, db, f.UniversityA, func(tx *sql.Tx) error {
		if err := tx.QueryRow(`insert into departments (university_id, name, code) values ($1, 'Computer Science', 'CS') returning id`, f.UniversityA).Scan(&f.DeptA); err != nil {
			return err
		}
		return tx.QueryRow(`insert into terms (university_id, name, starts_on, ends_on) values ($1, 'Fall 2026', '2026-09-01', '2026-12-15') returning id`, f.UniversityA).Scan(&f.TermA)
	}))
	return f
}
