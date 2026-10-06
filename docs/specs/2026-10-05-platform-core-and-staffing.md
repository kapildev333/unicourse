# Spec 1: Platform core and course staffing

Status: draft for review · Date: 2026-10-05 · Scope: sub-projects 0 (platform core) and 1 (courses, sections, staffing, rosters)

## 1. Intent

UniCourse is a multi-tenant platform for universities to host courses, manage sections, and link professors, TAs and students. The product vision covers course hosting, scheduling and integrations. This spec covers the first sellable slice: a department runs its term staffing and rosters in UniCourse instead of a spreadsheet.

**Success criterion.** A department admin at one university signs in with their campus Google or Microsoft account, creates a term, courses and sections, assigns an instructor and TAs to each section, imports a student roster from CSV, and every assigned person signs in and sees their sections. A second university's data is invisible to the first; an automated test asserts it.

**Wedge.** Incumbent LMSs (Canvas, Moodle, Blackboard) hold content and grades; TA assignment, section staffing and roster handling live in departmental spreadsheets. UniCourse starts there and sits beside the LMS. Course hosting (Spec 2) and scheduling (Spec 3) build on this data model.

## 2. Out of scope

Content hosting, assignments, grading, announcements · rooms, timeslots, enrollment caps, waitlists · SAML/Shibboleth · LTI · SIS sync (Banner, PeopleSoft) · billing · AI agents · mobile apps · soft delete (hard delete plus audit log; upgrade when a customer asks).

## 3. Stack and constraints

| | Decision |
|---|---|
| API | Go 1.27, `net/http` standard library. `http.ServeMux` with method and wildcard patterns for routing. No framework, router library, ORM, DI container or code generator. |
| Allowed third-party Go modules | `github.com/jackc/pgx/v5` (Postgres driver, via `database/sql`), `golang.org/x/oauth2` and `github.com/coreos/go-oidc/v3` (OIDC), `golang.org/x/crypto` (token hashing). Anything else is a PLAN.md decision. |
| Database | PostgreSQL 16. Hand-written SQL. Row-level security enforces tenancy. Migrations are numbered `.sql` files embedded with `embed` and applied by our own runner. |
| Web | React 18, Vite, TypeScript strict, TanStack Router (file-based) and TanStack Query, Tailwind. npm. |
| Contract | `apps/api/openapi.yaml`, hand-maintained. `apps/web/src/api/` client types are generated from it (`openapi-typescript`, dev dependency). A CI check asserts every route registered in Go exists in the contract and vice versa. |
| Tests | `go test` with the `testing` package only; integration tests run against real Postgres from `docker-compose.yml`. Web: Vitest + Testing Library for components; one Playwright end-to-end flow. |
| Local run | `docker compose up` (Postgres), `go run ./cmd/api`, `npm run dev`. |

Naming conventions are in `CLAUDE.md` and are part of this spec.

## 4. Domain model

All tables except `users` and `universities` carry `university_id` and are tenant-scoped.

| Entity | Fields (beyond `id uuid`, `created_at`, `updated_at`) | Rules |
|---|---|---|
| `universities` | `name`, `slug` (unique, used in URLs), `allowed_email_domains text[]`, `timezone` (IANA, e.g. `America/New_York`), `locale` (BCP 47, default `en-US`) | Created by a platform operator via CLI command, not through the API, in this spec. Timezone and locale are stored now because every later spec (scheduling, deadlines) needs them and backfilling is a migration against live customers. |
| `users` | `email` (unique, lowercased), `display_name` | Global identity. A user may belong to several universities (adjunct at two campuses). |
| `memberships` | `university_id`, `user_id`, `role` (`admin`, `dept_admin`, `member`), `status` (`invited`, `active`) | Unique on (`university_id`, `user_id`). `dept_admin` scope is the set of `department_admins` rows. |
| `department_admins` | `university_id`, `department_id`, `user_id` | Which departments a `dept_admin` manages. |
| `departments` | `university_id`, `name`, `code` (unique per university, e.g. `CS`) | |
| `terms` | `university_id`, `name` (e.g. `Fall 2026`), `starts_on`, `ends_on` | `starts_on < ends_on`. |
| `courses` | `university_id`, `department_id`, `code` (unique per university, e.g. `CS101`), `title` | Catalog entry, term-independent. |
| `sections` | `university_id`, `course_id`, `term_id`, `number` (e.g. `001`) | Unique on (`course_id`, `term_id`, `number`). |
| `section_staff` | `university_id`, `section_id`, `user_id`, `role` (`instructor`, `ta`) | Unique on (`section_id`, `user_id`). A section may have zero or more instructors and TAs. |
| `enrollments` | `university_id`, `section_id`, `user_id` | Unique on (`section_id`, `user_id`). Students are `users` with a `member` membership. |
| `invitations` | `university_id`, `email`, `role`, `department_ids uuid[]`, `token_hash`, `expires_at`, `accepted_at` | Token is random 32 bytes, stored hashed. Expires after 14 days. |
| `sessions` | `user_id`, `token_hash`, `expires_at`, `last_seen_at` | Cookie session, 30 days sliding. |
| `magic_links` | `email`, `token_hash`, `expires_at`, `used_at` | 15-minute expiry, single use. |
| `audit_log` | `university_id`, `actor_user_id`, `action`, `entity_type`, `entity_id`, `before jsonb`, `after jsonb`, `at` | Append-only. Written in the same transaction as the change. |

Every foreign key is declared. Uniqueness and check rules above are database constraints, not application checks.

### Tenancy enforcement

Every tenant-scoped table has an RLS policy `university_id = current_setting('app.university_id')::uuid`. The API's database role is not a superuser and does not bypass RLS. `pg.WithTx(ctx, universityID, fn)` opens a transaction, runs `SET LOCAL app.university_id = $1`, runs `fn`, commits. All tenant-scoped queries go through `WithTx`; there is no other way to obtain a connection in the domain packages. The platform CLI uses a separate role for `universities` administration.

## 5. Authentication and authorization

### Sign-in

1. **OIDC** with Google and Microsoft. The sign-in page asks for the university slug or email; the API picks the university by `allowed_email_domains` and redirects to the provider. On callback, the API verifies the ID token (`go-oidc`), lowercases the email, and requires that the email domain is in the university's `allowed_email_domains` or that a pending invitation exists for that exact email.
2. **Magic link** by email for users whose campus does not use Google or Microsoft. Same domain/invitation rule. Email sending is behind a `Mailer` interface with an SMTP implementation and a log-to-stdout implementation for development.
3. On first successful sign-in, the user row is created if absent, pending invitations for that email are accepted (membership becomes `active`, department admin rows are created), and a session cookie is set: `HttpOnly`, `Secure`, `SameSite=Lax`.

### Roles

| Role | Can |
|---|---|
| `admin` | Everything inside their university: departments, terms, invitations, memberships, all courses, sections, staff, enrollments, audit log. |
| `dept_admin` | Courses, sections, staff and enrollments for departments listed in `department_admins`. Read terms. Invite `member`s. |
| `member` with `section_staff.role = instructor` | Read their sections. Add and remove `ta` staff on their sections. Read rosters of their sections. |
| `member` with `section_staff.role = ta` or an enrollment | Read their sections and, for TAs, the roster. |
| any `member` | Read their own memberships and "my sections". |

Authorization lives in one package, `internal/authz`, exposing predicates such as `CanManageSection(ctx, actor, sectionID)`. Handlers call a predicate before any write; predicates query within the tenant transaction. RLS underneath guarantees that a missed predicate can only leak within a university, never across.

## 6. API

Base path `/api/v1`. JSON in and out, camelCase fields. Session cookie authenticates; every tenant-scoped route is under `/api/v1/universities/{slug}/...` and the middleware resolves the slug, checks the caller's membership, and opens `WithTx` for the handler.

| Method and path | Who | Notes |
|---|---|---|
| `POST /auth/oidc/{provider}/start`, `GET /auth/oidc/{provider}/callback` | anyone | Google, Microsoft |
| `POST /auth/magic-link`, `GET /auth/magic-link/{token}` | anyone | |
| `POST /auth/sign-out` | signed in | |
| `GET /me` | signed in | user plus memberships |
| `GET /universities/{slug}/me/sections` | member | sections where the caller is staff or enrolled, grouped by term |
| `GET,POST /universities/{slug}/departments`, `PATCH,DELETE .../departments/{id}` | admin | |
| `GET,POST .../terms`, `PATCH,DELETE .../terms/{id}` | admin (write), member (read) | |
| `GET,POST .../invitations`, `DELETE .../invitations/{id}` | admin; dept_admin for `member` role | |
| `GET .../memberships`, `PATCH,DELETE .../memberships/{id}` | admin | role changes, removal |
| `GET,POST .../courses`, `PATCH,DELETE .../courses/{id}` | admin, dept_admin (own departments) | |
| `GET,POST .../sections`, `GET,PATCH,DELETE .../sections/{id}` | admin, dept_admin; read for staff and enrolled | filter by `termId`, `courseId` |
| `GET,POST .../sections/{id}/staff`, `DELETE .../sections/{id}/staff/{userId}` | admin, dept_admin; instructor for `ta` rows | staff member must be an active `member` or an invited email (creates invitation) |
| `GET .../sections/{id}/enrollments`, `DELETE .../sections/{id}/enrollments/{userId}` | admin, dept_admin; read for staff | |
| `POST .../sections/{id}/enrollments/import/preview`, `POST .../sections/{id}/enrollments/import` | admin, dept_admin | see CSV import |
| `POST .../courses/import/preview`, `POST .../courses/import` | admin, dept_admin (own departments) | columns `departmentCode`, `code`, `title`; same preview-then-commit |
| `POST .../sections/import/preview`, `POST .../sections/import` | admin, dept_admin (own departments) | columns `courseCode`, `termName`, `number`, optional `instructorEmails`, `taEmails` (semicolon-separated); same preview-then-commit |
| `GET .../audit-log` | admin | filter by `entityType`, `entityId`, `actorUserId`, `from`, `to`; paginated |

Lists are paginated with `?limit=` (default 50, max 200) and `?cursor=`; responses are `{ items, nextCursor }`.

### Errors

One envelope for every non-2xx response:

```json
{ "error": { "code": "validation_failed", "message": "2 fields are invalid", "details": [ { "field": "endsOn", "message": "must be after startsOn" } ] } }
```

Codes: `unauthenticated` (401), `forbidden` (403), `not_found` (404), `conflict` (409, unique violations mapped from Postgres error 23505), `validation_failed` (400, lists every bad field), `rate_limited` (429, magic-link and sign-in endpoints), `internal` (500, message is generic, details empty, the real error is logged with a request id that is also returned in the `X-Request-Id` header).

### CSV import (rosters, courses, sections)

Onboarding a university means loading thousands of courses and sections, not typing them. One import mechanism in `internal/csvimport`, parameterised by a row handler, serves all three endpoints. Two steps so an admin never partially applies a bad file.

1. `preview`: multipart upload, max 5 MB, header row required, columns per endpoint as listed in the API table. Returns per-row results: `add`, `exists` (already present, skipped), `invite` (email not yet a member; an invitation will be created), or `error` with a reason (bad email, domain not allowed, unknown course/term/department code, duplicate row). Nothing is written.
2. `import`: same file plus the preview's `checksum`. If the checksum differs or any row is `error`, the request fails with `validation_failed` and the row list. Otherwise every add and invitation happens in one transaction, with one `audit_log` row per created entity.

## 7. Web app

Four screens, in `apps/web/src/features/`:

| Feature | Screens | Who |
|---|---|---|
| `auth` | Sign in (university or email, provider buttons, magic-link form), magic-link sent, accept invitation | everyone |
| `admin` | Departments, terms, invitations and members with role editing, audit log viewer | admin |
| `courses` | Course list per department; section list per term with instructor and TA chips; section detail with staff editor and roster (table, remove, CSV import with preview-then-commit and per-row errors) | admin, dept_admin, instructor (staff editor limited to TAs) |
| `my-sections` | Sections grouped by term with role badge; links into section detail (read-only for TAs and students) | every member |

Routing: `/sign-in`, `/invite/{token}`, `/{slug}/me`, `/{slug}/admin/*`, `/{slug}/courses/*`, `/{slug}/sections/{id}`. The API client is generated from `openapi.yaml`; no hand-written fetch calls outside `src/api/`. Query keys are per resource and invalidated on write. Forms show field errors from `details` next to the field and the `message` at the top.

Accessibility baseline: every interactive element keyboard reachable, labels on all inputs, focus moves to the first error on failed submit, colour never the only signal (role badges carry text).

## 8. Internal layout

```
apps/api/
├── cmd/api/main.go          config from env, open db, build mux, listen; nothing else
├── cmd/unicourse-admin/     CLI: create-university, run-migrations
├── internal/
│   ├── httpx/               mux registration, middleware (request id, logging, session, tenant), error envelope, JSON decode/encode, pagination
│   ├── pg/                  Open, WithTx (RLS), migrations runner, error mapping (23505 → conflict)
│   ├── auth/                OIDC, magic links, sessions, invitations, Mailer
│   ├── authz/               role predicates
│   ├── tenant/              universities (read), departments, terms, memberships + handlers
│   ├── course/              courses, sections, staff, enrollments + handlers, including the three import endpoints
│   ├── csvimport/           generic preview/commit engine: parse, per-row classify, checksum, all-or-nothing apply
│   ├── audit/               Write(tx, entry), List
│   └── testdb/              test-only: fresh schema per test package, two-university fixtures
├── migrations/              0001_init.sql, …
├── openapi.yaml
├── docker-compose.yml       postgres:16
└── go.mod                   module github.com/kapildev333/unicourse/apps/api

apps/web/
├── src/
│   ├── api/                 generated types, fetch wrapper (credentials, error envelope → typed ApiError)
│   ├── routes/              TanStack file routes
│   ├── features/            auth/, admin/, courses/, my-sections/
│   ├── components/          shared UI only (button, table, dialog, form field)
│   └── lib/                 small utilities
├── e2e/                     Playwright
└── package.json
```

Handlers live inside their domain package (`course/handler.go`) so a feature is readable in one directory; domain packages may import `net/http`. Each domain package exposes `Register(mux *http.ServeMux, deps)`.

## 9. Configuration

Environment only, `UNICOURSE_` prefix: `DATABASE_URL`, `ADDR` (default `:8080`), `BASE_URL`, `SESSION_SECRET`, `GOOGLE_CLIENT_ID/SECRET`, `MICROSOFT_CLIENT_ID/SECRET`, `SMTP_URL` (empty → stdout mailer), `WEB_ORIGIN` (CORS, development only; production serves the built web app from the API binary under `/`). Missing required variables fail at startup with a list of what is missing.

## 10. Scale and multi-university fit

Sizing assumption: a large university has ~50k students, ~5k sections per term, ~200k enrollments per term; one hundred such universities over five years is ~100M enrollment rows and ~1B audit rows. One PostgreSQL instance with the indexes implied by the unique constraints handles the enrollment volume; audit is the only table that needs partitioning, deferred with a trigger above.

What makes the design scale horizontally without redesign:

- The API is stateless: sessions, magic links and OIDC state live in Postgres, so any number of API processes run behind a load balancer.
- Every tenant row carries `university_id`, so moving a university to its own database or region is a copy filtered by one column, never a schema change.
- `WithTx` sets `app.university_id` per transaction, so connection pooling (pgx pool, later PgBouncer in transaction mode) is safe.
- Imports are bounded per request (5 MB) and run in one transaction; a 50k-row roster is ~3 MB and imports in seconds on commodity Postgres. Larger loads go through the SIS sync deferred to a later spec.

What makes it fit any university, not one:

- Terms are rows, not an enum: semesters, quarters, trimesters, summer sessions and non-standard terms all work.
- Course and department codes are free-form strings unique per university; numbering schemes are the university's.
- Multiple email domains per university, plus invitations for people outside those domains (adjuncts, visiting TAs).
- Timezone and locale per university from day one.
- Any number of instructors and TAs per section; staff can hold roles in several universities under one identity.
- Nothing in the data model or UI copy assumes a country, a grading scale, a calendar or a language beyond the default locale.

Structured logs with `log/slog`, JSON in production, one line per request (method, path, status, duration, request id, user id, university id). Audit log is the business-level trail. No metrics or tracing in this spec; `httpx` exposes `/healthz` (process up) and `/readyz` (database reachable).

## 12. Testing

| Layer | What | How |
|---|---|---|
| `pg` | migrations apply from empty; RLS blocks cross-tenant reads and writes | real Postgres from docker-compose; a test creates two universities and asserts that a transaction scoped to one sees nothing of the other, for every tenant table |
| `authz` | every predicate × every role | table-driven `go test` |
| domain packages | handlers through `httptest` against real Postgres: success, validation, conflict, forbidden | one `_test.go` per handler file; shared fixtures in `internal/testdb` |
| CSV import | preview classification, checksum mismatch, all-or-nothing | table-driven with small CSV fixtures |
| contract | routes in Go == routes in `openapi.yaml` | `go test` that walks the mux patterns and the YAML |
| web | each feature's main component: render, validation display, error envelope display | Vitest + Testing Library |
| end to end | admin creates term, course, section; assigns instructor and TA; imports roster; TA signs in (magic link via stdout mailer) and sees the section and roster | Playwright against `docker compose` + API + web |

Gates for every lane: `go vet ./...`, `go test ./...`, `npm run lint`, `npm run typecheck`, `npm test`. Playwright runs in CI and before each phase is marked done.

## 13. Deferred, with triggers

| Deferred | Add when |
|---|---|
| SAML/Shibboleth | first university whose IdP is not Google or Microsoft |
| SIS sync | first signed university grants API access |
| Soft delete | a customer asks to restore something |
| Platform admin UI | more than ~10 universities, or a non-engineer onboards them |
| Metrics/tracing | first production incident the request log cannot explain |
| Events table for AI agents | Spec for AI agents begins; `audit_log` is already the seed |
| Cross-listed courses (one course owned by several departments) | first university that cross-lists; change `courses.department_id` to a `course_departments` join with a primary flag — additive migration |
| Section hierarchy (lecture with lab/discussion sections) | Spec 3 (scheduling); add nullable `sections.parent_section_id` — additive migration |
| Custom roles per university (registrar, coordinator, chair) | second university whose org chart does not fit `admin`/`dept_admin`/`member`; `authz` is the single place to extend |
| Per-university branding (logo, colours, display name) | first customer asks; `universities` gets a `branding jsonb` column |
| Regional deployments (EU data residency) | first EU university; the stack is stateless and single-database, so a region is a second deployment of the same binaries and a second database, nothing shared |
| Audit log partitioning by month | `audit_log` passes ~100M rows or admin queries slow past 1 s |

## 14. Decisions needing the owner's call

None open. Decisions made in this spec and the hint behind each: Go stdlib (owner's constraint), separate `apps/api` and `apps/web` (owner's layout), handlers inside domain packages (proposed, no objection), hard delete with audit (proposed, no objection).
