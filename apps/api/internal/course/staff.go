package course

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/audit"
	"github.com/kapildev333/unicourse/apps/api/internal/auth"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/outbox"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

// StaffInvited is the 202 body of POST .../staff for an email that is not yet a member.
type StaffInvited struct {
	Invited    bool              `json:"invited"`
	Invitation tenant.Invitation `json:"invitation"`
}

type staffAudit struct {
	SectionID string `json:"sectionId"`
	UserID    string `json:"userId"`
	Email     string `json:"email"`
	Role      string `json:"role"`
}

// emailState is what resolveEmails knows about a normalised email in the caller's university.
// Status is "" (no membership), "active" or "invited"; InvitationID is the newest open invitation.
type emailState struct{ UserID, DisplayName, Status, InvitationID string }

// resolveEmails looks up many already-normalised emails in one query (RLS confines memberships
// and invitations to the tx's university; users is global). Reused by the import lanes.
func resolveEmails(ctx context.Context, tx *sql.Tx, emails []string) (map[string]emailState, error) {
	rows, err := tx.QueryContext(ctx, `select e.email, coalesce(u.id::text, ''), coalesce(u.display_name, ''), coalesce(m.status, ''),
		coalesce((select i.id::text from invitations i where i.email = e.email and i.accepted_at is null
			and i.expires_at > now() order by i.created_at desc limit 1), '')
		from unnest($1::text[]) as e(email)
		left join users u on u.email = e.email
		left join memberships m on m.user_id = u.id`, emails)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]emailState, len(emails))
	for rows.Next() {
		var email string
		var st emailState
		if err := rows.Scan(&email, &st.UserID, &st.DisplayName, &st.Status, &st.InvitationID); err != nil {
			return nil, err
		}
		out[email] = st
	}
	return out, rows.Err()
}

func sectionExists(r *http.Request, tx *sql.Tx, id string) error {
	if !tenant.IsUUID(id) {
		return httpx.NotFound("section")
	}
	var one int
	err := tx.QueryRowContext(r.Context(), `select 1 from sections where id = $1`, id).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return httpx.NotFound("section")
	}
	return err
}

// allow turns a predicate result into the 403 of the check order.
func allow(ok bool, err error) error {
	if err != nil {
		return err
	}
	if !ok {
		return httpx.Forbidden()
	}
	return nil
}

func listStaff(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	id := r.PathValue("id")
	if err := sectionExists(r, tx, id); err != nil {
		return 0, nil, err
	}
	// design §14.6: any admin or dept_admin of the university reads; members via CanReadSection
	if s.Role == authz.Member {
		if err := allow(authz.CanReadSection(r.Context(), tx, s, id)); err != nil {
			return 0, nil, err
		}
	}
	p, after, afterID, err := tenant.Page(r)
	if err != nil {
		return 0, nil, err
	}
	rows, err := tx.QueryContext(r.Context(), `select u.id, u.email, u.display_name, ss.role, ss.created_at, ss.id
		from section_staff ss join users u on u.id = ss.user_id
		where ss.section_id = $4 and ($1::timestamptz is null or (ss.created_at, ss.id) > ($1, $2::uuid))
		order by ss.created_at, ss.id limit $3`, after, afterID, p.Limit+1, id)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	out := httpx.List[Staff]{Items: []Staff{}}
	var lastAt time.Time
	var lastID string
	for rows.Next() {
		var st Staff
		var at time.Time
		var rowID string
		if err := rows.Scan(&st.UserID, &st.Email, &st.DisplayName, &st.Role, &at, &rowID); err != nil {
			return 0, nil, err
		}
		if len(out.Items) == p.Limit {
			out.NextCursor = httpx.NextCursor(lastAt.UTC(), lastID)
			break
		}
		out.Items = append(out.Items, st)
		lastAt, lastID = at, rowID
	}
	if err := rows.Err(); err != nil {
		return 0, nil, err
	}
	return http.StatusOK, out, nil
}

func createStaff(d tenant.Deps) tenant.Handler {
	return func(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
		ctx, secID := r.Context(), r.PathValue("id")
		if err := sectionExists(r, tx, secID); err != nil {
			return 0, nil, err
		}
		if err := allow(authz.CanManageTAs(ctx, tx, s, secID)); err != nil {
			return 0, nil, err
		}
		var in struct {
			Email string `json:"email"`
			Role  string `json:"role"`
		}
		if err := httpx.DecodeJSON(r, &in); err != nil {
			return 0, nil, err
		}
		var details []httpx.FieldError
		email, err := auth.NormalizeEmail(in.Email)
		if err != nil {
			details = append(details, httpx.FieldError{Field: "email", Message: "must be a valid email"})
		}
		if in.Role != "instructor" && in.Role != "ta" {
			details = append(details, httpx.FieldError{Field: "role", Message: "must be one of instructor, ta"})
		}
		if len(details) > 0 {
			return 0, nil, httpx.Validation(details...)
		}
		if in.Role == "instructor" {
			if err := allow(authz.CanManageSection(ctx, tx, s, secID)); err != nil {
				return 0, nil, err
			}
		}
		// serialise concurrent adds of one email so they cannot each create an invitation;
		// released at commit/rollback (a hash collision only serialises two unrelated adds)
		if _, err := tx.ExecContext(ctx, `select pg_advisory_xact_lock(hashtext($1))`, email); err != nil {
			return 0, nil, err
		}
		states, err := resolveEmails(ctx, tx, []string{email})
		if err != nil {
			return 0, nil, err
		}
		st := states[email]
		switch {
		case st.Status == "active":
			a := staffAudit{SectionID: secID, UserID: st.UserID, Email: email, Role: in.Role}
			var rowID string
			if err := tx.QueryRowContext(ctx, `insert into section_staff (university_id, section_id, user_id, role) values ($1,$2,$3,$4) returning id`,
				s.UniversityID, secID, st.UserID, in.Role).Scan(&rowID); err != nil {
				if err = pg.MapError(err); errors.Is(err, pg.ErrConflict) {
					return 0, nil, httpx.Conflict("user is already staff on this section")
				}
				return 0, nil, err
			}
			if err := audit.Write(ctx, tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "staff.create", EntityType: "staff", EntityID: rowID, After: a}); err != nil {
				return 0, nil, err
			}
			return http.StatusCreated, Staff{UserID: st.UserID, Email: email, DisplayName: st.DisplayName, Role: in.Role}, nil
		case st.Status != "":
			return 0, nil, tenant.ErrAlreadyMember
		}
		invID := st.InvitationID
		if invID == "" {
			inv, err := tenant.CreateInvitation(ctx, tx, tenant.NewInvitation{UniversityID: s.UniversityID, ActorUserID: s.UserID, Email: email, Role: authz.Member}, outbox.Enqueue, d.BaseURL)
			if err != nil {
				return 0, nil, err
			}
			invID = inv.ID
		}
		if _, err := tx.ExecContext(ctx, `insert into pending_section_staff (university_id, invitation_id, section_id, role) values ($1,$2,$3,$4)`,
			s.UniversityID, invID, secID, in.Role); err != nil {
			if err = pg.MapError(err); errors.Is(err, pg.ErrConflict) {
				return 0, nil, httpx.Conflict("invitation already pending for this section")
			}
			return 0, nil, err
		}
		inv, err := readInvitation(ctx, tx, invID)
		return http.StatusAccepted, StaffInvited{Invited: true, Invitation: inv}, err
	}
}

func readInvitation(ctx context.Context, tx *sql.Tx, id string) (tenant.Invitation, error) {
	var v tenant.Invitation
	var ids []byte
	var acc sql.NullTime
	err := tx.QueryRowContext(ctx, `select id, email, role, to_json(department_ids), expires_at, accepted_at from invitations where id = $1`, id).
		Scan(&v.ID, &v.Email, &v.Role, &ids, &v.ExpiresAt, &acc)
	if err != nil {
		return v, err
	}
	v.ExpiresAt = v.ExpiresAt.UTC()
	if acc.Valid {
		t := acc.Time.UTC()
		v.AcceptedAt = &t
	}
	v.DepartmentIDs = []string{}
	if err := json.Unmarshal(ids, &v.DepartmentIDs); err != nil {
		return v, err
	}
	return v, nil
}

func deleteStaff(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	ctx, secID := r.Context(), r.PathValue("id")
	if err := sectionExists(r, tx, secID); err != nil {
		return 0, nil, err
	}
	if err := allow(authz.CanManageTAs(ctx, tx, s, secID)); err != nil {
		return 0, nil, err
	}
	userID := r.PathValue("userId")
	if !tenant.IsUUID(userID) {
		return 0, nil, httpx.NotFound("staff")
	}
	a := staffAudit{SectionID: secID, UserID: userID}
	var rowID string
	err := tx.QueryRowContext(ctx, `select ss.id, ss.role, u.email from section_staff ss join users u on u.id = ss.user_id
		where ss.section_id = $1 and ss.user_id = $2 for update of ss`, secID, userID).Scan(&rowID, &a.Role, &a.Email)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil, httpx.NotFound("staff")
	} else if err != nil {
		return 0, nil, err
	}
	if a.Role == "instructor" {
		if err := allow(authz.CanManageSection(ctx, tx, s, secID)); err != nil {
			return 0, nil, err
		}
	}
	if _, err := tx.ExecContext(ctx, `delete from section_staff where id = $1`, rowID); err != nil {
		return 0, nil, err
	}
	if err := audit.Write(ctx, tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "staff.delete", EntityType: "staff", EntityID: rowID, Before: a}); err != nil {
		return 0, nil, err
	}
	return http.StatusNoContent, nil, nil
}

// deletePendingStaff cancels one pending assignment; the invitation itself stays (it may carry
// other sections or a department-admin role; DELETE /invitations/{id} revokes it).
func deletePendingStaff(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	ctx, secID, invID := r.Context(), r.PathValue("id"), r.PathValue("invitationId")
	if err := sectionExists(r, tx, secID); err != nil {
		return 0, nil, err
	}
	if err := allow(authz.CanManageTAs(ctx, tx, s, secID)); err != nil {
		return 0, nil, err
	}
	if !tenant.IsUUID(invID) {
		return 0, nil, httpx.NotFound("pending staff")
	}
	var before struct {
		InvitationID string `json:"invitationId"`
		Email        string `json:"email"`
		Role         string `json:"role"`
	}
	before.InvitationID = invID
	err := tx.QueryRowContext(ctx, `select i.email, p.role from pending_section_staff p join invitations i on i.id = p.invitation_id
		where p.section_id = $1 and p.invitation_id = $2 for update of p`, secID, invID).Scan(&before.Email, &before.Role)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil, httpx.NotFound("pending staff")
	} else if err != nil {
		return 0, nil, err
	}
	if before.Role == "instructor" {
		if err := allow(authz.CanManageSection(ctx, tx, s, secID)); err != nil {
			return 0, nil, err
		}
	}
	if _, err := tx.ExecContext(ctx, `delete from pending_section_staff where section_id = $1 and invitation_id = $2`, secID, invID); err != nil {
		return 0, nil, err
	}
	if err := audit.Write(ctx, tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "pending_staff.delete", EntityType: "section", EntityID: secID, Before: before}); err != nil {
		return 0, nil, err
	}
	return http.StatusNoContent, nil, nil
}
