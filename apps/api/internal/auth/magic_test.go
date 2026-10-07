package auth

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/outbox"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/testdb"
)

var urlRe = regexp.MustCompile(`https?://\S+`)

// lastURL applies the design §8 parse rule: the URL in the last "mail" log line.
func lastURL(t *testing.T, buf *bytes.Buffer) string {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		var m struct{ Msg, To, Text string }
		if json.Unmarshal([]byte(lines[i]), &m) == nil && m.Msg == "mail" {
			if u := urlRe.FindString(m.Text); u != "" {
				return u
			}
		}
	}
	t.Fatal("no mail line with a URL")
	return ""
}

func newMagic(t *testing.T) (*MagicLinks, *sql.DB, testdb.Fixture, *bytes.Buffer) {
	t.Helper()
	db := testdb.Open(t)
	fx := testdb.Seed(t, db)
	var buf bytes.Buffer
	return &MagicLinks{DB: db, Mailer: outbox.LogMailer{L: slog.New(slog.NewJSONHandler(&buf, nil))}, BaseURL: "http://api.test"}, db, fx, &buf
}

func tokenFromURL(t *testing.T, u string) string {
	t.Helper()
	i := strings.Index(u, "/api/v1/auth/magic-link/")
	if i < 0 {
		t.Fatalf("unexpected url %q", u)
	}
	tok := u[i+len("/api/v1/auth/magic-link/"):]
	if j := strings.IndexByte(tok, '?'); j >= 0 {
		tok = tok[:j]
	}
	return tok
}

func wantLinkErr(t *testing.T, err error, msg string) {
	t.Helper()
	var he *httpx.Error
	if !errors.As(err, &he) || he.Code != "validation_failed" || he.Status != 400 || he.Message != msg {
		t.Fatalf("want validation_failed %q, got %v", msg, err)
	}
}

func TestMagicSendAndConsume(t *testing.T) {
	m, db, _, buf := newMagic(t)
	ctx := context.Background()

	if err := m.Send(ctx, "  Alice@A.EDU ", ""); err != nil {
		t.Fatal(err)
	}
	if n := sqlCount(t, db, `select count(*) from magic_links where email = 'alice@a.edu'`); n != 1 {
		t.Fatalf("rows %d", n)
	}
	u := lastURL(t, buf)
	if !strings.HasPrefix(u, "http://api.test/api/v1/auth/magic-link/") {
		t.Fatalf("url %q", u)
	}
	tok := tokenFromURL(t, u)
	if n := sqlCount(t, db, `select count(*) from magic_links where token_hash = $1`, hashToken(tok)); n != 1 {
		t.Fatal("hash not stored")
	}
	if n := sqlCount(t, db, `select count(*) from magic_links where token_hash = $1`, []byte(tok)); n != 0 {
		t.Fatal("plaintext stored")
	}

	if email, err := m.Consume(ctx, tok); err != nil || email != "alice@a.edu" {
		t.Fatalf("consume: %q %v", email, err)
	}
	_, err := m.Consume(ctx, tok)
	wantLinkErr(t, err, "link already used")
}

func TestMagicSendNextAndNotAllowed(t *testing.T) {
	m, db, _, buf := newMagic(t)
	ctx := context.Background()

	buf.Reset()
	if err := m.Send(ctx, "x@nowhere.org", ""); err != nil {
		t.Fatal(err)
	}
	if n := sqlCount(t, db, `select count(*) from magic_links`); n != 0 || buf.Len() != 0 {
		t.Fatalf("not-allowed email left rows=%d mail=%q", n, buf.String())
	}

	if err := m.Send(ctx, "bob@a.edu", "/a/admin"); err != nil {
		t.Fatal(err)
	}
	if u := lastURL(t, buf); !strings.HasSuffix(u, "?next=%2Fa%2Fadmin") {
		t.Fatalf("next missing: %q", u)
	}
}

func TestMagicSendInvitedExternalEmail(t *testing.T) {
	m, db, fx, buf := newMagic(t)
	ctx := context.Background()
	insert := func(expires string) {
		t.Helper()
		_, hash, _ := NewToken()
		err := pg.WithTx(ctx, db, fx.UniversityA, func(tx *sql.Tx) error {
			_, err := tx.Exec(`insert into invitations (university_id, email, role, token_hash, expires_at) values ($1, 'adj@other.org', 'member', $2, now() + $3::interval)`,
				fx.UniversityA, hash, expires)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	insert("-1 second")
	if err := m.Send(ctx, "adj@other.org", ""); err != nil || sqlCount(t, db, `select count(*) from magic_links`) != 0 {
		t.Fatal("expired invitation must not allow")
	}
	insert("14 days")
	if err := m.Send(ctx, "adj@other.org", ""); err != nil || sqlCount(t, db, `select count(*) from magic_links`) != 1 {
		t.Fatal("pending invitation must allow")
	}
	lastURL(t, buf)
}

func TestMagicSendBadEmail(t *testing.T) {
	m, _, _, _ := newMagic(t)
	var he *httpx.Error
	if err := m.Send(context.Background(), "nope", ""); !errors.As(err, &he) || he.Code != "validation_failed" {
		t.Fatalf("got %v", err)
	}
}

func TestMagicConsumeFailures(t *testing.T) {
	m, db, _, buf := newMagic(t)
	ctx := context.Background()
	mk := func() (string, []byte) {
		t.Helper()
		if err := m.Send(ctx, "alice@a.edu", ""); err != nil {
			t.Fatal(err)
		}
		tok := tokenFromURL(t, lastURL(t, buf))
		return tok, hashToken(tok)
	}

	tok, h := mk()
	sqlExec(t, db, `update magic_links set expires_at = now() - interval '1 second' where token_hash = $1`, h)
	_, err := m.Consume(ctx, tok)
	wantLinkErr(t, err, "link expired")

	tok, h = mk()
	sqlExec(t, db, `update magic_links set used_at = now(), expires_at = now() - interval '1 second' where token_hash = $1`, h)
	_, err = m.Consume(ctx, tok)
	wantLinkErr(t, err, "link already used")

	_, err = m.Consume(ctx, "garbage")
	wantLinkErr(t, err, "invalid link")
	unknown, _, _ := NewToken()
	_, err = m.Consume(ctx, unknown)
	wantLinkErr(t, err, "invalid link")
}

func TestMagicConsumeRace(t *testing.T) {
	m, _, _, buf := newMagic(t)
	ctx := context.Background()
	if err := m.Send(ctx, "alice@a.edu", ""); err != nil {
		t.Fatal(err)
	}
	tok := tokenFromURL(t, lastURL(t, buf))
	const n = 10
	var wg sync.WaitGroup
	var mu sync.Mutex
	wins, used := 0, 0
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := m.Consume(ctx, tok)
			mu.Lock()
			defer mu.Unlock()
			var he *httpx.Error
			switch {
			case err == nil:
				wins++
			case errors.As(err, &he) && he.Message == "link already used":
				used++
			default:
				t.Errorf("unexpected: %v", err)
			}
		}()
	}
	wg.Wait()
	if wins != 1 || used != n-1 {
		t.Fatalf("wins=%d used=%d", wins, used)
	}
}
