package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// WithTx runs fn in a transaction with app.university_id set (transaction-local), so row level
// security confines every query to that university. It commits when fn returns nil, otherwise
// rolls back; a panic in fn rolls back and is re-raised.
func WithTx(ctx context.Context, db *sql.DB, universityID string, fn func(tx *sql.Tx) error) error {
	if universityID == "" {
		return errors.New("pg.WithTx: empty university id")
	}
	return run(ctx, db, func(tx *sql.Tx) error {
		if err := SetUniversity(ctx, tx, universityID); err != nil {
			return err
		}
		return fn(tx)
	})
}

// WithPlatformTx sets no tenant setting; only for universities, users, sessions, magic_links and outbox_mails
// (tenant tables return no rows and reject writes).
func WithPlatformTx(ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) error) error {
	return run(ctx, db, fn)
}

// WithUserTx sets app.user_id and/or app.email (empty string = leave unset) and never
// app.university_id. Only auth uses it: SignIn (email) and GET /me (user id).
func WithUserTx(ctx context.Context, db *sql.DB, userID, email string, fn func(tx *sql.Tx) error) error {
	return run(ctx, db, func(tx *sql.Tx) error {
		for _, s := range [][2]string{{"app.user_id", userID}, {"app.email", email}} {
			if s[1] == "" {
				continue
			}
			if _, err := tx.ExecContext(ctx, `select set_config($1, $2, true)`, s[0], s[1]); err != nil {
				return fmt.Errorf("set %s: %w", s[0], err)
			}
		}
		return fn(tx)
	})
}

// SetUniversity sets app.university_id for the rest of the open transaction.
func SetUniversity(ctx context.Context, tx *sql.Tx, universityID string) error {
	if _, err := tx.ExecContext(ctx, `select set_config('app.university_id', $1, true)`, universityID); err != nil {
		return fmt.Errorf("set app.university_id: %w", err)
	}
	return nil
}

func run(ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) error) (err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
		if err != nil {
			tx.Rollback()
		}
	}()
	if err = fn(tx); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
