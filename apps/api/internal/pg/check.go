package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/kapildev333/unicourse/apps/api/migrations"
)

// CheckAPIRole fails when the connected role could switch row level security off or bypass it:
// a superuser, a BYPASSRLS role, or a member of the role owning schema public or any relation in it.
func CheckAPIRole(ctx context.Context, db *sql.DB) error {
	var role string
	var unsafe bool
	err := db.QueryRowContext(ctx, `select current_user, rolsuper or rolbypassrls
    or pg_has_role(current_user, (select nspowner from pg_namespace where nspname = 'public'), 'MEMBER')
    or exists (select 1 from pg_class where relnamespace = 'public'::regnamespace and pg_has_role(current_user, relowner, 'MEMBER'))
from pg_roles where rolname = current_user`).Scan(&role, &unsafe)
	if err != nil {
		return fmt.Errorf("check database role: %w", err)
	}
	if unsafe {
		return fmt.Errorf("database role %q is a superuser, bypasses row level security or owns the schema: UNICOURSE_DATABASE_URL must use the API role (unicourse), not the owner", role)
	}
	return nil
}

// CheckMigrated fails when a migration embedded in this binary has no schema_migrations row.
// Rows for migrations the binary does not know are fine (a newer binary migrated first).
func CheckMigrated(ctx context.Context, db *sql.DB) error {
	files, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)
	applied := map[string]bool{}
	rows, err := db.QueryContext(ctx, `select name from schema_migrations`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var n string
			if err := rows.Scan(&n); err != nil {
				return fmt.Errorf("check migrations: %w", err)
			}
			applied[n] = true
		}
		err = rows.Err()
	}
	var pe *pgconn.PgError
	if err != nil && !(errors.As(err, &pe) && pe.Code == "42P01") { // 42P01: never migrated
		return fmt.Errorf("check migrations: %w", err)
	}
	var missing []string
	for _, f := range files {
		if !applied[f] {
			missing = append(missing, f)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("database is missing migrations %s: run unicourse-admin migrate", strings.Join(missing, ", "))
	}
	return nil
}
