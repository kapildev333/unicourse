package pg

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"sort"

	"github.com/kapildev333/unicourse/apps/api/migrations"
)

const migrateLockKey = 7263550 // advisory lock: two API processes starting together must not race

// Migrate applies migrations/*.sql in name order, each in its own transaction together with its
// schema_migrations row. Already-applied files are skipped.
func Migrate(ctx context.Context, db *sql.DB) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `select pg_advisory_lock($1)`, migrateLockKey); err != nil {
		return fmt.Errorf("migrate: lock: %w", err)
	}
	defer conn.ExecContext(context.WithoutCancel(ctx), `select pg_advisory_unlock($1)`, migrateLockKey)

	if _, err := conn.ExecContext(ctx, `create table if not exists schema_migrations (name text primary key, applied_at timestamptz not null default now())`); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	files, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, name := range files {
		var done bool
		if err := conn.QueryRowContext(ctx, `select exists (select 1 from schema_migrations where name = $1)`, name).Scan(&done); err != nil {
			return fmt.Errorf("migrate %s: %w", name, err)
		}
		if done {
			continue
		}
		body, err := fs.ReadFile(migrations.FS, name)
		if err != nil {
			return err
		}
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("migrate %s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx, string(body)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migrate %s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx, `insert into schema_migrations (name) values ($1)`, name); err != nil {
			tx.Rollback()
			return fmt.Errorf("migrate %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("migrate %s: %w", name, err)
		}
	}
	return nil
}
