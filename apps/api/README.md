# UniCourse API

Go, `net/http` standard library only.

## Run

```
docker compose up -d
export UNICOURSE_DATABASE_URL=postgres://unicourse:unicourse@localhost:5432/unicourse?sslmode=disable
export UNICOURSE_TEST_DATABASE_URL=postgres://unicourse:unicourse@localhost:5432/unicourse_test?sslmode=disable
export UNICOURSE_BASE_URL=http://localhost:8080
export UNICOURSE_SESSION_SECRET=dev-secret-dev-secret-dev-secret-00
go run ./cmd/api
```

Compose runs Postgres 16 with superuser `postgres`/`postgres` and an app role `unicourse`/`unicourse`
(`NOSUPERUSER NOBYPASSRLS`, so row-level security applies). `docker/initdb.sql` creates the databases
`unicourse` and `unicourse_test` and only runs on a fresh volume.

Gates: `gofmt -l .`, `go vet ./...`, `go test ./... -count=1 -race` with `UNICOURSE_TEST_DATABASE_URL` set (unset skips the database tests). DB tests serialise through an advisory lock, so `-p 1` is not needed. The API applies migrations at startup.

## Create the first university

```
export UNICOURSE_DATABASE_URL=... UNICOURSE_BASE_URL=http://localhost:8080
go run ./cmd/unicourse-admin migrate
go run ./cmd/unicourse-admin create-university --name "Example" --slug example \
  --domains example.edu --timezone America/New_York --admin-email root@example.edu
```

No mail is sent. The command prints the admin invitation URL (`<base>/invite/<token>?email=...`); send it to
the admin, who signs in with that email and becomes the university's admin. `create-university -h` lists all flags.

## Environment

Environment only. A missing required variable fails startup with the list of what is missing.

| Variable | Required | Meaning |
|---|---|---|
| `UNICOURSE_DATABASE_URL` | yes | Postgres connection string |
| `UNICOURSE_BASE_URL` | yes | public URL of the API (OAuth redirects, magic links) |
| `UNICOURSE_SESSION_SECRET` | yes | secret for session and token signing, at least 32 bytes |
| `UNICOURSE_ADDR` | no | listen address, default `:8080` |
| `UNICOURSE_GOOGLE_CLIENT_ID`, `UNICOURSE_GOOGLE_CLIENT_SECRET` | no | Google sign-in |
| `UNICOURSE_MICROSOFT_CLIENT_ID`, `UNICOURSE_MICROSOFT_CLIENT_SECRET` | no | Microsoft sign-in |
| `UNICOURSE_SMTP_URL` | no | mail server; empty means the stdout mailer |
| `UNICOURSE_WEB_ORIGIN` | no | CORS origin, development only (production serves the web app from the API binary) |
| `UNICOURSE_TEST_DATABASE_URL` | tests | database used by integration tests |
