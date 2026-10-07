// Command unicourse-admin is the platform operator's CLI: apply migrations and create universities.
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strings"
	"text/tabwriter"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/kapildev333/unicourse/apps/api/internal/auth"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/blob"
	"github.com/kapildev333/unicourse/apps/api/internal/outbox"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/seed"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

const usage = `usage: unicourse-admin <command> [flags]

commands:
  migrate               apply database migrations
  create-university     create a university and an admin invitation (see create-university -h)
  prune-files [--delete]  list (or remove) files no material references, and materials whose file is missing
  seed-demo [--slug s] [--reset]  load the demo university (see seed-demo -h)

environment: UNICOURSE_OWNER_DATABASE_URL (all commands; the owner role, not the API role), UNICOURSE_BASE_URL (create-university, seed-demo), UNICOURSE_FILES_DIR (prune-files, seed-demo)
`

const createHelp = `create-university flags:
  --name         university name (required)
  --slug         url slug, lowercase letters, digits, hyphens (required)
  --domains      comma separated allowed email domains, e.g. a.edu,b.edu (required)
  --timezone     IANA timezone, e.g. America/New_York (required)
  --admin-email  email of the first admin (required; may be outside --domains)
  --locale       BCP 47 locale (default en-US)

No mail is sent, even if UNICOURSE_SMTP_URL is set: the invitation URL is printed on stdout
and the operator forwards it to the admin. Invalid input (including a duplicate slug) exits 2.
`

const seedHelp = `seed-demo flags:
  --slug    url slug of the demo university (default northfield); emails use <slug>.edu
  --reset   first delete the university with this slug and its files (only that one)

Loads the embedded Northfield University dataset (internal/seed/demo.json) through the API's own
handlers: departments, terms, courses, sections, staff, rosters, grades and section content.
For local databases. No mail is queued; personas sign in with a magic link from the API log.
An existing slug without --reset exits 2. Needs UNICOURSE_BASE_URL and UNICOURSE_FILES_DIR.
`

var (
	slugRE   = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
	localeRE = regexp.MustCompile(`^[A-Za-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)
	domainRE = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$`)
)

// reserved slugs collide with top-level web routes, which win over /{slug}/...
var reserved = []string{"api", "assets", "invite", "sign-in", "healthz", "readyz", "index.html", "favicon.ico"}

// captor stands in for outbox.Enqueue: it remembers the invitation URL and neither sends nor
// queues anything (D28: a running API process would otherwise mail the admin).
type captor interface {
	Enqueue(context.Context, *sql.Tx, ...outbox.Mail) error
	URL() string
}

// newMailer is a variable so tests can inject a failing enqueuer.
var newMailer = func(prefix string) captor { return &capture{prefix: prefix} }

type capture struct{ prefix, url string }

func (m *capture) URL() string { return m.url }

func (m *capture) Enqueue(_ context.Context, _ *sql.Tx, mails ...outbox.Mail) error {
	for _, mail := range mails {
		for _, l := range strings.Split(mail.Text, "\n") {
			if strings.HasPrefix(l, m.prefix) {
				m.url = l
			}
		}
	}
	return nil
}

// run executes one command. open is called only once the arguments are valid, so usage and
// validation errors never need a database. Usage (including -h) exits 2.
func run(ctx context.Context, args []string, env func(string) string, open func() (*sql.DB, error), stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprint(stderr, usage)
		return 2
	}
	switch args[0] {
	case "migrate":
		if len(args) > 1 {
			fmt.Fprintf(stderr, "migrate: unexpected argument %q\n", args[1])
			return 2
		}
		db, err := open()
		if err != nil {
			fmt.Fprintln(stderr, "database:", err)
			return 1
		}
		if err := pg.Migrate(ctx, db); err != nil {
			fmt.Fprintln(stderr, "migrate:", err)
			return 1
		}
		fmt.Fprintln(stdout, "migrations applied")
		return 0
	case "create-university":
		return createUniversity(ctx, args[1:], env, open, stdout, stderr)
	case "prune-files":
		return pruneFiles(ctx, args[1:], env, open, stdout, stderr)
	case "seed-demo":
		return seedDemo(ctx, args[1:], env, open, stdout, stderr)
	}
	fmt.Fprintf(stderr, "unknown command %q\n%s", args[0], usage)
	return 2
}

func createUniversity(ctx context.Context, args []string, env func(string) string, open func() (*sql.DB, error), stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("create-university", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, createHelp) }
	name := fs.String("name", "", "")
	slug := fs.String("slug", "", "")
	domains := fs.String("domains", "", "")
	timezone := fs.String("timezone", "", "")
	locale := fs.String("locale", "en-US", "")
	adminEmail := fs.String("admin-email", "", "")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "create-university: unexpected argument %q\n", fs.Arg(0))
		return 2
	}
	fail := func(format string, a ...any) int {
		fmt.Fprintf(stderr, "create-university: "+format+"\n", a...)
		return 2
	}

	var missing []string
	for _, f := range []struct{ flag, v string }{{"--name", *name}, {"--slug", *slug}, {"--domains", *domains},
		{"--timezone", *timezone}, {"--admin-email", *adminEmail}} {
		if strings.TrimSpace(f.v) == "" {
			missing = append(missing, f.flag)
		}
	}
	if len(missing) > 0 {
		return fail("missing required flags: %s", strings.Join(missing, ", "))
	}
	base := strings.TrimRight(env("UNICOURSE_BASE_URL"), "/")
	if base == "" {
		return fail("UNICOURSE_BASE_URL is not set")
	}
	if !slugRE.MatchString(*slug) {
		return fail("--slug must match %s", slugRE)
	}
	if slices.Contains(reserved, *slug) {
		return fail("--slug %q is reserved (it would collide with a web route)", *slug)
	}
	if utf8.RuneCountInString(*name) > 200 || strings.IndexFunc(*name, unicode.IsControl) >= 0 {
		return fail("--name must be at most 200 characters and contain no control characters")
	}
	if !localeRE.MatchString(*locale) {
		return fail("--locale: %q is not a BCP 47 locale such as en-US", *locale)
	}
	var doms []string
	for _, d := range strings.Split(*domains, ",") {
		d = strings.ToLower(strings.TrimSpace(d))
		if !domainRE.MatchString(d) {
			return fail("--domains: invalid domain %q (no @, e.g. example.edu)", d)
		}
		if !slices.Contains(doms, d) {
			doms = append(doms, d)
		}
	}
	if _, err := time.LoadLocation(*timezone); err != nil || *timezone == "Local" {
		return fail("--timezone: unknown IANA timezone %q", *timezone)
	}
	email, err := auth.NormalizeEmail(*adminEmail)
	if err != nil {
		return fail("--admin-email: %q is not a valid email address", *adminEmail)
	}

	db, err := open()
	if err != nil {
		fmt.Fprintln(stderr, "database:", err)
		return 1
	}
	m := newMailer(base + "/invite/")
	err = pg.WithPlatformTx(ctx, db, func(tx *sql.Tx) error {
		var id string
		if err := tx.QueryRowContext(ctx,
			`insert into universities (name, slug, allowed_email_domains, timezone, locale) values ($1, $2, $3, $4, $5) returning id`,
			strings.TrimSpace(*name), *slug, doms, *timezone, *locale).Scan(&id); err != nil {
			return pg.MapError(err)
		}
		if err := pg.SetUniversity(ctx, tx, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `select seed_us_grade_scale($1)`, id); err != nil {
			return err
		}
		_, err := tenant.CreateInvitation(ctx, tx, tenant.NewInvitation{UniversityID: id, Email: email, Role: authz.Admin}, m.Enqueue, base)
		return err
	})
	switch {
	case errors.Is(err, pg.ErrConflict):
		return fail("slug already exists: %s", *slug)
	case err != nil:
		fmt.Fprintln(stderr, "create-university:", err)
		return 1
	case m.URL() == "":
		fmt.Fprintln(stderr, "create-university: invitation url missing from mail text")
		return 1
	}
	fmt.Fprintf(stdout, "created university %s\nadmin invitation for %s (forward this URL):\n%s\n", *slug, email, m.URL())
	return 0
}

// seedDemo loads the embedded demo university (see internal/seed). Every exit-2 path returns before open().
func seedDemo(ctx context.Context, args []string, env func(string) string, open func() (*sql.DB, error), stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("seed-demo", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, seedHelp) }
	slug := fs.String("slug", "", "")
	reset := fs.Bool("reset", false, "")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "seed-demo: unexpected argument %q\n", fs.Arg(0))
		return 2
	}
	f, err := seed.Load(seed.Demo, *slug, time.Now().Year())
	if err != nil {
		fmt.Fprintln(stderr, "seed-demo:", err)
		return 2
	}
	switch {
	case !slugRE.MatchString(f.University.Slug):
		fmt.Fprintf(stderr, "seed-demo: --slug must match %s\n", slugRE)
		return 2
	case slices.Contains(reserved, f.University.Slug):
		fmt.Fprintf(stderr, "seed-demo: --slug %q is reserved (it would collide with a web route)\n", f.University.Slug)
		return 2
	}
	base := strings.TrimRight(env("UNICOURSE_BASE_URL"), "/")
	if base == "" {
		fmt.Fprintln(stderr, "seed-demo: UNICOURSE_BASE_URL is not set")
		return 2
	}
	root := env("UNICOURSE_FILES_DIR")
	if root == "" {
		fmt.Fprintln(stderr, "seed-demo: UNICOURSE_FILES_DIR is not set")
		return 2
	}
	dir, err := blob.New(root)
	if err != nil {
		fmt.Fprintln(stderr, "seed-demo:", err)
		return 1
	}
	db, err := open()
	if err != nil {
		fmt.Fprintln(stderr, "database:", err)
		return 1
	}
	rep, err := seed.Run(ctx, db, f, seed.Options{Reset: *reset, BaseURL: base, Files: dir})
	for _, w := range rep.Warnings {
		fmt.Fprintln(stderr, "seed-demo: warning:", w)
	}
	switch {
	case errors.Is(err, seed.ErrExists):
		fmt.Fprintf(stderr, "seed-demo: university %s already exists; rerun with --reset to replace it\n", f.University.Slug)
		return 2
	case err != nil:
		fmt.Fprintln(stderr, "seed-demo:", err)
		return 1
	}
	web := strings.TrimRight(env("UNICOURSE_WEB_ORIGIN"), "/")
	if web == "" {
		web = base
	}
	fmt.Fprintf(stdout, "seeded %s (%s)\nsign in at %s/sign-in with an email below; with UNICOURSE_SMTP_URL empty the API log prints the link\n\n", rep.Name, rep.Slug, web)
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ROLE\tNAME\tEMAIL\tSEES")
	for _, p := range rep.Personas {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", p.Role, p.Name, p.Email, p.Sees)
	}
	tw.Flush()
	return 0
}

// pruneFiles lists, or with --delete removes, committed blobs no materials row references (skipping
// blobs under an hour old: an upload between rename and commit looks orphaned, and .tmp/ files under
// a day old), and lists materials whose file is missing. Missing files are only reported; the rows stay.
func pruneFiles(ctx context.Context, args []string, env func(string) string, open func() (*sql.DB, error), stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("prune-files", flag.ContinueOnError)
	fs.SetOutput(stderr)
	del := fs.Bool("delete", false, "remove the orphan files instead of listing them")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "prune-files: unexpected argument %q\n", fs.Arg(0))
		return 2
	}
	root := env("UNICOURSE_FILES_DIR")
	if root == "" {
		fmt.Fprintln(stderr, "prune-files: UNICOURSE_FILES_DIR is not set")
		return 2
	}
	dir, err := blob.New(root)
	if err != nil {
		fmt.Fprintln(stderr, "prune-files:", err)
		return 1
	}
	db, err := open()
	if err != nil {
		fmt.Fprintln(stderr, "database:", err)
		return 1
	}
	rows := map[string]bool{}
	var unis []string
	err = pg.WithPlatformTx(ctx, db, func(tx *sql.Tx) error {
		r, err := tx.QueryContext(ctx, `select id::text from universities order by id`)
		if err != nil {
			return err
		}
		defer r.Close()
		for r.Next() {
			var id string
			if err := r.Scan(&id); err != nil {
				return err
			}
			unis = append(unis, id)
		}
		return r.Err()
	})
	// the owner role is under forced row level security: one tenant transaction per university
	for _, uni := range unis {
		if err != nil {
			break
		}
		err = pg.WithTx(ctx, db, uni, func(tx *sql.Tx) error {
			r, err := tx.QueryContext(ctx, `select id::text from materials`)
			if err != nil {
				return err
			}
			defer r.Close()
			for r.Next() {
				var id string
				if err := r.Scan(&id); err != nil {
					return err
				}
				rows[uni+"/"+id] = true
			}
			return r.Err()
		})
	}
	if err != nil {
		fmt.Fprintln(stderr, "prune-files:", err)
		return 1
	}
	blobs := map[string]bool{}
	var orphans []string
	err = dir.Walk(func(uni, id string, mod time.Time) error {
		k := uni + "/" + id
		blobs[k] = true
		if !rows[k] && time.Since(mod) >= time.Hour {
			orphans = append(orphans, k)
		}
		return nil
	})
	if err == nil {
		// a crashed upload leaves its temp file; a day is far longer than any upload takes
		err = dir.WalkTemp(func(uni, name string, mod time.Time) error {
			if time.Since(mod) >= 24*time.Hour {
				orphans = append(orphans, uni+"/.tmp/"+name)
			}
			return nil
		})
	}
	if err != nil {
		fmt.Fprintln(stderr, "prune-files:", err)
		return 1
	}
	var missing []string
	for k := range rows {
		if !blobs[k] {
			missing = append(missing, k)
		}
	}
	slices.Sort(orphans)
	slices.Sort(missing)
	code, removed := 0, 0
	for _, k := range orphans {
		if !*del {
			fmt.Fprintln(stdout, "orphan", k)
			continue
		}
		uni, id, _ := strings.Cut(k, "/")
		var err error
		if name, ok := strings.CutPrefix(id, ".tmp/"); ok {
			err = dir.RemoveTemp(uni, name)
		} else {
			err = dir.Remove(uni, id)
		}
		if err != nil {
			fmt.Fprintf(stderr, "prune-files: remove %s: %v\n", k, err)
			code = 1
			continue
		}
		removed++
		fmt.Fprintln(stdout, "removed", k)
	}
	for _, k := range missing {
		fmt.Fprintln(stdout, "missing", k)
	}
	plural := func(n int, what string) string {
		if n == 1 {
			return fmt.Sprintf("%d %s file", n, what)
		}
		return fmt.Sprintf("%d %s files", n, what)
	}
	found, tail := len(orphans), ""
	if *del {
		found, tail = removed, " removed"
	}
	fmt.Fprintf(stdout, "%s%s, %s\n", plural(found, "orphan"), tail, plural(len(missing), "missing"))
	return code
}

func main() {
	ctx := context.Background()
	url := os.Getenv("UNICOURSE_OWNER_DATABASE_URL")
	open := func() (*sql.DB, error) {
		if url == "" {
			return nil, errors.New("missing environment: UNICOURSE_OWNER_DATABASE_URL")
		}
		return pg.Open(ctx, url)
	}
	os.Exit(run(ctx, os.Args[1:], os.Getenv, open, os.Stdout, os.Stderr))
}
