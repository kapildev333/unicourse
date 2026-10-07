// Package testdb gives integration tests a migrated, empty database, migrated by the owner role and used as the API role.
package testdb

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	"github.com/kapildev333/unicourse/apps/api/internal/pg"
)

// lockKey serialises DB test packages (go test runs packages in parallel against one database):
// Open holds a session advisory lock until the test's cleanup finishes.
const lockKey = 7263551

// Open is OpenWithOwner without the owner pool; nearly every test wants only the API pool.
func Open(t *testing.T) *sql.DB {
	t.Helper()
	api, _ := OpenWithOwner(t)
	return api
}

// OpenWithOwner connects twice to the same _test database: as the owner (UNICOURSE_TEST_OWNER_DATABASE_URL),
// which migrates and truncates, and as the API role (UNICOURSE_TEST_DATABASE_URL), which the tests
// use like the API does. Skips when UNICOURSE_TEST_DATABASE_URL is unset.
func OpenWithOwner(t *testing.T) (api, owner *sql.DB) {
	t.Helper()
	apiURL := os.Getenv("UNICOURSE_TEST_DATABASE_URL")
	if apiURL == "" {
		t.Skip("UNICOURSE_TEST_DATABASE_URL not set")
	}
	ownerURL := os.Getenv("UNICOURSE_TEST_OWNER_DATABASE_URL")
	if ownerURL == "" {
		t.Fatal("UNICOURSE_TEST_OWNER_DATABASE_URL not set: testdb migrates and truncates as the owner role (unicourse_owner)")
	}
	ctx := context.Background()
	owner, err := pg.Open(ctx, ownerURL)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := owner.Conn(ctx)
	if err != nil {
		owner.Close()
		t.Fatal(err)
	}
	if _, err := lock.ExecContext(ctx, `select pg_advisory_lock($1)`, lockKey); err != nil {
		lock.Close()
		owner.Close()
		t.Fatal(err)
	}
	// registered right after the lock so a later failure cannot leave it held
	cleaned := false
	t.Cleanup(func() {
		if cleaned {
			truncate(t, owner)
		}
		lock.ExecContext(ctx, `select pg_advisory_unlock($1)`, lockKey)
		lock.Close()
		owner.Close()
		if api != nil {
			api.Close()
		}
	})
	var name string
	if err := owner.QueryRowContext(ctx, `select current_database()`).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(name, "_test") {
		t.Fatalf("refusing to run: database %q does not end in _test and testdb truncates every table", name)
	}
	var unsafe bool
	if err := owner.QueryRowContext(ctx, `select rolsuper or rolbypassrls from pg_roles where rolname = current_user`).Scan(&unsafe); err != nil {
		t.Fatal(err)
	}
	if unsafe {
		t.Fatal("test owner role is a superuser or has BYPASSRLS: row level security would not apply to the CLI tests; connect as role unicourse_owner")
	}
	if err := pg.Migrate(ctx, owner); err != nil {
		t.Fatal(err)
	}
	truncate(t, owner)
	cleaned = true
	if api, err = pg.Open(ctx, apiURL); err != nil {
		t.Fatal(err)
	}
	var apiName string
	if err := api.QueryRowContext(ctx, `select current_database()`).Scan(&apiName); err != nil {
		t.Fatal(err)
	}
	if apiName != name {
		t.Fatalf("UNICOURSE_TEST_DATABASE_URL and UNICOURSE_TEST_OWNER_DATABASE_URL point at different databases (%q, %q)", apiName, name)
	}
	if err := pg.CheckAPIRole(ctx, api); err != nil {
		t.Fatal(err)
	}
	return api, owner
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
