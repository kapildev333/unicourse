package pg_test

import (
	"context"
	"testing"

	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/testdb"
)

var tenantTables = []string{"memberships", "department_admins", "departments", "terms", "courses", "sections",
	"section_staff", "enrollments", "invitations", "audit_log", "pending_section_staff"}

func TestMigrate(t *testing.T) {
	db := testdb.Open(t)
	ctx := context.Background()
	// fresh database: the role owns public, so it can be recreated
	if _, err := db.ExecContext(ctx, `drop schema public cascade; create schema public`); err != nil {
		t.Fatal(err)
	}
	for i := range 2 { // second call must apply nothing
		if err := pg.Migrate(ctx, db); err != nil {
			t.Fatal(err)
		}
		var n int
		db.QueryRow(`select count(*) from schema_migrations`).Scan(&n)
		if n != 1 {
			t.Fatalf("run %d: %d schema_migrations rows", i, n)
		}
	}
	var tables int
	db.QueryRow(`select count(*) from pg_tables where schemaname = 'public' and tablename <> 'schema_migrations'`).Scan(&tables)
	if tables != 15 {
		t.Fatalf("%d tables, want 15", tables)
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
	if len(tenant) != 11 {
		t.Fatalf("%d tenant tables %v, want 11", len(tenant), tenant)
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
	for _, name := range []string{"universities", "users", "sessions", "magic_links"} {
		var on bool
		db.QueryRow(`select relrowsecurity from pg_class where oid = $1::regclass`, name).Scan(&on)
		if on {
			t.Errorf("%s must not have rls", name)
		}
	}
}

func TestMigrateConcurrent(t *testing.T) {
	db := testdb.Open(t)
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
