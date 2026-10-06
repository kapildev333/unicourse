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
	"strings"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/auth"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
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
	domainRE = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$`)
)

// captureMailer remembers the invitation URL instead of sending anything.
type captureMailer struct{ prefix, url string }

func (m *captureMailer) Send(_ context.Context, _, _, text string) error {
	for _, l := range strings.Split(text, "\n") {
		if strings.HasPrefix(l, m.prefix) {
			m.url = l
		}
	}
	return nil
}

func run(ctx context.Context, args []string, env func(string) string, db *sql.DB, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprint(stderr, usage)
		return 2
	}
	switch args[0] {
	case "migrate":
		if err := pg.Migrate(ctx, db); err != nil {
			fmt.Fprintln(stderr, "migrate:", err)
			return 1
		}
		fmt.Fprintln(stdout, "migrations applied")
		return 0
	case "create-university":
		return createUniversity(ctx, args[1:], env, db, stdout, stderr)
	}
	fmt.Fprintf(stderr, "unknown command %q\n%s", args[0], usage)
	return 2
}

func createUniversity(ctx context.Context, args []string, env func(string) string, db *sql.DB, stdout, stderr io.Writer) int {
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
	var doms []string
	for _, d := range strings.Split(*domains, ",") {
		d = strings.ToLower(strings.TrimSpace(d))
		if !domainRE.MatchString(d) {
			return fail("--domains: invalid domain %q (no @, e.g. example.edu)", d)
		}
		doms = append(doms, d)
	}
	if _, err := time.LoadLocation(*timezone); err != nil {
		return fail("--timezone: unknown IANA timezone %q", *timezone)
	}
	email, err := auth.NormalizeEmail(*adminEmail)
	if err != nil {
		return fail("--admin-email: %q is not a valid email address", *adminEmail)
	}

	m := &captureMailer{prefix: base + "/invite/"}
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
		_, err := tenant.CreateInvitation(ctx, tx, tenant.NewInvitation{UniversityID: id, Email: email, Role: authz.Admin}, m, base)
		return err
	})
	switch {
	case errors.Is(err, pg.ErrConflict):
		return fail("slug already exists: %s", *slug)
	case err != nil:
		fmt.Fprintln(stderr, "create-university:", err)
		return 1
	case m.url == "":
		fmt.Fprintln(stderr, "create-university: invitation url missing from mail text")
		return 1
	}
	fmt.Fprintf(stdout, "created university %s\nadmin invitation for %s (forward this URL):\n%s\n", *slug, email, m.url)
	return 0
}

func main() {
	url := os.Getenv("UNICOURSE_DATABASE_URL")
	if url == "" {
		fmt.Fprintln(os.Stderr, "missing environment: UNICOURSE_DATABASE_URL")
		os.Exit(2)
	}
	ctx := context.Background()
	db, err := pg.Open(ctx, url)
	if err != nil {
		fmt.Fprintln(os.Stderr, "database:", err)
		os.Exit(1)
	}
	code := run(ctx, os.Args[1:], os.Getenv, db, os.Stdout, os.Stderr)
	db.Close()
	os.Exit(code)
}
