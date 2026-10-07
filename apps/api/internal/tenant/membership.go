package tenant

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/audit"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
)

// Membership is the contract resource; createdAt only feeds the list cursor.
type Membership struct {
	ID            string     `json:"id"`
	UserID        string     `json:"userId"`
	Email         string     `json:"email"`
	DisplayName   string     `json:"displayName"`
	Role          authz.Role `json:"role"`
	Status        string     `json:"status"`
	DepartmentIDs []string   `json:"departmentIds"`
	createdAt     time.Time
}

const membershipSelect = `select m.id, m.user_id, u.email, u.display_name, m.role, m.status, m.created_at,
	coalesce((select json_agg(da.department_id order by da.department_id)
	          from department_admins da where da.user_id = m.user_id), '[]')
	from memberships m join users u on u.id = m.user_id `

type rowScanner interface{ Scan(...any) error }

func scanMembership(row rowScanner) (Membership, error) {
	var m Membership
	var ids []byte
	if err := row.Scan(&m.ID, &m.UserID, &m.Email, &m.DisplayName, &m.Role, &m.Status, &m.createdAt, &ids); err != nil {
		return m, err
	}
	var err error
	m.DepartmentIDs, err = ScanIDs(ids)
	return m, err
}

func getMembership(ctx context.Context, tx *sql.Tx, id string, forUpdate bool) (Membership, error) {
	q := membershipSelect + `where m.id = $1`
	if forUpdate {
		q += ` for update of m`
	}
	m, err := scanMembership(tx.QueryRowContext(ctx, q, id))
	if errors.Is(err, sql.ErrNoRows) {
		return m, httpx.NotFound("membership")
	}
	return m, err
}

// roleDepts validates role and departmentIds (shared by membership updates and invitations) and
// returns the deduplicated ids. An empty role means "missing".
func roleDepts(role string, ids []string) ([]string, []httpx.FieldError) {
	var d []httpx.FieldError
	r := authz.Role(role)
	switch r {
	case authz.Admin, authz.DeptAdmin, authz.Member:
	case "":
		d = append(d, httpx.FieldError{Field: "role", Message: "is required"})
	default:
		d = append(d, httpx.FieldError{Field: "role", Message: "must be one of admin, dept_admin, member"})
	}
	out := []string{}
	for _, id := range ids {
		if !IsUUID(id) {
			return out, append(d, httpx.FieldError{Field: "departmentIds", Message: "must contain only ids"})
		}
		if !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	if r == authz.DeptAdmin && len(out) == 0 {
		d = append(d, httpx.FieldError{Field: "departmentIds", Message: "is required for dept_admin"})
	} else if (r == authz.Admin || r == authz.Member) && len(out) > 0 {
		d = append(d, httpx.FieldError{Field: "departmentIds", Message: "must be empty unless role is dept_admin"})
	}
	return out, d
}

// checkDepts is 404 "department" unless every id is a department of this university (RLS scopes the count).
func checkDepts(ctx context.Context, tx *sql.Tx, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	var n int
	if err := tx.QueryRowContext(ctx, `select count(*) from departments where id = any($1::uuid[])`, ids).Scan(&n); err != nil {
		return err
	}
	if n != len(ids) {
		return httpx.NotFound("department")
	}
	return nil
}

// lockMembers serialises membership changes per university; it must run before the last-admin count.
func lockMembers(ctx context.Context, tx *sql.Tx, uni string) error {
	_, err := tx.ExecContext(ctx, `select 1 from universities where id = $1 for no key update`, uni)
	return err
}

// guardLastAdmin is 409 when m is the only active admin.
func guardLastAdmin(ctx context.Context, tx *sql.Tx, m Membership) error {
	if m.Role != authz.Admin || m.Status != "active" {
		return nil
	}
	var n int
	if err := tx.QueryRowContext(ctx, `select count(*) from memberships where role = 'admin' and status = 'active'`).Scan(&n); err != nil {
		return err
	}
	if n <= 1 {
		return httpx.Conflict("university must keep one admin")
	}
	return nil
}

func listMemberships(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	if !authz.IsAdmin(s) {
		return 0, nil, httpx.Forbidden()
	}
	p, after, afterID, err := Page(r)
	if err != nil {
		return 0, nil, err
	}
	rows, err := tx.QueryContext(r.Context(), membershipSelect+`where ($1::timestamptz is null or (m.created_at, m.id) > ($1, $2::uuid))
		order by m.created_at, m.id limit $3`, after, afterID, p.Limit+1)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	out := httpx.List[Membership]{Items: []Membership{}}
	var last Membership
	for rows.Next() {
		m, err := scanMembership(rows)
		if err != nil {
			return 0, nil, err
		}
		if len(out.Items) == p.Limit {
			out.NextCursor = httpx.NextCursor(last.createdAt.UTC(), last.ID)
			break
		}
		out.Items = append(out.Items, m)
		last = m
	}
	if err := rows.Err(); err != nil {
		return 0, nil, err
	}
	return http.StatusOK, out, nil
}

func updateMembership(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	if !authz.IsAdmin(s) {
		return 0, nil, httpx.Forbidden()
	}
	ctx := r.Context()
	id := r.PathValue("id")
	if !IsUUID(id) {
		return 0, nil, httpx.NotFound("membership")
	}
	var in struct {
		Role          string   `json:"role"`
		DepartmentIDs []string `json:"departmentIds"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	ids, details := roleDepts(in.Role, in.DepartmentIDs)
	if len(details) > 0 {
		return 0, nil, httpx.Validation(details...)
	}
	if err := lockMembers(ctx, tx, s.UniversityID); err != nil {
		return 0, nil, err
	}
	before, err := getMembership(ctx, tx, id, true)
	if err != nil {
		return 0, nil, err
	}
	if err := checkDepts(ctx, tx, ids); err != nil {
		return 0, nil, err
	}
	if authz.Role(in.Role) != authz.Admin {
		if err := guardLastAdmin(ctx, tx, before); err != nil {
			return 0, nil, err
		}
	}
	if _, err := tx.ExecContext(ctx, `update memberships set role = $2 where id = $1`, id, in.Role); err != nil {
		return 0, nil, err
	}
	if _, err := tx.ExecContext(ctx, `delete from department_admins where user_id = $1 and not (department_id = any($2::uuid[]))`, before.UserID, ids); err != nil {
		return 0, nil, err
	}
	if _, err := tx.ExecContext(ctx, `insert into department_admins (university_id, department_id, user_id)
		select $1, d, $2 from unnest($3::uuid[]) d on conflict do nothing`, s.UniversityID, before.UserID, ids); err != nil {
		return 0, nil, err
	}
	after, err := getMembership(ctx, tx, id, false)
	if err != nil {
		return 0, nil, err
	}
	if err := audit.Write(ctx, tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "membership.update", EntityType: "membership", EntityID: id, Before: before, After: after}); err != nil {
		return 0, nil, err
	}
	return http.StatusOK, after, nil
}

func deleteMembership(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	if !authz.IsAdmin(s) {
		return 0, nil, httpx.Forbidden()
	}
	ctx := r.Context()
	id := r.PathValue("id")
	if !IsUUID(id) {
		return 0, nil, httpx.NotFound("membership")
	}
	if err := lockMembers(ctx, tx, s.UniversityID); err != nil {
		return 0, nil, err
	}
	before, err := getMembership(ctx, tx, id, true)
	if err != nil {
		return 0, nil, err
	}
	if err := guardLastAdmin(ctx, tx, before); err != nil {
		return 0, nil, err
	}
	if _, err := tx.ExecContext(ctx, `delete from department_admins where user_id = $1`, before.UserID); err != nil {
		return 0, nil, err
	}
	if _, err := tx.ExecContext(ctx, `delete from memberships where id = $1`, id); err != nil {
		return 0, nil, err
	}
	if err := audit.Write(ctx, tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "membership.delete", EntityType: "membership", EntityID: id, Before: before}); err != nil {
		return 0, nil, err
	}
	return http.StatusNoContent, nil, nil
}
