package auth

import (
	"context"
	"database/sql"
	"errors"
	"net/url"

	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/outbox"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
)

type MagicLinks struct {
	DB      *sql.DB
	Wake    func() // nudges the outbox sender after commit; may be nil
	BaseURL string
}

func linkErr(msg string) error {
	return &httpx.Error{Status: 400, Code: "validation_failed", Message: msg}
}

// Send queues a 15 minute sign-in link in the outbox (sent after commit, never in the request). Emails
// that are not allowed (no matching domain, no pending invitation) return nil with no row and no mail,
// so callers cannot enumerate. next must already be validated by the caller; it is not stored.
func (m *MagicLinks) Send(ctx context.Context, email, next string) error {
	email, err := NormalizeEmail(email)
	if err != nil {
		return err
	}
	tok, hash, err := NewToken()
	if err != nil {
		return err
	}
	link := m.BaseURL + "/api/v1/auth/magic-link/" + tok
	if next != "" {
		link += "?next=" + url.QueryEscape(next)
	}
	ok := false
	err = pg.WithUserTx(ctx, m.DB, "", email, func(tx *sql.Tx) error {
		if ok, err = allowed(ctx, tx, email); err != nil || !ok {
			return err
		}
		_, err = tx.ExecContext(ctx, `insert into magic_links (email, token_hash, expires_at) values ($1, $2, now() + interval '15 minutes')`, email, hash)
		if err != nil {
			return err
		}
		return outbox.Enqueue(ctx, tx, outbox.Mail{To: email, Subject: "Your UniCourse sign-in link",
			Text: "Use this link to sign in (valid for 15 minutes):\n" + link + "\n", Urgent: true})
	})
	if err == nil && ok && m.Wake != nil {
		m.Wake()
	}
	return err
}

// Consume marks the link used and returns its email. The single UPDATE enforces single use and
// expiry atomically, so exactly one concurrent caller wins.
func (m *MagicLinks) Consume(ctx context.Context, token string) (email string, err error) {
	if !validToken(token) {
		return "", linkErr("invalid link")
	}
	hash := hashToken(token)
	err = pg.WithPlatformTx(ctx, m.DB, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx, `update magic_links set used_at = now() where token_hash = $1 and used_at is null and expires_at > now() returning email`, hash).Scan(&email)
		if err == nil {
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		var used sql.NullTime
		err = tx.QueryRowContext(ctx, `select used_at from magic_links where token_hash = $1`, hash).Scan(&used)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return linkErr("invalid link")
		case err != nil:
			return err
		case used.Valid:
			return linkErr("link already used")
		}
		return linkErr("link expired")
	})
	if err != nil {
		return "", err
	}
	return email, nil
}
