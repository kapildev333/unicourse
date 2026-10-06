package pg_test

import (
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/testdb"
)

func TestMapError(t *testing.T) {
	db := testdb.Open(t)
	f := testdb.Seed(t, db)
	missing := "00000000-0000-0000-0000-000000000001"

	// duplicate department code -> conflict, constraint name kept
	err := pg.WithTx(ctx, db, f.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`insert into departments (university_id, name, code) values ($1, 'Dup', 'CS')`, f.UniversityA)
		return pg.MapError(err)
	})
	if !errors.Is(err, pg.ErrConflict) || errors.Is(err, pg.ErrNotFound) {
		t.Fatalf("duplicate: %v", err)
	}
	if want := "departments_university_id_code_key"; !strings.Contains(err.Error(), want) {
		t.Errorf("constraint name missing from %q", err)
	}

	// FK to a missing term -> not found
	err = pg.WithTx(ctx, db, f.UniversityA, func(tx *sql.Tx) error {
		var course string
		if err := tx.QueryRow(`insert into courses (university_id, department_id, code, title) values ($1, $2, 'C1', 'C') returning id`, f.UniversityA, f.DeptA).Scan(&course); err != nil {
			t.Fatal(err)
		}
		_, err := tx.Exec(`insert into sections (university_id, course_id, term_id, number) values ($1, $2, $3, '1')`, f.UniversityA, course, missing)
		return pg.MapError(err)
	})
	if !errors.Is(err, pg.ErrNotFound) {
		t.Fatalf("missing term: %v", err)
	}

	if err := pg.MapError(sql.ErrNoRows); !errors.Is(err, pg.ErrNotFound) {
		t.Fatalf("no rows: %v", err)
	}
	other := errors.New("other")
	if pg.MapError(other) != other || pg.MapDeleteError(other) != other || pg.MapError(nil) != nil || pg.MapDeleteError(nil) != nil {
		t.Fatal("unrelated errors must pass through")
	}
	if err := pg.MapDeleteError(sql.ErrNoRows); !errors.Is(err, pg.ErrNotFound) {
		t.Fatalf("delete no rows: %v", err)
	}
}

func TestMapDeleteError(t *testing.T) {
	db := testdb.Open(t)
	f := testdb.Seed(t, db)
	// deleting a department that a course still references -> conflict (restrict), not not-found
	err := pg.WithTx(ctx, db, f.UniversityA, func(tx *sql.Tx) error {
		if _, err := tx.Exec(`insert into courses (university_id, department_id, code, title) values ($1, $2, 'C1', 'C')`, f.UniversityA, f.DeptA); err != nil {
			t.Fatal(err)
		}
		_, err := tx.Exec(`delete from departments where id = $1`, f.DeptA)
		return pg.MapDeleteError(err)
	})
	if !errors.Is(err, pg.ErrConflict) || errors.Is(err, pg.ErrNotFound) {
		t.Fatalf("delete referenced: %v", err)
	}
}
