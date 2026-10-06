package pg

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

// MapError turns database errors into sentinels: no rows and 23503 (foreign key to a missing
// parent) are ErrNotFound, 23505 (unique) is ErrConflict; the constraint name is kept in the text.
// 23514 (check) and 22P02 (invalid text) stay unmapped (500): handlers validate first.
func MapError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: %w", ErrNotFound, err)
	}
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		switch pe.Code {
		case "23505":
			return fmt.Errorf("%w: %s: %w", ErrConflict, pe.ConstraintName, err)
		case "23503":
			return fmt.Errorf("%w: %s: %w", ErrNotFound, pe.ConstraintName, err)
		}
	}
	return err
}

// MapDeleteError is MapError for DELETE statements, where 23503 means the row is still
// referenced: ErrConflict.
func MapDeleteError(err error) error {
	var pe *pgconn.PgError
	if errors.As(err, &pe) && pe.Code == "23503" {
		return fmt.Errorf("%w: %s: %w", ErrConflict, pe.ConstraintName, err)
	}
	return MapError(err)
}
