package auth

import (
	"context"
	"database/sql"
	"strings"

	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
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
