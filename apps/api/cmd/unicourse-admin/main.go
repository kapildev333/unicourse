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
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/kapildev333/unicourse/apps/api/internal/auth"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/outbox"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

const usage = `usage: unicourse-admin <command> [flags]

commands:
  migrate               apply database migrations
  create-university     create a university and an admin invitation (see create-university -h)

environment: UNICOURSE_DATABASE_URL (all commands), UNICOURSE_BASE_URL (create-university)
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

func main() {
	ctx := context.Background()
	url := os.Getenv("UNICOURSE_DATABASE_URL")
	open := func() (*sql.DB, error) {
		if url == "" {
			return nil, errors.New("missing environment: UNICOURSE_DATABASE_URL")
		}
		return pg.Open(ctx, url)
	}
	os.Exit(run(ctx, os.Args[1:], os.Getenv, open, os.Stdout, os.Stderr))
}
