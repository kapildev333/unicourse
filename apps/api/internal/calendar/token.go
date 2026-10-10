package calendar

import (
	"crypto/sha256"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/audit"
	"github.com/kapildev333/unicourse/apps/api/internal/auth"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

type feedInfo struct {
	URL       string    `json:"url,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

func getFeed(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	var out feedInfo
	err := tx.QueryRowContext(r.Context(), `select created_at from calendar_feeds where user_id = $1`, s.UserID).Scan(&out.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil, httpx.NotFound("calendar feed")
	}
	out.CreatedAt = out.CreatedAt.UTC()
	return http.StatusOK, out, err
}

// createFeed replaces the member's feed (create and rotate are the same call). The URL is returned once;
// only the token's SHA-256 is stored, and the audit entry carries neither.
func createFeed(d tenant.Deps) tenant.Handler {
	return func(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
		ctx := r.Context()
		token, hash, err := auth.NewToken()
		if err != nil {
			return 0, nil, err
		}
		if _, err := tx.ExecContext(ctx, `delete from calendar_feeds where user_id = $1`, s.UserID); err != nil {
			return 0, nil, err
		}
		var id string
		out := feedInfo{URL: strings.TrimRight(d.BaseURL, "/") + "/api/v1/universities/" + r.PathValue("slug") + "/calendar-feeds/" + token + ".ics"}
		if err := tx.QueryRowContext(ctx, `insert into calendar_feeds (university_id, user_id, token_hash) values ($1, $2, $3) returning id, created_at`,
			s.UniversityID, s.UserID, hash).Scan(&id, &out.CreatedAt); err != nil {
			return 0, nil, err
		}
		out.CreatedAt = out.CreatedAt.UTC()
		if err := audit.Write(ctx, tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "calendar_feed.create", EntityType: "calendar_feed", EntityID: id,
			After: map[string]string{"userId": s.UserID}}); err != nil {
			return 0, nil, err
		}
		return http.StatusCreated, tenant.After(func(w http.ResponseWriter, _ *http.Request) error {
			w.Header().Set("Cache-Control", "no-store")
			httpx.WriteJSON(w, http.StatusCreated, out)
			return nil
		}), nil
	}
}

func deleteFeed(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	var id string
	err := tx.QueryRowContext(r.Context(), `delete from calendar_feeds where user_id = $1 returning id`, s.UserID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil, httpx.NotFound("calendar feed")
	}
	if err != nil {
		return 0, nil, err
	}
	return http.StatusNoContent, nil, audit.Write(r.Context(), tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "calendar_feed.delete", EntityType: "calendar_feed", EntityID: id,
		Before: map[string]string{"userId": s.UserID}})
}

// hashToken is the stored form of a feed token, the same SHA-256 auth uses for sessions.
func hashToken(t string) []byte {
	h := sha256.Sum256([]byte(t))
	return h[:]
}
