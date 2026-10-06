package tenant

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/audit"
	"github.com/kapildev333/unicourse/apps/api/internal/auth"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
)

// Invitation is the contract resource; createdAt only feeds the list cursor.
type Invitation struct {
	ID            string     `json:"id"`
	Email         string     `json:"email"`
	Role          authz.Role `json:"role"`
	DepartmentIDs []string   `json:"departmentIds"`
	ExpiresAt     time.Time  `json:"expiresAt"`
	AcceptedAt    *time.Time `json:"acceptedAt,omitempty"`
	createdAt     time.Time
}

// NewInvitation is the input of CreateInvitation. ActorUserID is "" for the CLI.
type NewInvitation struct {
	UniversityID, ActorUserID string
	Email                     string // raw; normalised inside
	Role                      authz.Role
	DepartmentIDs             []string
}

var (
	ErrAlreadyMember     = httpx.Conflict("user is already a member")
	ErrInvitationPending = httpx.Conflict("invitation already pending for this email")
)

const invitationSelect = `select id, email, role, to_json(department_ids), expires_at, accepted_at, created_at from invitations `

func scanInvitation(row rowScanner) (Invitation, error) {
	var v Invitation
	var ids []byte
	var acc sql.NullTime
	if err := row.Scan(&v.ID, &v.Email, &v.Role, &ids, &v.ExpiresAt, &acc, &v.createdAt); err != nil {
		return v, err
	}
	v.ExpiresAt = v.ExpiresAt.UTC()
	if acc.Valid {
		t := acc.Time.UTC()
		v.AcceptedAt = &t
	}
	var err error
	v.DepartmentIDs, err = ScanIDs(ids)
	return v, err
}

// CreateInvitation does no authorisation (callers check authz.CanInvite). tx must be a
// pg.WithTx(in.UniversityID) transaction. Mail is sent before commit.
func CreateInvitation(ctx context.Context, tx *sql.Tx, in NewInvitation, m auth.Mailer, baseURL string) (Invitation, error) {
	var details []httpx.FieldError
	email, err := auth.NormalizeEmail(in.Email)
	if err != nil {
		details = append(details, httpx.FieldError{Field: "email", Message: "must be a valid email"})
	}
	ids, rd := roleDepts(string(in.Role), in.DepartmentIDs)
	if details = append(details, rd...); len(details) > 0 {
		return Invitation{}, httpx.Validation(details...)
	}
	if err := checkDepts(ctx, tx, ids); err != nil {
		return Invitation{}, err
	}
	var exists bool
	if err := tx.QueryRowContext(ctx, `select exists(select 1 from memberships m join users u on u.id = m.user_id where u.email = $1)`, email).Scan(&exists); err != nil {
		return Invitation{}, err
	}
	if exists {
		return Invitation{}, ErrAlreadyMember
	}
	if err := tx.QueryRowContext(ctx, `select exists(select 1 from invitations where email = $1 and accepted_at is null and expires_at > now())`, email).Scan(&exists); err != nil {
		return Invitation{}, err
	}
	if exists {
		return Invitation{}, ErrInvitationPending
	}
	token, hash, err := auth.NewToken()
	if err != nil {
		return Invitation{}, err
	}
	inv, err := scanInvitation(tx.QueryRowContext(ctx, `insert into invitations (university_id, email, role, department_ids, token_hash, expires_at)
		values ($1, $2, $3, $4::uuid[], $5, now() + interval '14 days')
		returning id, email, role, to_json(department_ids), expires_at, accepted_at, created_at`,
		in.UniversityID, email, string(in.Role), ids, hash))
	if err != nil {
		return Invitation{}, err
	}
	if err := audit.Write(ctx, tx, in.UniversityID, audit.Entry{ActorUserID: in.ActorUserID, Action: "invitation.create", EntityType: "invitation", EntityID: inv.ID, After: inv}); err != nil {
		return Invitation{}, err
	}
	var name string
	if err := tx.QueryRowContext(ctx, `select name from universities where id = $1`, in.UniversityID).Scan(&name); err != nil {
		return Invitation{}, err
	}
	text := "You have been invited to join " + name + " on UniCourse as " + string(in.Role) + ".\n\nAccept the invitation:\n" +
		auth.InvitationURL(baseURL, token, email) + "\n\nThis invitation expires on " + inv.ExpiresAt.Format("2006-01-02") + ".\n"
	if err := m.Send(ctx, email, "You are invited to "+name+" on UniCourse", text); err != nil {
		return Invitation{}, err
	}
	return inv, nil
}

func listInvitations(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	if s.Role != authz.Admin && s.Role != authz.DeptAdmin {
		return 0, nil, httpx.Forbidden()
	}
	p, after, afterID, err := Page(r)
	if err != nil {
		return 0, nil, err
	}
	rows, err := tx.QueryContext(r.Context(), invitationSelect+`where ($4 or role = 'member')
		and ($1::timestamptz is null or (created_at, id) > ($1, $2::uuid))
		order by created_at, id limit $3`, after, afterID, p.Limit+1, authz.IsAdmin(s))
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	out := httpx.List[Invitation]{Items: []Invitation{}}
	var last Invitation
	for rows.Next() {
		v, err := scanInvitation(rows)
		if err != nil {
			return 0, nil, err
		}
		if len(out.Items) == p.Limit {
			out.NextCursor = httpx.NextCursor(last.createdAt.UTC(), last.ID)
			break
		}
		out.Items = append(out.Items, v)
		last = v
	}
	if err := rows.Err(); err != nil {
		return 0, nil, err
	}
	return http.StatusOK, out, nil
}

func createInvitation(d Deps) Handler {
	return func(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
		if s.Role != authz.Admin && s.Role != authz.DeptAdmin {
			return 0, nil, httpx.Forbidden()
		}
		var in struct {
			Email         string
			Role          string
			DepartmentIDs []string
		}
		if err := httpx.DecodeJSON(r, &in); err != nil {
			return 0, nil, err
		}
		switch role := authz.Role(in.Role); role {
		case authz.Admin, authz.DeptAdmin, authz.Member:
			if !authz.CanInvite(s, role) {
				return 0, nil, httpx.Forbidden()
			}
		}
		inv, err := CreateInvitation(r.Context(), tx, NewInvitation{UniversityID: s.UniversityID, ActorUserID: s.UserID,
			Email: in.Email, Role: authz.Role(in.Role), DepartmentIDs: in.DepartmentIDs}, d.Mailer, d.BaseURL)
		if err != nil {
			return 0, nil, err
		}
		return http.StatusCreated, inv, nil
	}
}

func deleteInvitation(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	if s.Role != authz.Admin && s.Role != authz.DeptAdmin {
		return 0, nil, httpx.Forbidden()
	}
	id := r.PathValue("id")
	if !IsUUID(id) {
		return 0, nil, httpx.NotFound("invitation")
	}
	before, err := scanInvitation(tx.QueryRowContext(r.Context(), invitationSelect+`where id = $1 for update`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil, httpx.NotFound("invitation")
	} else if err != nil {
		return 0, nil, err
	}
	if !authz.CanInvite(s, before.Role) {
		return 0, nil, httpx.Forbidden()
	}
	if before.AcceptedAt != nil {
		return 0, nil, httpx.Conflict("invitation already accepted")
	}
	if _, err := tx.ExecContext(r.Context(), `delete from invitations where id = $1`, id); err != nil {
		return 0, nil, err
	}
	if err := audit.Write(r.Context(), tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "invitation.delete", EntityType: "invitation", EntityID: id, Before: before}); err != nil {
		return 0, nil, err
	}
	return http.StatusNoContent, nil, nil
}
