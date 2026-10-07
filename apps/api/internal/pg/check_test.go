package pg_test

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"sort"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/testdb"
	"github.com/kapildev333/unicourse/apps/api/migrations"
)

// assertGrants: the API role has the four DML privileges on every table and nothing more
// (select only on schema_migrations).
func assertGrants(t *testing.T, api *sql.DB) {
	t.Helper()
	rows, err := api.Query(`select tablename from pg_tables where schemaname = 'public'`)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for rows.Next() {
		var n string
		rows.Scan(&n)
		names = append(names, n)
	}
	rows.Close()
	for _, n := range names {
		for _, priv := range []string{"select", "insert", "update", "delete", "truncate", "references", "trigger"} {
			want := priv == "select" || (n != "schema_migrations" && (priv == "insert" || priv == "update" || priv == "delete"))
			var got bool
			if err := api.QueryRow(`select has_table_privilege(current_user, $1::regclass, $2)`, "public."+n, priv).Scan(&got); err != nil {
				t.Fatal(n, priv, err)
			}
			if got != want {
				t.Errorf("%s: %s = %v, want %v", n, priv, got, want)
			}
		}
	}
}

func TestAPIRoleCannotEscalate(t *testing.T) {
	api, owner := testdb.OpenWithOwner(t)
	ctx := context.Background()
	for _, stmt := range []string{
		`alter table memberships disable row level security`,
		`alter table memberships no force row level security`,
		`drop policy tenant_isolation on memberships`,
		`truncate outbox_mails`,
		`create table p4l2_probe (id int)`,
		`create schema p4l2_probe`,
		`select * from pg_authid limit 1`,
		`insert into schema_migrations (name) values ('x')`,
	} {
		_, err := api.ExecContext(ctx, stmt)
		var pe *pgconn.PgError
		if !errors.As(err, &pe) || pe.Code != "42501" {
			t.Errorf("%s: err = %v, want permission denied (42501)", stmt, err)
		}
	}
	assertGrants(t, api)

	if _, err := owner.ExecContext(ctx, `create table p4l2_probe (id int)`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { owner.Exec(`drop table if exists p4l2_probe`) })
	var dml, trunc bool
	if err := api.QueryRowContext(ctx, `select has_table_privilege('p4l2_probe', 'select') and has_table_privilege('p4l2_probe', 'insert') and has_table_privilege('p4l2_probe', 'update') and has_table_privilege('p4l2_probe', 'delete'), has_table_privilege('p4l2_probe', 'truncate')`).Scan(&dml, &trunc); err != nil {
		t.Fatal(err)
	}
	if !dml || trunc {
		t.Errorf("default privileges on a new table: dml %v truncate %v, want true false", dml, trunc)
	}
	if _, err := owner.ExecContext(ctx, `drop table p4l2_probe`); err != nil {
		t.Fatal(err)
	}

	if err := pg.CheckAPIRole(ctx, api); err != nil {
		t.Errorf("api role: %v", err)
	}
	if err := pg.CheckAPIRole(ctx, owner); err == nil || !strings.Contains(err.Error(), "unicourse_owner") {
		t.Errorf("owner role: err = %v, want refusal naming unicourse_owner", err)
	}
}

func TestCheckMigrated(t *testing.T) {
	api, owner := testdb.OpenWithOwner(t)
	ctx := context.Background()
	if err := pg.CheckMigrated(ctx, api); err != nil {
		t.Fatal(err)
	}
	files, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	last := files[len(files)-1]
	if _, err := owner.Exec(`delete from schema_migrations where name = $1`, last); err != nil {
		t.Fatal(err)
	}
	restore := func() { owner.Exec(`insert into schema_migrations (name) values ($1) on conflict do nothing`, last) }
	t.Cleanup(restore)
	err = pg.CheckMigrated(ctx, api)
	if err == nil || !strings.Contains(err.Error(), last) || !strings.Contains(err.Error(), "unicourse-admin migrate") {
		t.Fatalf("missing %s: err = %v", last, err)
	}
	restore()
	if _, err := owner.Exec(`insert into schema_migrations (name) values ('9999_future.sql')`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { owner.Exec(`delete from schema_migrations where name = '9999_future.sql'`) })
	if err := pg.CheckMigrated(ctx, api); err != nil {
		t.Errorf("database ahead of binary: %v", err)
	}
}
