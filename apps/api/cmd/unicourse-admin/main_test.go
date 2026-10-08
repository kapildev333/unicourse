package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/auth"
	"github.com/kapildev333/unicourse/apps/api/internal/blob"
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
	for _, args := range [][]string{nil, {"-h"}, {"create-university", "-h"}, {"create-university"}, {"seed-demo", "-h"}, {"bogus"}} {
		var o, e bytes.Buffer
		if code := run(context.Background(), args, env, open, &o, &e); code != 2 || strings.Contains(e.String(), "database opened") {
			t.Errorf("%v: code %d, stderr %q", args, code, e.String())
		}
	}
}

func TestCreateUniversitySeedsScale(t *testing.T) {
	_, db := testdb.OpenWithOwner(t)
	if code, _, errOut := admin(t, db, create()...); code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	var id string
	if err := db.QueryRow(`select id from universities where slug = 'example'`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	var got []string
	err := pg.WithTx(context.Background(), db, id, func(tx *sql.Tx) error {
		rows, err := tx.Query(`select letter, min_percent::text, points::text from grade_scales order by min_percent desc`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var l, m, p string
			rows.Scan(&l, &m, &p)
			got = append(got, l+" "+m+" "+p)
		}
		return rows.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "A 93.00 4.00,A- 90.00 3.70,B+ 87.00 3.30,B 83.00 3.00,B- 80.00 2.70,C+ 77.00 2.30,C 73.00 2.00,C- 70.00 1.70,D+ 67.00 1.30,D 63.00 1.00,D- 60.00 0.70,F 0.00 0.00"
	if strings.Join(got, ",") != want {
		t.Fatalf("scale = %v", got)
	}
}

func TestPruneFiles(t *testing.T) {
	_, db := testdb.OpenWithOwner(t)
	if code, _, errOut := admin(t, db, create()...); code != 0 {
		t.Fatalf("create: %d %s", code, errOut)
	}
	var uni, user string
	if err := db.QueryRow(`select id from universities where slug = 'example'`).Scan(&uni); err != nil {
		t.Fatal(err)
	}
	if err := pg.WithPlatformTx(context.Background(), db, func(tx *sql.Tx) error {
		return tx.QueryRow(`insert into users (email) values ('u@example.edu') returning id`).Scan(&user)
	}); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	files, err := blob.New(root)
	if err != nil {
		t.Fatal(err)
	}
	put := func(id string) {
		tmp, err := files.Put(uni, strings.NewReader("x"))
		if err == nil {
			err = files.Commit(tmp, uni, id)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	var good, gone string
	err = pg.WithTx(context.Background(), db, uni, func(tx *sql.Tx) error {
		var dept, course, term, sec, wk string
		if err := tx.QueryRow(`insert into departments (university_id, name, code) values ($1, 'D', 'D') returning id`, uni).Scan(&dept); err != nil {
			return err
		}
		if err := tx.QueryRow(`insert into terms (university_id, name, starts_on, ends_on) values ($1, 'T', '2026-09-01', '2026-12-01') returning id`, uni).Scan(&term); err != nil {
			return err
		}
		if err := tx.QueryRow(`insert into courses (university_id, department_id, code, title) values ($1, $2, 'C1', 'C') returning id`, uni, dept).Scan(&course); err != nil {
			return err
		}
		if err := tx.QueryRow(`insert into sections (university_id, course_id, term_id, number) values ($1, $2, $3, '1') returning id`, uni, course, term).Scan(&sec); err != nil {
			return err
		}
		if err := tx.QueryRow(`insert into weeks (university_id, section_id, title, position) values ($1, $2, 'W', 1) returning id`, uni, sec).Scan(&wk); err != nil {
			return err
		}
		for i, dst := range []*string{&good, &gone} {
			if err := tx.QueryRow(`insert into materials (university_id, section_id, week_id, title, file_name, content_type, size_bytes, sha256, position, uploaded_by_user_id)
				values ($1, $2, $3, 'T', 't.pdf', 'application/pdf', 1, repeat('a', 64), $4, $5) returning id`, uni, sec, wk, i+1, user).Scan(dst); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	const oldID, newID = "11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222"
	put(good)
	put(oldID)
	put(newID)
	twoHours := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(filepath.Join(root, uni, oldID), twoHours, twoHours); err != nil {
		t.Fatal(err)
	}
	if _, err := files.Put(uni, strings.NewReader("x")); err != nil { // a young .tmp file
		t.Fatal(err)
	}
	const oldTmpName = "up-crashed"
	tmpDir := filepath.Join(root, uni, ".tmp")
	twoDays := time.Now().Add(-48 * time.Hour)
	if err := os.WriteFile(filepath.Join(tmpDir, oldTmpName), []byte("x"), 0o600); err != nil { // a crashed upload from two days ago
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath.Join(tmpDir, oldTmpName), twoDays, twoDays); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) (int, string, string) {
		var o, e bytes.Buffer
		envf := func(k string) string {
			if k == "UNICOURSE_FILES_DIR" {
				return root
			}
			return ""
		}
		code := runCmd(args, envf, db, &o, &e)
		return code, o.String(), e.String()
	}
	exists := func(id string) bool { _, err := os.Stat(filepath.Join(root, uni, id)); return err == nil }

	code, out, errOut := run("prune-files")
	want := "orphan " + uni + "/.tmp/" + oldTmpName + "\norphan " + uni + "/" + oldID + "\nmissing " + uni + "/" + gone + "\n2 orphan files, 1 missing file\n"
	if code != 0 || out != want || errOut != "" || !exists(oldID) {
		t.Fatalf("list: %d %q %q", code, out, errOut)
	}
	code, out, errOut = run("prune-files", "--delete")
	want = "removed " + uni + "/.tmp/" + oldTmpName + "\nremoved " + uni + "/" + oldID + "\nmissing " + uni + "/" + gone + "\n2 orphan files removed, 1 missing file\n"
	if code != 0 || out != want || errOut != "" {
		t.Fatalf("delete: %d %q %q", code, out, errOut)
	}
	if exists(oldID) || !exists(newID) || !exists(good) {
		t.Fatalf("old %v new %v good %v", exists(oldID), exists(newID), exists(good))
	}
	if ents, _ := os.ReadDir(tmpDir); len(ents) != 1 {
		t.Fatalf(".tmp = %v", ents)
	}
	if n := count(t, db, `select count(*) from materials`); n != 2 {
		t.Fatalf("rows touched: %d", n)
	}
}

func runCmd(args []string, env func(string) string, db *sql.DB, o, e io.Writer) int {
	return run(context.Background(), args, env, func() (*sql.DB, error) { return db, nil }, o, e)
}

func TestPruneFilesUsage(t *testing.T) {
	open := func() (*sql.DB, error) { return nil, errors.New("database opened") }
	dir := t.TempDir()
	for _, c := range []struct {
		args []string
		root string
		code int
		msg  string
	}{
		{[]string{"prune-files", "extra"}, dir, 2, "unexpected argument"},
		{[]string{"prune-files"}, "", 2, "UNICOURSE_FILES_DIR is not set"},
		{[]string{"prune-files"}, filepath.Join(dir, "missing"), 1, "UNICOURSE_FILES_DIR"},
	} {
		var o, e bytes.Buffer
		envf := func(string) string { return c.root }
		if code := run(context.Background(), c.args, envf, open, &o, &e); code != c.code || !strings.Contains(e.String(), c.msg) || strings.Contains(e.String(), "database opened") {
			t.Errorf("%v: code %d, stderr %q", c.args, code, e.String())
		}
	}
}

func mapEnv(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestSeedDemoUsage(t *testing.T) {
	dir := t.TempDir()
	both := map[string]string{"UNICOURSE_BASE_URL": baseURL, "UNICOURSE_FILES_DIR": dir}
	cases := []struct {
		args []string
		env  map[string]string
		code int
		want string
	}{
		{[]string{"seed-demo", "-h"}, both, 2, "seed-demo flags"},
		{[]string{"seed-demo", "extra"}, both, 2, "unexpected argument"},
		{[]string{"seed-demo", "--slug", "Bad_Slug"}, both, 2, "slug"},
		{[]string{"seed-demo", "--slug", "invite"}, both, 2, "reserved"},
		{[]string{"seed-demo"}, map[string]string{"UNICOURSE_FILES_DIR": dir}, 2, "UNICOURSE_BASE_URL is not set"},
		{[]string{"seed-demo"}, map[string]string{"UNICOURSE_BASE_URL": baseURL}, 2, "UNICOURSE_FILES_DIR is not set"},
		{[]string{"seed-demo"}, map[string]string{"UNICOURSE_BASE_URL": baseURL, "UNICOURSE_FILES_DIR": filepath.Join(dir, "missing")}, 1, "UNICOURSE_FILES_DIR"},
		{[]string{"-h"}, nil, 2, "seed-demo"},
	}
	open := func() (*sql.DB, error) { return nil, errors.New("database opened") }
	for _, c := range cases {
		var o, e bytes.Buffer
		code := run(context.Background(), c.args, mapEnv(c.env), open, &o, &e)
		if code != c.code || !strings.Contains(e.String(), c.want) || strings.Contains(e.String(), "database opened") {
			t.Errorf("%v: code %d, stderr %q", c.args, code, e.String())
		}
	}
}

func TestSeedDemo(t *testing.T) {
	_, db := testdb.OpenWithOwner(t)
	e := mapEnv(map[string]string{"UNICOURSE_BASE_URL": baseURL, "UNICOURSE_FILES_DIR": t.TempDir()})
	open := func() (*sql.DB, error) { return db, nil }
	var o, er bytes.Buffer
	if code := run(context.Background(), []string{"seed-demo"}, e, open, &o, &er); code != 0 {
		t.Fatalf("code %d: %s", code, er.String())
	}
	if !strings.HasPrefix(o.String(), "seeded Northfield University (northfield)") || !strings.Contains(o.String(), "registrar@northfield.edu") {
		t.Fatalf("stdout: %s", o.String())
	}
	o.Reset()
	er.Reset()
	if code := run(context.Background(), []string{"seed-demo"}, e, open, &o, &er); code != 2 || !strings.Contains(er.String(), "already exists; rerun with --reset") {
		t.Fatalf("second run: code %d: %s", code, er.String())
	}
}

func TestDatabaseURL(t *testing.T) {
	both := mapEnv(map[string]string{"UNICOURSE_OWNER_DATABASE_URL": "o", "UNICOURSE_PLATFORM_DATABASE_URL": "p"})
	owner := mapEnv(map[string]string{"UNICOURSE_OWNER_DATABASE_URL": "o"})
	for _, c := range []struct {
		cmd  string
		env  func(string) string
		want string
		name string
	}{
		{"create-university", both, "p", "UNICOURSE_PLATFORM_DATABASE_URL"},
		{"create-university", owner, "o", "UNICOURSE_OWNER_DATABASE_URL"},
		{"migrate", both, "o", "UNICOURSE_OWNER_DATABASE_URL"},
		{"prune-files", both, "o", "UNICOURSE_OWNER_DATABASE_URL"},
		{"seed-demo", both, "o", "UNICOURSE_OWNER_DATABASE_URL"},
		{"", both, "o", "UNICOURSE_OWNER_DATABASE_URL"},
	} {
		if u, n := databaseURL(c.cmd, c.env); u != c.want || n != c.name {
			t.Errorf("%q: got %q %q, want %q %q", c.cmd, u, n, c.want, c.name)
		}
	}
}

// openPlatform connects as unicourse_platform: UNICOURSE_TEST_PLATFORM_DATABASE_URL, else the test owner
// URL with its credentials replaced. Call it only after testdb.OpenWithOwner (migrated, lock held).
func openPlatform(t *testing.T) *sql.DB {
	t.Helper()
	raw := os.Getenv("UNICOURSE_TEST_PLATFORM_DATABASE_URL")
	if raw == "" {
		u, err := url.Parse(os.Getenv("UNICOURSE_TEST_OWNER_DATABASE_URL"))
		if err != nil {
			t.Fatal(err)
		}
		u.User = url.UserPassword("unicourse_platform", "unicourse_platform")
		raw = u.String()
	}
	db, err := pg.Open(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestCreateUniversityAsPlatformRole(t *testing.T) {
	_, owner := testdb.OpenWithOwner(t)
	platform := openPlatform(t)
	code, out, errOut := admin(t, platform, create()...)
	if code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	if !strings.Contains(out, baseURL+"/invite/") {
		t.Fatalf("no invitation url in %q", out)
	}
	if n := count(t, owner, `select count(*) from grade_scales`); n != 12 {
		t.Errorf("grade_scales rows = %d, want 12", n)
	}
	if n := count(t, owner, `select count(*) from invitations`); n != 1 {
		t.Errorf("invitations = %d, want 1", n)
	}
	if n := count(t, owner, `select count(*) from audit_log where action = 'invitation.create'`); n != 1 {
		t.Errorf("invitation.create audit rows = %d, want 1", n)
	}
}

func sqlState(err error) string {
	var pe interface{ SQLState() string }
	if errors.As(err, &pe) {
		return pe.SQLState()
	}
	return ""
}

func TestPlatformRoleCannotMigrate(t *testing.T) {
	testdb.OpenWithOwner(t)
	platform := openPlatform(t)
	if err := pg.Migrate(context.Background(), platform); err == nil || sqlState(err) != "42501" {
		t.Fatalf("Migrate as platform role: %v (state %q), want 42501", err, sqlState(err))
	}
}

func TestPlatformRoleIsNotOwner(t *testing.T) {
	testdb.OpenWithOwner(t)
	platform := openPlatform(t)
	if err := pg.CheckAPIRole(context.Background(), platform); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := platform.QueryRow(`select count(*) from departments`).Scan(&n); sqlState(err) != "42501" {
		t.Fatalf("select departments as platform role: %v, want 42501", err)
	}
}

// D135: even with app.university_id set to an existing university, the platform role sees no rows and
// writes none in the tables create-university uses.
func TestPlatformRoleCannotActInExistingUniversity(t *testing.T) {
	api, _ := testdb.OpenWithOwner(t)
	fx := testdb.Seed(t, api)
	platform := openPlatform(t)
	uni := fx.UniversityA
	err := pg.WithPlatformTx(context.Background(), platform, func(tx *sql.Tx) error {
		if err := pg.SetUniversity(context.Background(), tx, uni); err != nil {
			return err
		}
		var n int
		if err := tx.QueryRow(`select count(*) from memberships`).Scan(&n); err != nil || n != 0 {
			t.Errorf("memberships visible: %d, %v", n, err)
		}
		if err := tx.QueryRow(`select count(*) from invitations`).Scan(&n); err != nil || n != 0 {
			t.Errorf("invitations visible: %d, %v", n, err)
		}
		_, err := tx.Exec(`insert into invitations (university_id, email, role, token_hash, expires_at)
			values ($1, 'x@example.edu', 'admin', '\x01', now() + interval '1 day')`, uni)
		if sqlState(err) != "42501" {
			t.Errorf("invitation insert into existing university: %v, want 42501", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
