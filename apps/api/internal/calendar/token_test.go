package calendar

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

type feedOut struct {
	URL       string
	CreatedAt string
}

// mint POSTs the feed of email and returns its URL path (after the host) and token.
func (e *env) mint(email string) (path, token string) {
	e.T.Helper()
	st, raw := e.call(e.cookieOf(email), "POST", "/me/calendar-feed", nil)
	var out feedOut
	if st != 201 || json.Unmarshal(raw, &out) != nil || out.CreatedAt == "" {
		e.T.Fatalf("create: %d %s", st, raw)
	}
	const pre = "http://app.test"
	if !strings.HasPrefix(out.URL, pre+"/api/v1/universities/a/calendar-feeds/") || !strings.HasSuffix(out.URL, ".ics") {
		e.T.Fatalf("url %s", out.URL)
	}
	path = strings.TrimPrefix(out.URL, pre)
	return path, strings.TrimSuffix(path[strings.LastIndex(path, "/")+1:], ".ics")
}

func TestFeedTokenRoutes(t *testing.T) {
	e := newEnv(t)
	if st, _ := e.call(e.Stud, "GET", "/me/calendar-feed", nil); st != 404 {
		t.Fatalf("none: %d", st)
	}
	if st, _ := e.call(e.Stud, "DELETE", "/me/calendar-feed", nil); st != 404 {
		t.Fatalf("delete none: %d", st)
	}
	if st, _ := e.call(nil, "POST", "/me/calendar-feed", nil); st != 401 {
		t.Fatalf("anonymous: %d", st)
	}

	_, tok1 := e.mint("stud@a.edu")
	var hash []byte
	var rows int
	e.tx(func(tx *sql.Tx) error {
		return tx.QueryRow(`select count(*), min(token_hash) from calendar_feeds where user_id = $1`, e.UserID["stud@a.edu"]).Scan(&rows, &hash)
	})
	sum := sha256.Sum256([]byte(tok1))
	if rows != 1 || string(hash) != string(sum[:]) {
		t.Fatalf("stored %d rows, hash match %v", rows, string(hash) == string(sum[:]))
	}
	if st, raw := e.call(e.Stud, "GET", "/me/calendar-feed", nil); st != 200 || !strings.Contains(string(raw), `"createdAt"`) || strings.Contains(string(raw), tok1) {
		t.Fatalf("get: %d %s", st, raw)
	}

	// rotate: a new token replaces the old one
	_, tok2 := e.mint("stud@a.edu")
	e.tx(func(tx *sql.Tx) error {
		return tx.QueryRow(`select count(*) from calendar_feeds where user_id = $1`, e.UserID["stud@a.edu"]).Scan(&rows)
	})
	if tok2 == tok1 || rows != 1 {
		t.Fatalf("rotate: same=%v rows=%d", tok1 == tok2, rows)
	}

	// own membership only
	if st, _ := e.call(e.Stud2, "GET", "/me/calendar-feed", nil); st != 404 {
		t.Fatalf("stud2 sees stud's feed: %d", st)
	}
	if st, _ := e.call(e.Stud2, "DELETE", "/me/calendar-feed", nil); st != 404 {
		t.Fatalf("stud2 deleted stud's feed: %d", st)
	}
	if st, _ := e.call(e.Stud, "DELETE", "/me/calendar-feed", nil); st != 204 {
		t.Fatalf("delete: %d", st)
	}
	if st, _ := e.call(e.Stud, "GET", "/me/calendar-feed", nil); st != 404 {
		t.Fatalf("after delete: %d", st)
	}

	// audit: create, create (rotate), delete; never the token or its hash
	var actions string
	var blob string
	e.tx(func(tx *sql.Tx) error {
		return tx.QueryRow(`select coalesce(string_agg(action, ',' order by at, id), ''), coalesce(string_agg(coalesce(before::text, '') || coalesce(after::text, ''), ''), '')
			from audit_log where entity_type = 'calendar_feed'`).Scan(&actions, &blob)
	})
	if actions != "calendar_feed.create,calendar_feed.create,calendar_feed.delete" {
		t.Fatalf("audit %v", actions)
	}
	if strings.Contains(blob, tok1) || strings.Contains(blob, tok2) || !strings.Contains(blob, e.UserID["stud@a.edu"]) {
		t.Fatalf("audit payload %s", blob)
	}
}

func TestFeedTokenNoStore(t *testing.T) {
	e := newEnv(t)
	req, _ := http.NewRequest("POST", e.Srv.URL+"/api/v1/universities/a/me/calendar-feed", nil)
	req.AddCookie(e.Stud)
	res, err := http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != 201 || res.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("%v %d %q", err, res.StatusCode, res.Header.Get("Cache-Control"))
	}
	res.Body.Close()
}
