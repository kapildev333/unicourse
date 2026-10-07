package auth

import (
	"context"
	"database/sql"
	"net/url"
	"strings"

	"github.com/kapildev333/unicourse/apps/api/internal/audit"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
)

var ErrNotAllowed = &httpx.Error{Status: 403, Code: "forbidden", Message: "your email is not allowed for this university"}

// allowed reports whether the email's domain belongs to any university or a pending, unexpired
// invitation exists for it. email must be normalised; tx must be a pg.WithUserTx with app.email = email
// so the invitations_by_email policy applies.
func allowed(ctx context.Context, tx *sql.Tx, email string) (ok bool, err error) {
	domain := email[strings.LastIndex(email, "@")+1:]
	err = tx.QueryRowContext(ctx, `select exists (select 1 from universities where $1 = any(allowed_email_domains))
		or exists (select 1 from invitations where email = $2 and accepted_at is null and expires_at > now())`, domain, email).Scan(&ok)
	return ok, err
}

// InvitationURL is the link mailed to an invitee: {baseURL}/invite/{token}?email={email}.
func InvitationURL(baseURL, token, email string) string {
	return baseURL + "/invite/" + token + "?email=" + url.QueryEscape(email)
}

// SignIn finds or creates the user for a verified email and accepts every pending invitation addressed
// to it, in any university, in one transaction. displayName (then the oldest open invitation's stored name) only fills an empty name.
func SignIn(ctx context.Context, db *sql.DB, email, displayName string) (userID string, err error) {
	if email, err = NormalizeEmail(email); err != nil {
		return "", err
	}
	err = pg.WithUserTx(ctx, db, "", email, func(tx *sql.Tx) error {
		ok, err := allowed(ctx, tx, email)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotAllowed
		}
		if err := tx.QueryRowContext(ctx, `insert into users (email, display_name) values ($1, $2)
			on conflict (email) do update
			  set display_name = case when users.display_name = '' then excluded.display_name else users.display_name end
			returning id`, email, displayName).Scan(&userID); err != nil {
			return pg.MapError(err)
		}
		// Read every invitation before the first SetUniversity: invitations_by_email switches off once a
		// university is set (design section 16.2).
		type inv struct{ id, uni, name string }
		var invs []inv
		rows, err := tx.QueryContext(ctx, `select id, university_id, display_name from invitations
			where email = $1 and accepted_at is null and expires_at > now() order by created_at, id`, email)
		if err != nil {
			return pg.MapError(err)
		}
		for rows.Next() {
			var i inv
			if err := rows.Scan(&i.id, &i.uni, &i.name); err != nil {
				rows.Close()
				return err
			}
			invs = append(invs, i)
		}
		if err := rows.Close(); err != nil {
			return err
		}
		if err := rows.Err(); err != nil {
			return err
		}
		for _, i := range invs { // the oldest open invitation with a name, only into an empty name
			if i.name != "" {
				if _, err := tx.ExecContext(ctx, `update users set display_name = $2 where id = $1 and display_name = ''`, userID, i.name); err != nil {
					return pg.MapError(err)
				}
				break
			}
		}
		for _, i := range invs {
			if err := pg.SetUniversity(ctx, tx, i.uni); err != nil {
				return err
			}
			var role, depts string
			err := tx.QueryRowContext(ctx, `update invitations set accepted_at = now()
				where id = $1 and accepted_at is null and expires_at > now()
				returning role, array_to_string(department_ids, ',')`, i.id).Scan(&role, &depts)
			if err == sql.ErrNoRows {
				continue // a concurrent sign-in won
			}
			if err != nil {
				return pg.MapError(err)
			}
			deptIDs := []string{}
			if depts != "" {
				deptIDs = strings.Split(depts, ",")
			}
			acc := []audit.Entry{{ActorUserID: userID, Action: "invitation.accept", EntityType: "invitation", EntityID: i.id,
				After: map[string]any{"email": email, "role": role, "departmentIds": deptIDs}}}

			// Membership before the change. A concurrent first sign-in can leave this empty (statement snapshot),
			// so the before of a membership.update may then be missing.
			var oldID, oldRole, oldStatus string
			var oldDepts []string
			oldFound := true
			err = tx.QueryRowContext(ctx, `select id, role, status from memberships where university_id = $1 and user_id = $2`, i.uni, userID).
				Scan(&oldID, &oldRole, &oldStatus)
			if err == sql.ErrNoRows {
				oldFound = false
			} else if err != nil {
				return pg.MapError(err)
			}
			if oldFound {
				if oldDepts, err = userDepts(ctx, tx, userID); err != nil {
					return err
				}
			}

			// A no-op conflict (equal or higher role, already active) updates nothing and returns no row.
			var mID, mRole, mStatus string
			var inserted bool
			err = tx.QueryRowContext(ctx, `insert into memberships (university_id, user_id, role, status) values ($1, $2, $3, 'active')
				  on conflict (university_id, user_id) do update set status = 'active',
				    role = case when array_position(array['member','dept_admin','admin'], excluded.role)
				                   > array_position(array['member','dept_admin','admin'], memberships.role)
				           then excluded.role else memberships.role end
				  where memberships.status <> 'active'
				     or array_position(array['member','dept_admin','admin'], excluded.role)
				      > array_position(array['member','dept_admin','admin'], memberships.role)
				  returning id, role, status, xmax = 0`, i.uni, userID, role).Scan(&mID, &mRole, &mStatus, &inserted)
			hasM := err == nil
			if err != nil && err != sql.ErrNoRows {
				return pg.MapError(err)
			}
			if !hasM {
				mID, mRole, mStatus = oldID, oldRole, oldStatus
			}

			deptsAdded := false
			if role == "dept_admin" {
				// Only ids that exist in this university (RLS scopes departments): a stale or foreign id is
				// skipped instead of failing the whole sign-in with a foreign-key error.
				res, err := tx.ExecContext(ctx, `insert into department_admins (university_id, department_id, user_id)
				 select $1, d.id, $2 from departments d where d.id = any(string_to_array($3::text, ',')::uuid[])
				 on conflict do nothing`, i.uni, userID, depts)
				if err != nil {
					return pg.MapError(err)
				}
				n, _ := res.RowsAffected()
				deptsAdded = n > 0
			}
			// An untouched membership whose department set grew is still a change.
			if hasM || (oldFound && deptsAdded) {
				after, err := userDepts(ctx, tx, userID)
				if err != nil {
					return err
				}
				e := audit.Entry{ActorUserID: userID, Action: "membership.create", EntityType: "membership", EntityID: mID,
					After: map[string]any{"role": mRole, "status": mStatus, "departmentIds": after}}
				if oldFound {
					e.Action = "membership.update"
					e.Before = map[string]any{"role": oldRole, "status": oldStatus, "departmentIds": oldDepts}
				}
				acc = append(acc, e)
			}

			rs, err := tx.QueryContext(ctx, `insert into section_staff (university_id, section_id, user_id, role)
				 select university_id, section_id, $1, role from pending_section_staff where invitation_id = $2
				 on conflict do nothing returning id, section_id, role`, userID, i.id)
			if err != nil {
				return pg.MapError(err)
			}
			for rs.Next() {
				var id, sec, r string
				if err := rs.Scan(&id, &sec, &r); err != nil {
					rs.Close()
					return err
				}
				acc = append(acc, audit.Entry{ActorUserID: userID, Action: "staff.create", EntityType: "staff", EntityID: id,
					After: staffAudit{SectionID: sec, UserID: userID, Email: email, Role: r}})
			}
			if err := rs.Close(); err != nil {
				return err
			}
			if err := rs.Err(); err != nil {
				return pg.MapError(err)
			}

			rs, err = tx.QueryContext(ctx, `insert into enrollments (university_id, section_id, user_id)
				 select university_id, section_id, $1 from pending_enrollments where invitation_id = $2
				 on conflict do nothing returning id, section_id`, userID, i.id)
			if err != nil {
				return pg.MapError(err)
			}
			for rs.Next() {
				var id, sec string
				if err := rs.Scan(&id, &sec); err != nil {
					rs.Close()
					return err
				}
				acc = append(acc, audit.Entry{ActorUserID: userID, Action: "enrollment.create", EntityType: "enrollment", EntityID: id,
					After: enrollmentAudit{SectionID: sec, UserID: userID, Email: email}})
			}
			if err := rs.Close(); err != nil {
				return err
			}
			if err := rs.Err(); err != nil {
				return pg.MapError(err)
			}
			if err := audit.WriteMany(ctx, tx, i.uni, acc); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return userID, nil
}

// Same JSON shapes as internal/course's staffAudit and enrollmentAudit (auth cannot import course).
type staffAudit struct {
	SectionID string `json:"sectionId"`
	UserID    string `json:"userId"`
	Email     string `json:"email"`
	Role      string `json:"role"`
}

type enrollmentAudit struct {
	SectionID string `json:"sectionId"`
	UserID    string `json:"userId"`
	Email     string `json:"email"`
}

// userDepts is the user's full department set in the current university (RLS-scoped), sorted; never nil.
func userDepts(ctx context.Context, tx *sql.Tx, userID string) ([]string, error) {
	rs, err := tx.QueryContext(ctx, `select department_id from department_admins where user_id = $1 order by department_id`, userID)
	if err != nil {
		return nil, pg.MapError(err)
	}
	defer rs.Close()
	out := []string{}
	for rs.Next() {
		var d string
		if err := rs.Scan(&d); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rs.Err()
}
