# UniCourse API

Go, `net/http` standard library only.

## Run

```
docker compose up -d
export UNICOURSE_DATABASE_URL=postgres://unicourse:unicourse@localhost:5432/unicourse?sslmode=disable
export UNICOURSE_TEST_DATABASE_URL=postgres://unicourse:unicourse@localhost:5432/unicourse_test?sslmode=disable
export UNICOURSE_OWNER_DATABASE_URL=postgres://unicourse_owner:unicourse_owner@localhost:5432/unicourse?sslmode=disable
export UNICOURSE_TEST_OWNER_DATABASE_URL=postgres://unicourse_owner:unicourse_owner@localhost:5432/unicourse_test?sslmode=disable
export UNICOURSE_BASE_URL=http://localhost:8080
export UNICOURSE_SESSION_SECRET=dev-secret-dev-secret-dev-secret-00
mkdir -p data/files
export UNICOURSE_FILES_DIR=$PWD/data/files
go run ./cmd/unicourse-admin migrate   # the API does not migrate
go run ./cmd/api
```

Compose runs Postgres 16 with superuser `postgres`/`postgres` and two roles, both
`NOSUPERUSER NOBYPASSRLS`: `unicourse_owner`/`unicourse_owner` owns the databases and every table and runs migrations;
`unicourse`/`unicourse` is the API role. The API role owns nothing, so it cannot disable row level security, drop
policies or create tables, and it has no TRUNCATE (TRUNCATE ignores row level security). `docker/initdb.sql` creates
the roles and the databases `unicourse`, `unicourse_test` and `unicourse_e2e` and only runs on a fresh volume.

Gates: `gofmt -l .`, `go vet ./...`, `go test ./... -count=1 -race` with `UNICOURSE_TEST_DATABASE_URL` and `UNICOURSE_TEST_OWNER_DATABASE_URL` set (unset `UNICOURSE_TEST_DATABASE_URL` skips the database tests; the owner URL unset fails them). DB tests serialise through an advisory lock, so `-p 1` is not needed. The API does not migrate: it exits at startup when the database is behind its migrations (run `unicourse-admin migrate`) or when `UNICOURSE_DATABASE_URL` uses the owner role, and `/readyz` reports unavailable while a migration is missing.

## Create the first university

```
export UNICOURSE_OWNER_DATABASE_URL=... UNICOURSE_BASE_URL=http://localhost:8080
go run ./cmd/unicourse-admin migrate
go run ./cmd/unicourse-admin create-university --name "Example" --slug example \
  --domains example.edu --timezone America/New_York --admin-email root@example.edu
```

No mail is sent. The command prints the admin invitation URL (`<base>/invite/<token>?email=...`); send it to
the admin, who signs in with that email and becomes the university's admin. `create-university -h` lists all flags. Usage output (no arguments, `-h`, unknown command) exits 2 and needs no database. Reserved slugs (`api`, `assets`, `invite`, `sign-in`, ...) are rejected.

## Load the demo

    export UNICOURSE_OWNER_DATABASE_URL=... UNICOURSE_BASE_URL=http://localhost:8080 UNICOURSE_FILES_DIR=$PWD/data/files
    go run ./cmd/unicourse-admin migrate
    go run ./cmd/unicourse-admin seed-demo            # --reset to replace it, --slug to load a second copy

Loads Northfield University from `internal/seed/demo.json` (edit the file to change the dataset; `<year>`,
`<year+1>` and `<domain>` are substituted) through the API's own handlers, so every write is validated and
audited like a user's. Prints one line per persona: sign in at `/sign-in` with that email and copy the magic
link from the API log (`UNICOURSE_SMTP_URL` empty). No mail is queued. `--reset` deletes only the university
with that slug and its files. For local databases only.

## Environment

Environment only. A missing required variable fails startup with the list of what is missing.

| Variable | Required | Meaning |
|---|---|---|
| `UNICOURSE_DATABASE_URL` | yes | Postgres connection string for the API role `unicourse` (not the owner) |
| `UNICOURSE_OWNER_DATABASE_URL` | CLI | Postgres connection string for the owner role `unicourse_owner`; used by `unicourse-admin` (all commands), never set it for the API process |
| `UNICOURSE_BASE_URL` | yes | public URL of the API (OAuth redirects, magic links) |
| `UNICOURSE_SESSION_SECRET` | yes | secret for session and token signing, at least 32 bytes |
| `UNICOURSE_FILES_DIR` | yes | existing writable directory for uploaded material files, laid out `<university_id>/<material_id>` (temp uploads in `<university_id>/.tmp/`); every API process must mount the same directory; back it up together with the database. Locally `apps/api/data/files` (gitignored) |
| `UNICOURSE_ADDR` | no | listen address, default `:8080` |
| `UNICOURSE_GOOGLE_CLIENT_ID`, `UNICOURSE_GOOGLE_CLIENT_SECRET` | no | Google sign-in |
| `UNICOURSE_MICROSOFT_CLIENT_ID`, `UNICOURSE_MICROSOFT_CLIENT_SECRET` | no | Microsoft sign-in |
| `UNICOURSE_SMTP_URL` | no | mail server; empty means the stdout mailer. Mail is queued in `outbox_mails` and sent by every API process |
| `UNICOURSE_WEB_ORIGIN` | no | CORS origin, development only (production serves the web app from the API binary) |
| `UNICOURSE_WEB_DIR` | no | built web app (`apps/web/dist`) to serve at `/`; empty means API only |
| `UNICOURSE_TRUSTED_PROXIES` | no | comma-separated CIDRs or IPs of the reverse proxies in front of the API; X-Forwarded-For is read only on connections from these (rightmost untrusted hop). Empty trusts none and keys rate limits on the connection address. List proxy addresses only, never client networks |
| `UNICOURSE_TEST_DATABASE_URL` | tests | database used by integration tests, as the API role |
| `UNICOURSE_TEST_OWNER_DATABASE_URL` | tests | the same database as the owner role (migrates and truncates) |

## Database roles and migrations

- Migrations run as `unicourse_owner` (`unicourse-admin migrate`). Tables and sequences a migration creates are granted to the API role automatically (default privileges set in `0004_role_split.sql`); anything else the API needs (a new schema, a function with revoked EXECUTE) is granted in that migration.
- Deploy order: migrate first, then roll the binaries. Every migration must work with the previous binary (expand, then contract in a later migration). A binary refuses to start, and `/readyz` fails, only when an embedded migration is missing from the database; a database ahead of the binary is fine.
- Production, once, as the database administrator:

```sql
create role unicourse_owner login password '<secret>' nosuperuser nobypassrls nocreatedb;
create role unicourse login password '<secret>' nosuperuser nobypassrls nocreatedb;
create database unicourse owner unicourse_owner;
```

  then `UNICOURSE_OWNER_DATABASE_URL=... unicourse-admin migrate`. The API process gets only `UNICOURSE_DATABASE_URL`; the owner credential is never in its environment. The role names are fixed in `0004`.
- An existing local volume: `docker compose down -v && docker compose up -d` (loses data), or keep the data and, as `postgres` (`docker compose exec -T postgres psql -U postgres -v ON_ERROR_STOP=1`):

```sql
create role unicourse_owner login password 'unicourse_owner' nosuperuser nobypassrls nocreatedb; -- skip this line if the role already exists
\c unicourse
reassign owned by unicourse to unicourse_owner;
\c unicourse_test
reassign owned by unicourse to unicourse_owner;
\c unicourse_e2e
reassign owned by unicourse to unicourse_owner;
```

  then `unicourse-admin migrate` with the owner URL against each of the three databases.

## Files

Material files live under `UNICOURSE_FILES_DIR/<university_id>/<material_id>`; uploads are first written to `<university_id>/.tmp/`. Only `internal/blob` touches the directory.

- The database row is the source of truth: a file without a row is garbage, a row without a file is a logged 500 on download.
- Restore the database and the files directory together.
- `UNICOURSE_OWNER_DATABASE_URL=... UNICOURSE_FILES_DIR=... unicourse-admin prune-files` lists files no material references and materials whose file is missing (rows are never touched); `--delete` removes the orphan files. Files younger than one hour and `.tmp/` are skipped; `.tmp/` files older than a day can be deleted by hand.
- One volume is the ceiling (D88): all API processes must share the directory. Moving to an S3-compatible store means replacing `internal/blob` behind the same methods.

## Run the whole app from one binary

```
cd apps/web && npm run build
cd ../api && UNICOURSE_WEB_DIR=../web/dist go run ./cmd/api
```

The API serves `apps/web/dist` under `/`: real files as they are (`/assets/*` cached immutably), any other path without a file extension as `index.html` (client-side routes), and `/api/*` or a missing file with an extension as the JSON 404 envelope. Startup fails if the directory has no `index.html`.
