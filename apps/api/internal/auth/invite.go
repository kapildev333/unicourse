package auth

import (
	"context"
	"database/sql"
	"net/url"
	"strings"

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
// to it, in any university, in one transaction. displayName only fills an empty name.
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
		type inv struct{ id, uni string }
		var invs []inv
		rows, err := tx.QueryContext(ctx, `select id, university_id from invitations
			where email = $1 and accepted_at is null and expires_at > now() order by created_at, id`, email)
		if err != nil {
			return pg.MapError(err)
		}
		for rows.Next() {
			var i inv
			if err := rows.Scan(&i.id, &i.uni); err != nil {
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
			type stmt struct {
				q    string
				args []any
			}
			stmts := []stmt{
				{`insert into memberships (university_id, user_id, role, status) values ($1, $2, $3, 'active')
				 on conflict (university_id, user_id) do update set status = 'active',
				   role = case when array_position(array['member','dept_admin','admin'], excluded.role)
				                  > array_position(array['member','dept_admin','admin'], memberships.role)
				          then excluded.role else memberships.role end`, []any{i.uni, userID, role}},
				{`insert into section_staff (university_id, section_id, user_id, role)
				 select university_id, section_id, $1, role from pending_section_staff where invitation_id = $2
				 on conflict do nothing`, []any{userID, i.id}},
				{`insert into enrollments (university_id, section_id, user_id)
				 select university_id, section_id, $1 from pending_enrollments where invitation_id = $2
				 on conflict do nothing`, []any{userID, i.id}},
			}
			if role == "dept_admin" {
				stmts = append(stmts, stmt{`insert into department_admins (university_id, department_id, user_id)
				 select $1, d::uuid, $2 from unnest(string_to_array($3::text, ',')) d on conflict do nothing`, []any{i.uni, userID, depts}})
			}
			for _, s := range stmts {
				if _, err := tx.ExecContext(ctx, s.q, s.args...); err != nil {
					return pg.MapError(err)
				}
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return userID, nil
}
