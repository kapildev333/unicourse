// Package tenant is the university-scoped request pipeline and its handlers.
package tenant

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/kapildev333/unicourse/apps/api/internal/auth"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/blob"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
)

type Deps struct {
	DB       *sql.DB
	Sessions *auth.SessionStore
	BaseURL  string   // cfg.BaseURL, used for invitation URLs
	Files    blob.Dir // UNICOURSE_FILES_DIR; zero in tests that never touch files
}

// Handler is a tenant-scoped handler. tx is the request's only transaction (RLS = s.UniversityID).
// Return (status, body, nil) on success; body nil means no body. Return an error to roll back.
// The status must be non-zero: a zero status commits and then fails the response with a 500.
type Handler func(r *http.Request, tx *sql.Tx, s authz.Scope) (status int, body any, err error)

// Raw is a handler body written verbatim instead of JSON (the gradebook CSV).
type Raw struct {
	ContentType, Filename string
	Body                  []byte
}

// After is a handler body that Wrap runs once the transaction has committed, instead of writing JSON.
// It writes the whole response (status included). An error it returns goes to the error envelope, so it
// must return before writing anything when it fails. Its request carries the scope (authz.ScopeFromContext).
// Used for file downloads, uploads (streamed outside the request transaction) and blob removal after deletes.
type After func(w http.ResponseWriter, r *http.Request) error

// UniversityID is the request's university, "" outside Wrap.
func UniversityID(ctx context.Context) string {
	s, _ := authz.ScopeFromContext(ctx)
	return s.UniversityID
}

// Wrap runs session -> slug -> membership -> tenant tx -> h; the response is written after commit.
func Wrap(d Deps, h Handler) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		ctx := r.Context()
		a, err := auth.Authenticate(w, r, d.Sessions)
		if err != nil {
			return err
		}
		var uniID string
		err = pg.WithPlatformTx(ctx, d.DB, func(tx *sql.Tx) error {
			return tx.QueryRowContext(ctx, `select id from universities where slug = $1`, r.PathValue("slug")).Scan(&uniID)
		})
		if errors.Is(err, sql.ErrNoRows) {
			return httpx.NotFound("university")
		}
		if err != nil {
			return err
		}
		httpx.WithUniversityID(ctx, uniID)

		var (
			status int
			body   any
			sr     *http.Request
		)
		err = pg.WithTx(ctx, d.DB, uniID, func(tx *sql.Tx) error {
			// RLS limits memberships to uniID, so the same user's membership elsewhere is invisible.
			var role string
			var ids []byte
			err := tx.QueryRowContext(ctx, `select m.role,
				case when m.role = 'dept_admin' then coalesce((select json_agg(da.department_id order by da.department_id) from department_admins da where da.user_id = m.user_id), '[]') else '[]' end
				from memberships m where m.user_id = $1 and m.status = 'active'`, a.UserID).Scan(&role, &ids)
			if errors.Is(err, sql.ErrNoRows) {
				return httpx.Forbidden()
			}
			if err != nil {
				return err
			}
			deptIDs, err := ScanIDs(ids)
			if err != nil {
				return err
			}
			s := authz.Scope{UniversityID: uniID, UserID: a.UserID, Role: authz.Role(role), DeptIDs: deptIDs}
			sr = r.WithContext(authz.WithScope(ctx, s))
			status, body, err = h(sr, tx, s)
			return err
		})
		if err != nil {
			return err
		}
		if f, ok := body.(After); ok {
			return f(w, sr)
		}
		if body == nil {
			w.WriteHeader(status)
			return nil
		}
		if raw, ok := body.(Raw); ok {
			w.Header().Set("Content-Type", raw.ContentType)
			if raw.Filename != "" {
				w.Header().Set("Content-Disposition", `attachment; filename="`+raw.Filename+`"`)
			}
			w.WriteHeader(status)
			_, _ = w.Write(raw.Body) // headers are sent; a write error is a client disconnect, not a 500
			return nil
		}
		httpx.WriteJSON(w, status, body)
		return nil
	}
}
