package auth

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
)

const (
	SessionCookie = "unicourse_session"
	sessionMaxAge = 30 * 24 * 60 * 60
)

type SessionStore struct{ DB *sql.DB }

// Create inserts a 30 day session and returns the plaintext token (only its hash is stored).
func (s *SessionStore) Create(ctx context.Context, userID string) (string, error) {
	tok, hash, err := NewToken()
	if err != nil {
		return "", err
	}
	err = pg.WithPlatformTx(ctx, s.DB, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `insert into sessions (user_id, token_hash, expires_at) values ($1, $2, now() + interval '30 days')`, userID, hash)
		return err
	})
	return tok, err
}

// Lookup resolves a token. Unknown, malformed or expired tokens return httpx.Unauthenticated();
// an expired row is deleted. slid reports that expiry was extended (caller re-sets the cookie).
func (s *SessionStore) Lookup(ctx context.Context, token string) (a Actor, slid bool, err error) {
	if !validToken(token) {
		return a, false, httpx.Unauthenticated()
	}
	hash := hashToken(token)
	found := false
	err = pg.WithPlatformTx(ctx, s.DB, func(tx *sql.Tx) error {
		var id string
		var expired, stale bool
		err := tx.QueryRowContext(ctx, `select s.id, s.user_id, u.email, s.expires_at <= now(), s.last_seen_at < now() - interval '1 day'
			from sessions s join users u on u.id = s.user_id where s.token_hash = $1`, hash).Scan(&id, &a.UserID, &a.Email, &expired, &stale)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if expired {
			_, err = tx.ExecContext(ctx, `delete from sessions where token_hash = $1 and expires_at <= now()`, hash)
			return err
		}
		found = true
		if stale {
			slid = true
			_, err = tx.ExecContext(ctx, `update sessions set last_seen_at = now(), expires_at = now() + interval '30 days' where id = $1`, id)
		}
		return err
	})
	if err != nil {
		return Actor{}, false, err
	}
	if !found {
		return Actor{}, false, httpx.Unauthenticated()
	}
	return a, slid, nil
}

func (s *SessionStore) Revoke(ctx context.Context, token string) error {
	return pg.WithPlatformTx(ctx, s.DB, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `delete from sessions where token_hash = $1`, hashToken(token))
		return err
	})
}

// Authenticate reads the session cookie, slides expiry (re-setting the cookie) and records the
// user for the request log. Absent or expired sessions return httpx.Unauthenticated().
func Authenticate(w http.ResponseWriter, r *http.Request, s *SessionStore) (Actor, error) {
	c, err := r.Cookie(SessionCookie)
	if err != nil {
		return Actor{}, httpx.Unauthenticated()
	}
	a, slid, err := s.Lookup(r.Context(), c.Value)
	if err != nil {
		return Actor{}, err
	}
	if slid {
		setSessionCookie(w, c.Value)
	}
	httpx.WithUserID(r.Context(), a.UserID)
	return a, nil
}

func RequireSession(store *SessionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			a, err := Authenticate(w, r, store)
			if err != nil {
				httpx.WriteError(w, r, err)
				return
			}
			next.ServeHTTP(w, r.WithContext(withActor(r.Context(), a)))
		})
	}
}

func setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{Name: SessionCookie, Value: token, Path: "/", MaxAge: sessionMaxAge,
		HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: SessionCookie, Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode})
}
