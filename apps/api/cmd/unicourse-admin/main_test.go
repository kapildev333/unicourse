package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/kapildev333/unicourse/apps/api/internal/auth"
	"github.com/kapildev333/unicourse/apps/api/internal/outbox"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/testdb"
)

const baseURL = "http://localhost:8080"

func env(k string) string {
	if k == "UNICOURSE_BASE_URL" {
		return baseURL
	}
	return ""
}

func admin(t *testing.T, db *sql.DB, args ...string) (code int, out, errOut string) {
	t.Helper()
	var o, e bytes.Buffer
	code = run(context.Background(), args, env, func() (*sql.DB, error) { return db, nil }, &o, &e)
	return code, o.String(), e.String()
}

// count runs q inside every university's tenant scope (the tables are under row level security) and sums.
func count(t *testing.T, db *sql.DB, q string) (total int) {
	t.Helper()
	var ids []string
	rows, err := db.Query(`select id from universities`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close()
	if len(ids) == 0 {
		ids = []string{""}
	}
	for _, id := range ids {
		var n int
		err := pg.WithTx(context.Background(), db, id, func(tx *sql.Tx) error { return tx.QueryRow(q).Scan(&n) })
		if err != nil && id != "" {
			t.Fatal(err)
		}
		total += n
	}
	return total
}

func create(extra ...string) []string {
	a := []string{"create-university", "--name", "Example", "--slug", "example", "--domains", "Example.edu, ex.edu",
		"--timezone", "America/New_York", "--admin-email", "Root@Example.edu"}
	return append(a, extra...)
}

func TestCreateUniversityAndAccept(t *testing.T) {
	_, db := testdb.OpenWithOwner(t)
	code, out, errOut := admin(t, db, create()...)
	if code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if !strings.HasPrefix(lines[len(lines)-1], baseURL+"/invite/") {
		t.Fatalf("last line is not the invitation url: %q", out)
	}
	if n := count(t, db, `select count(*) from outbox_mails`); n != 0 {
		t.Fatalf("CLI queued %d mails (D28)", n)
	}
	var tz, loc, doms string
	if err := db.QueryRow(`select timezone, locale, array_to_string(allowed_email_domains, ',') from universities where slug = 'example'`).Scan(&tz, &loc, &doms); err != nil {
		t.Fatal(err)
	}
	if tz != "America/New_York" || loc != "en-US" || doms != "example.edu,ex.edu" {
		t.Fatalf("got %q %q %q", tz, loc, doms)
	}
	if n := count(t, db, `select count(*) from invitations where email = 'root@example.edu' and role = 'admin'`); n != 1 {
		t.Fatalf("invitations = %d", n)
	}
	if n := count(t, db, `select count(*) from audit_log where action = 'invitation.create' and actor_user_id is null`); n != 1 {
		t.Fatalf("audit rows = %d", n)
	}
	if _, err := auth.SignIn(context.Background(), db, "root@example.edu", ""); err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, `select count(*) from memberships m join users u on u.id = m.user_id where u.email = 'root@example.edu' and m.role = 'admin' and m.status = 'active'`); n != 1 {
		t.Fatalf("memberships = %d", n)
	}
}

func TestCreateUniversityRejects(t *testing.T) {
	_, db := testdb.OpenWithOwner(t)
	for name, tc := range map[string]struct {
		args []string
		want string
	}{
		"timezone": {create("--timezone", "Mars/Base"), "timezone"},
		"slug":     {create("--slug", "Bad_Slug"), "slug"},
		"domains":  {create("--domains", "a@b.edu"), "domains"},
		"email":    {create("--admin-email", "nope"), "admin-email"},
		"reserved": {create("--slug", "invite"), "reserved"},
		"local":    {create("--timezone", "Local"), "timezone"},
		"locale":   {create("--locale", "english_US"), "locale"},
		"emptyloc": {create("--locale", ""), "locale"},
		"nul":      {create("--name", "a\x00b"), "name"},
		"ctrl":     {create("--name", "a\nb"), "name"},
		"long":     {create("--name", strings.Repeat("x", 201)), "name"},
		"stray":    {create("extra"), "unexpected argument"},
		"missing":  {[]string{"create-university"}, "--name, --slug, --domains, --timezone, --admin-email"},
	} {
		if code, _, errOut := admin(t, db, tc.args...); code != 2 || !strings.Contains(errOut, tc.want) {
			t.Errorf("%s: code %d, stderr %q", name, code, errOut)
		}
	}
	if n := count(t, db, `select count(*) from universities`); n != 0 {
		t.Fatalf("universities = %d", n)
	}
}

func TestCreateUniversityDuplicateSlug(t *testing.T) {
	_, db := testdb.OpenWithOwner(t)
	if code, _, e := admin(t, db, create()...); code != 0 {
		t.Fatal(e)
	}
	code, _, errOut := admin(t, db, create("--admin-email", "other@example.edu")...)
	if code != 2 || !strings.Contains(errOut, "slug already exists") {
		t.Fatalf("code %d, stderr %q", code, errOut)
	}
	if n := count(t, db, `select count(*) from universities`); n != 1 {
		t.Fatalf("universities = %d", n)
	}
	if n := count(t, db, `select count(*) from invitations`); n != 1 {
		t.Fatalf("invitations = %d", n)
	}
}

func TestMigrateAndUsage(t *testing.T) {
	_, db := testdb.OpenWithOwner(t)
	if code, _, e := admin(t, db, "migrate"); code != 0 {
		t.Fatalf("code %d: %s", code, e)
	}
	for _, args := range [][]string{nil, {"bogus"}, {"-h"}} {
		if code, _, e := admin(t, db, args...); code != 2 || !strings.Contains(e, "usage") {
			t.Errorf("%v: code %d, stderr %q", args, code, e)
		}
	}
}

func TestMigrateAsAPIRoleFails(t *testing.T) {
	api := testdb.Open(t)
	if code, _, e := admin(t, api, "migrate"); code != 1 || !strings.Contains(e, "permission denied") {
		t.Fatalf("code %d, stderr %q, want 1 and permission denied", code, e)
	}
}

type failMailer struct{}

func (failMailer) Enqueue(context.Context, *sql.Tx, ...outbox.Mail) error { return errors.New("boom") }
func (failMailer) URL() string                                            { return "" }

func TestCreateUniversityIsAtomic(t *testing.T) {
	_, db := testdb.OpenWithOwner(t)
	old := newMailer
	newMailer = func(string) captor { return failMailer{} }
	defer func() { newMailer = old }()
	if code, _, e := admin(t, db, create()...); code != 1 || !strings.Contains(e, "boom") {
		t.Fatalf("code %d: %s", code, e)
	}
	if n := count(t, db, `select count(*) from universities`); n != 0 {
		t.Fatalf("universities = %d", n)
	}
}

func TestDomainsDeduplicated(t *testing.T) {
	_, db := testdb.OpenWithOwner(t)
	if code, _, e := admin(t, db, create("--domains", "a.edu,A.edu, a.edu")...); code != 0 {
		t.Fatal(e)
	}
	if n := count(t, db, `select count(*) from universities where cardinality(allowed_email_domains) = 1`); n != 1 {
		t.Fatalf("domains not deduplicated")
	}
}

func TestUsageNeedsNoDatabase(t *testing.T) {
	open := func() (*sql.DB, error) { return nil, errors.New("database opened") }
	for _, args := range [][]string{nil, {"-h"}, {"create-university", "-h"}, {"create-university"}, {"bogus"}} {
		var o, e bytes.Buffer
		if code := run(context.Background(), args, env, open, &o, &e); code != 2 || strings.Contains(e.String(), "database opened") {
			t.Errorf("%v: code %d, stderr %q", args, code, e.String())
		}
	}
}
