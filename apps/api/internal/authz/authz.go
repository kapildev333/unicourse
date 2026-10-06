// Package authz holds the role rules: pure predicates on Scope and section predicates that read the tenant tx.
package authz

import (
	"context"
	"database/sql"
	"errors"
	"slices"
)

type Role string

const (
	Admin     Role = "admin"
	DeptAdmin Role = "dept_admin"
	Member    Role = "member"
)

// Scope is the caller's standing in one university; DeptIDs is set only for dept_admin.
type Scope struct {
	UniversityID, UserID string
	Role                 Role
	DeptIDs              []string
}

type ctxKey struct{}

// WithScope is for package tenant only.
func WithScope(ctx context.Context, s Scope) context.Context {
	return context.WithValue(ctx, ctxKey{}, s)
}

func ScopeFromContext(ctx context.Context) (Scope, bool) {
	s, ok := ctx.Value(ctxKey{}).(Scope)
	return s, ok
}

func IsAdmin(s Scope) bool { return s.Role == Admin }

func CanManageDepartment(s Scope, deptID string) bool {
	return s.Role == Admin || (s.Role == DeptAdmin && slices.Contains(s.DeptIDs, deptID))
}

func CanInvite(s Scope, role Role) bool {
	switch s.Role {
	case Admin:
		return true
	case DeptAdmin:
		return role == Member
	}
	return false
}

// The section predicates return false (not an error) for a section that does not exist or is
// invisible to the tenant tx. Callers load the entity first (404), then ask (403).

func CanManageSection(ctx context.Context, tx *sql.Tx, s Scope, sectionID string) (bool, error) {
	switch s.Role {
	case Admin:
		return true, nil
	case DeptAdmin:
		var deptID string
		err := tx.QueryRowContext(ctx, `select c.department_id from sections s join courses c on c.id = s.course_id where s.id = $1`, sectionID).Scan(&deptID)
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return CanManageDepartment(s, deptID), nil
	}
	return false, nil
}

func CanManageTAs(ctx context.Context, tx *sql.Tx, s Scope, sectionID string) (bool, error) {
	return manageOr(ctx, tx, s, sectionID, `select exists(select 1 from section_staff where section_id = $1 and user_id = $2 and role = 'instructor')`)
}

func CanReadRoster(ctx context.Context, tx *sql.Tx, s Scope, sectionID string) (bool, error) {
	return manageOr(ctx, tx, s, sectionID, `select exists(select 1 from section_staff where section_id = $1 and user_id = $2)`)
}

func CanReadSection(ctx context.Context, tx *sql.Tx, s Scope, sectionID string) (bool, error) {
	return manageOr(ctx, tx, s, sectionID, `select exists(select 1 from section_staff where section_id = $1 and user_id = $2) or exists(select 1 from enrollments where section_id = $1 and user_id = $2)`)
}

func manageOr(ctx context.Context, tx *sql.Tx, s Scope, sectionID, q string) (bool, error) {
	if ok, err := CanManageSection(ctx, tx, s, sectionID); ok || err != nil {
		return ok, err
	}
	var ok bool
	err := tx.QueryRowContext(ctx, q, sectionID, s.UserID).Scan(&ok)
	return ok, err
}
