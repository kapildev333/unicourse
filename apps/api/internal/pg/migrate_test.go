package pg_test

import (
	"context"
	"io/fs"
	"testing"

	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/testdb"
	"github.com/kapildev333/unicourse/apps/api/migrations"
)

var tenantTables = []string{"memberships", "department_admins", "departments", "terms", "courses", "sections",
	"section_staff", "enrollments", "invitations", "audit_log", "pending_section_staff", "pending_enrollments", "section_meetings",
	"grade_scales", "assessments", "scores", "final_grades"}

func TestMigrate(t *testing.T) {
	api, db := testdb.OpenWithOwner(t)
	ctx := context.Background()
	// fresh database: the owner owns public, so it can be recreated
	if _, err := db.ExecContext(ctx, `drop schema public cascade; create schema public`); err != nil {
		t.Fatal(err)
	}
	files, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil {
		t.Fatal(err)
	}
	for i := range 2 { // second call must apply nothing
		if err := pg.Migrate(ctx, db); err != nil {
			t.Fatal(err)
		}
		var n int
		db.QueryRow(`select count(*) from schema_migrations`).Scan(&n)
		if n != len(files) {
			t.Fatalf("run %d: %d schema_migrations rows, want %d", i, n, len(files))
		}
	}
	var tables int
	db.QueryRow(`select count(*) from pg_tables where schemaname = 'public' and tablename <> 'schema_migrations'`).Scan(&tables)
	if tables != 22 {
		t.Fatalf("%d tables, want 22", tables)
	}
	rows, err := db.Query(`select table_name from information_schema.columns where table_schema = 'public' and column_name = 'university_id' and table_name <> 'universities'`)
	if err != nil {
		t.Fatal(err)
	}
	var tenant []string
	for rows.Next() {
		var n string
		rows.Scan(&n)
		tenant = append(tenant, n)
	}
	rows.Close()
	if len(tenant) != 17 {
		t.Fatalf("%d tenant tables %v, want 17", len(tenant), tenant)
	}
	for _, name := range tenant {
		var on, force bool
		if err := db.QueryRow(`select relrowsecurity, relforcerowsecurity from pg_class where oid = $1::regclass`, name).Scan(&on, &force); err != nil {
			t.Fatal(name, err)
		}
		if !on || !force {
			t.Errorf("%s: rls %v force %v", name, on, force)
		}
		var n int
		db.QueryRow(`select count(*) from pg_policies where schemaname = 'public' and tablename = $1 and policyname = 'tenant_isolation' and qual is not null and with_check is not null`, name).Scan(&n)
		if n != 1 {
			t.Errorf("%s: tenant_isolation policy with qual and with_check missing", name)
		}
	}
	for _, name := range []string{"universities", "users", "sessions", "magic_links", "outbox_mails"} {
		var on bool
		db.QueryRow(`select relrowsecurity from pg_class where oid = $1::regclass`, name).Scan(&on)
		if on {
			t.Errorf("%s must not have rls", name)
		}
	}
	// 0004 replays the grants on a recreated schema
	assertGrants(t, api)
	if err := pg.CheckAPIRole(ctx, api); err != nil {
		t.Error(err)
	}
}

func TestMigrateConcurrent(t *testing.T) {
	_, db := testdb.OpenWithOwner(t)
	ctx := context.Background()
	db.ExecContext(ctx, `drop schema public cascade; create schema public`)
	errs := make(chan error, 4)
	for range 4 {
		go func() { errs <- pg.Migrate(ctx, db) }()
	}
	for range 4 {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
}
