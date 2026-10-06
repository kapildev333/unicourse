package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"html/template"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
)

// Deps is everything the auth routes need. Handlers dereference it at request time only, so
// Register works with the zero value (the contract test builds the router that way).
type Deps struct {
	DB        *sql.DB
	Sessions  *SessionStore
	Magic     *MagicLinks
	Providers map[string]Provider
	Secret    []byte
	WebBase   string // cfg.WebOrigin if set, else cfg.BaseURL; browser redirects land here
	// EmailLimit and IPLimit guard POST /auth/magic-link (5 per email, 300 per IP, per hour).
	EmailLimit, IPLimit *Limiter
}

// Register adds the seven auth routes.
func Register(r *httpx.Router, d Deps) {
	r.Handle("POST", "/api/v1/auth/oidc/{provider}/start", d.oidcStart)
	r.Handle("GET", "/api/v1/auth/oidc/{provider}/callback", d.oidcCallbackHandler)
	r.Handle("POST", "/api/v1/auth/magic-link", d.requestMagicLink)
	r.Handle("GET", "/api/v1/auth/magic-link/{token}", d.confirmMagicLink)
	r.Handle("POST", "/api/v1/auth/magic-link/{token}", d.consumeMagicLink)
	r.Handle("POST", "/api/v1/auth/sign-out", d.signOut)
	r.Handle("GET", "/api/v1/me", d.me)
}

// safeNext returns s when it is a same-site relative path, else "".
func safeNext(s string) string {
	if len(s) > 512 || !strings.HasPrefix(s, "/") || strings.HasPrefix(s, "//") || strings.HasPrefix(s, `/\`) ||
		strings.ContainsAny(s, "\r\n\t") {
		return ""
	}
	return s
}

func (d Deps) web(path string) string { return strings.TrimRight(d.WebBase, "/") + path }

// signInError redirects to the sign-in page with the error code as the reason.
func (d Deps) signInError(w http.ResponseWriter, r *http.Request, code string) {
	http.Redirect(w, r, d.web("/sign-in?error="+url.QueryEscape(code)), http.StatusFound)
}

// failRedirect maps a sign-in failure to a redirect reason; anything unexpected is returned (500).
func (d Deps) failRedirect(w http.ResponseWriter, r *http.Request, err error) error {
	var he *httpx.Error
	switch {
	case errors.Is(err, errOIDC):
		d.signInError(w, r, "unauthenticated")
	case errors.Is(err, ErrNotAllowed):
		d.signInError(w, r, "forbidden")
	case errors.As(err, &he) && (he.Code == "validation_failed" || he.Code == "rate_limited"):
		slog.InfoContext(r.Context(), "sign-in rejected", "code", he.Code, "reason", he.Message)
		d.signInError(w, r, he.Code)
	default:
		// Explicit 500: a stray pg.ErrNotFound/ErrConflict must not surface as 404/409 on a redirect flow.
		slog.ErrorContext(r.Context(), "sign-in failed", "err", err)
		return &httpx.Error{Status: 500, Code: "internal", Message: "internal error"}
	}
	return nil
}

// landing creates the session, sets the cookie and redirects to next (if safe and the user has a
// membership), else the slug's or first membership's home, else sign-in?error=no_membership.
func (d Deps) landing(w http.ResponseWriter, r *http.Request, userID, slug, next string) error {
	next = safeNext(next) // second line of defence for values recovered from the OIDC state
	tok, err := d.Sessions.Create(r.Context(), userID)
	if err != nil {
		return err
	}
	setSessionCookie(w, tok)
	ms, err := d.memberships(r.Context(), userID)
	if err != nil {
		return err
	}
	dest := "/sign-in?error=no_membership"
	if len(ms) > 0 {
		dest = "/" + ms[0].University.Slug + "/me"
		for _, m := range ms {
			if m.University.Slug == slug {
				dest = "/" + slug + "/me"
				break
			}
		}
		if next != "" {
			dest = next
		}
	}
	http.Redirect(w, r, d.web(dest), http.StatusFound)
	return nil
}

type university struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type membership struct {
	University    university `json:"university"`
	Role          string     `json:"role"`
	DepartmentIDs []string   `json:"departmentIds"`
}

// memberships returns the user's active memberships ordered by university name; never nil.
func (d Deps) memberships(ctx context.Context, userID string) ([]membership, error) {
	out := []membership{}
	err := pg.WithUserTx(ctx, d.DB, userID, "", func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `
select u.id, u.slug, u.name, m.role,
       to_json(array(select da.department_id::text from department_admins da
                      where da.user_id = m.user_id and da.university_id = m.university_id
                      order by da.department_id))
from memberships m join universities u on u.id = m.university_id
where m.user_id = $1 and m.status = 'active'
order by u.name`, userID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var m membership
			var ids []byte
			if err := rows.Scan(&m.University.ID, &m.University.Slug, &m.University.Name, &m.Role, &ids); err != nil {
				return err
			}
			if err := json.Unmarshal(ids, &m.DepartmentIDs); err != nil {
				return err
			}
			out = append(out, m)
		}
		return rows.Err()
	})
	return out, err
}

func (d Deps) me(w http.ResponseWriter, r *http.Request) error {
	a, err := Authenticate(w, r, d.Sessions)
	if err != nil {
		return err
	}
	var u struct {
		ID          string `json:"id"`
		Email       string `json:"email"`
		DisplayName string `json:"displayName"`
	}
	err = pg.WithUserTx(r.Context(), d.DB, a.UserID, "", func(tx *sql.Tx) error {
		return tx.QueryRowContext(r.Context(), `select id, email, display_name from users where id = $1`, a.UserID).
			Scan(&u.ID, &u.Email, &u.DisplayName)
	})
	if err != nil {
		return err
	}
	ms, err := d.memberships(r.Context(), a.UserID)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"user": u, "memberships": ms})
	return nil
}

func (d Deps) signOut(w http.ResponseWriter, r *http.Request) error {
	clearSessionCookie(w)
	if c, err := r.Cookie(SessionCookie); err == nil {
		if err := d.Sessions.Revoke(r.Context(), c.Value); err != nil {
			return err
		}
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (d Deps) oidcStart(w http.ResponseWriter, r *http.Request) error {
	p, ok := d.Providers[r.PathValue("provider")]
	if !ok {
		return httpx.NotFound("provider")
	}
	var body struct{ Slug, Email, Next string }
	if err := httpx.DecodeJSON(r, &body); err != nil {
		return err
	}
	if (body.Slug == "") == (body.Email == "") {
		return httpx.Validation(httpx.FieldError{Field: "slug", Message: "give exactly one of slug or email"})
	}
	email := ""
	if body.Email != "" {
		var err error
		if email, err = NormalizeEmail(body.Email); err != nil {
			return err
		}
	}
	authURL, cookie, err := p.begin(d.Secret, body.Slug, email, safeNext(body.Next), time.Now())
	if err != nil {
		return err
	}
	setOIDCCookie(w, cookie)
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"url": authURL})
	return nil
}

func (d Deps) oidcCallbackHandler(w http.ResponseWriter, r *http.Request) error {
	uid, st, err := d.oidcCallback(w, r)
	if err != nil {
		return d.failRedirect(w, r, err)
	}
	return d.landing(w, r, uid, st.Slug, st.Next)
}

func (d Deps) requestMagicLink(w http.ResponseWriter, r *http.Request) error {
	var body struct{ Email, Next string }
	if err := httpx.DecodeJSON(r, &body); err != nil {
		return err
	}
	email, err := NormalizeEmail(body.Email)
	if err != nil {
		return err
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if !d.IPLimit.Allow("ip:"+host) || !d.EmailLimit.Allow("email:"+email) {
		return httpx.RateLimited()
	}
	if err := d.Magic.Send(r.Context(), email, safeNext(body.Next)); err != nil {
		return err
	}
	w.WriteHeader(http.StatusAccepted)
	return nil
}

var confirmPage = template.Must(template.New("confirm").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Sign in to UniCourse</title></head>
<body style="font-family:system-ui,sans-serif;max-width:24rem;margin:4rem auto;padding:0 1rem">
<h1>Sign in to UniCourse</h1>
<form method="post" action="{{.Action}}"><button type="submit" style="font-size:1rem;padding:.6rem 1.2rem">Continue signing in</button></form>
</body></html>
`))

// confirmMagicLink never consumes the token: mail scanners and link previews issue GETs.
func (d Deps) confirmMagicLink(w http.ResponseWriter, r *http.Request) error {
	action := "/api/v1/auth/magic-link/" + url.PathEscape(r.PathValue("token"))
	if n := safeNext(r.URL.Query().Get("next")); n != "" {
		action += "?next=" + url.QueryEscape(n)
	}
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "no-store")
	h.Set("Referrer-Policy", "no-referrer")
	return confirmPage.Execute(w, map[string]string{"Action": action})
}

func (d Deps) consumeMagicLink(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	email, err := d.Magic.Consume(r.Context(), r.PathValue("token"))
	if err != nil {
		return d.failRedirect(w, r, err)
	}
	uid, err := SignIn(r.Context(), d.DB, email, "")
	if err != nil {
		return d.failRedirect(w, r, err)
	}
	return d.landing(w, r, uid, "", safeNext(r.URL.Query().Get("next")))
}
