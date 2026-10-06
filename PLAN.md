# UniCourse - orchestration plan

Spec: `docs/specs/2026-10-05-platform-core-and-staffing.md` · Implementation plan (task detail, TDD steps): `docs/plans/2026-10-05-platform-core-and-staffing.md`

## Decisions
| # | Decision | Status |
|---|----------|--------|
| D1 | Stack and constraints as spec §3: Go 1.27 stdlib `net/http`, pgx/x-oauth2/go-oidc/x-crypto only, PostgreSQL 16 with RLS, React 18 + Vite + TS strict + TanStack + Tailwind, npm | Agreed (owner, spec 2026-10-05) |
| D2 | Contract file is `apps/api/openapi.json`, not YAML (stdlib has no YAML parser; contract test reads it) | Agreed (owner, PR #4) |
| D3 | Web runtime/dev deps beyond the framework: `openapi-fetch`, `openapi-typescript` only (plan Task 3) | Orchestrator 2026-10-05, owner to confirm |
| D4 | Postgres roles: the image superuser stays `postgres`; the API/test role `unicourse` is `NOSUPERUSER NOBYPASSRLS` and owns `unicourse` and `unicourse_test` (superusers bypass RLS, so the plan's `POSTGRES_USER=unicourse` would void the tenancy test). Compose init script and a CI step create it | Orchestrator 2026-10-05, owner to confirm |
| D5 | (withdrawn) The app READMEs are tracked, not gitignored - the orchestrator misread a command output. Lanes write run instructions in `apps/api/README.md` and `apps/web/README.md` as the plan says | Withdrawn 2026-10-06 |
| D6 | Lane logs are written in the lane's own worktree (`.claude/lanes/<LANE-ID>.md`) and land with the lane's PR; lanes read each other's via `.worktrees/*/.claude/lanes/` | Orchestrator 2026-10-05 |
| D7 | A sub-agent has ~100k tokens, so plan tasks 5, 6, 8, 10, 11 are built as sequential sub-lanes (`<LANE-ID>a`, `b`, …) on the same branch, cut by that task's designer | Orchestrator 2026-10-05 |
| D8 | Owner is away overnight 2026-10-05 and delegated every decision ("don't wait for my decisions"); the orchestrator decides, records it in this table as "owner to confirm", and never changes usage limits, on-demand usage or credits | Agreed (owner, 2026-10-05) |
| D9 | Cross-university reads under RLS: three extra SELECT-only policies (`memberships_self`, `department_admins_self` keyed on `app.user_id`; `invitations_by_email` keyed on `app.email`), set only by `pg.WithUserTx`, which never sets `app.university_id`; all writes stay under `tenant_isolation`. Every policy wraps the setting in `nullif(..., '')` (a pooled connection returns `''`, not NULL). Design: `docs/design/P1-L1.md` §3 | Orchestrator 2026-10-06 on the P1-L1 designer's proposal, owner to confirm |
| D10 | Pending staff assignments for not-yet-members live in a tenant table `pending_section_staff (invitation_id, section_id, role)` with cascading FKs, applied by `auth.SignIn` (not a jsonb column: FKs are declared, a deleted section cannot break a later sign-in) | Orchestrator 2026-10-06, owner to confirm |
| D11 | Magic links are not consumed on GET (mail scanners such as Outlook Safe Links open every link): `GET /auth/magic-link/{token}` returns a small API-rendered confirm page whose button POSTs to the same URL; `POST /auth/magic-link/{token}` consumes and redirects. One route added to spec §6 | Orchestrator 2026-10-06, owner to confirm |
| D12 | OIDC state/nonce/PKCE live in an HMAC-signed cookie, not a Postgres table (deviation from spec §10; the API stays stateless). `UNICOURSE_SESSION_SECRET` must be at least 32 bytes | Orchestrator 2026-10-06, owner to confirm |
| D13 | Sign-in allow rule is checked against all universities (allowed domain anywhere, or a pending invitation for that exact email); a session grants nothing until tenant middleware finds an active membership. Magic-link rate limit: 5 per email per hour, 300 per IP per hour (raised from the plan's 20 because a campus shares one NAT address), in memory | Orchestrator 2026-10-06, owner to confirm |
| D14 | `RegisterAll` cannot live in `httpx` (import cycle with the domain packages): route wiring and the contract test live in a new package `internal/app`; `httpx.Router` answers unknown routes with the 404 envelope | Orchestrator 2026-10-06 |
| D15 | Deleting a department, term or course that is still referenced is a 409 `conflict` (`pg.MapDeleteError`; FKs `on delete restrict`); staff, enrollments and pending staff are parts of a section and cascade with it | Orchestrator 2026-10-06, owner to confirm |
| D16 | Contract conventions (`docs/design/P0-L4.md` §2, §6): path keys relative to `/api/v1`; no field is ever `null` (optional = omitted, arrays always present); `/healthz` and `/readyz` are outside the contract; sign-in callbacks always answer 302 and report failures as `/sign-in?error=<code>`; audit log is newest first | Orchestrator 2026-10-06, owner to confirm |

## Method
- Orchestrator launches lanes, verifies every landing (re-run gates, then real Postgres / HTTP / browser), maintains this file.
- One directory = one owner at a time. Hot spots are single-lane: `apps/api/internal/httpx/routes.go`, `apps/api/cmd/api/main.go`, `apps/api/migrations/`, `apps/api/openapi.json`, `.github/workflows/ci.yml`.
- One integration suite at a time, orchestrator included, until runs get an ephemeral database (single local Postgres from `apps/api/docker-compose.yml`).
- Test-driven: every worker step is failing test → implementation → pass → commit, as written in the implementation plan. Reviewer re-runs gates; orchestrator re-runs them again before the PR.
- Order per lane: designer (kind D) → worker → reviewer → orchestrator verification → PR → green `ci` → squash-merge.
- Agents escalate only decisions that belong in the table above.

## Phase 0 - Foundation
| Lane | Model | Scope (disjoint) | Status |
|------|-------|------------------|--------|
| P0-L1 - API skeleton and `httpx` (Task 1, W) | Sonnet 5.5 | `apps/api/{go.mod,cmd/api,internal/config,internal/httpx,docker-compose.yml,docker/}` | **done** - [#6](https://github.com/kapildev333/unicourse/pull/6). Evidence: gates re-run green with `-race`; built binary over real HTTP: `/healthz` 200, `/readyz` 503, unknown route and wrong method return the 404 envelope, `X-Request-Id` echoed or regenerated, SIGTERM exits 0; compose Postgres up with role `unicourse` rolsuper=f rolbypassrls=f |
| P0-L2 - `pg`, schema, RLS, `testdb`, `audit` (Task 2, W) | Sonnet 5.5 | `apps/api/{migrations,internal/pg,internal/testdb,internal/audit}`, `cmd/api/main.go`, `ci.yml` api job | worker running |
| P0-L3 - Web skeleton and API client (Task 3, W) | Sonnet 5.5 | `apps/web/` | **done** - [#7](https://github.com/kapildev333/unicourse/pull/7). Evidence: `npm ci`, lint, typecheck, 10 tests, build re-run green from a clean install with no generated-file drift; headless Chrome against `vite preview` renders all six routes |
| P0-L4 - `openapi.json` and contract test (Task 4, D) | Opus 5.5 design, Sonnet 5.5 build | `docs/design/P0-L4.md`, `apps/api/openapi.json`, `apps/api/internal/httpx/routes.go`, `apps/api/internal/app`, `apps/web/src/api/types.ts`, `ci.yml` web job | design done (42 operations, +1 by D11); worker running |

## Phase 1 - Platform core
| Lane | Model | Scope (disjoint) | Status |
|------|-------|------------------|--------|
| P1-L1 - Authentication (Task 5, D) | Opus 5.5 design, Sonnet 5.5 build | `docs/design/P1-L1.md`, `apps/api/internal/auth` | design done; build as P1-L1a (primitives), b (`SignIn`, OIDC), c (handlers, wiring) after P0-L2 and P0-L4 land |
| P1-L2 - Tenant scope, `authz`, departments, terms, memberships, invitations, audit route (Task 6, D) | Opus 5.5 design, Sonnet 5.5 build | `docs/design/P1-L2.md`, `apps/api/internal/{authz,tenant}` | designer running |
| P1-L3 - `unicourse-admin` CLI (Task 7, W) | Sonnet 5.5 | `apps/api/cmd/unicourse-admin` | not started |
| P1-L4 - Web `auth` and `admin` features (Task 8, D) | Opus 5.5 design, Sonnet 5.5 build | `docs/design/P1-L4.md`, `apps/web/src/{features/auth,features/admin,routes}` | designer running |

## Phase 2 - Courses, sections, staffing, rosters
| Lane | Model | Scope (disjoint) | Status |
|------|-------|------------------|--------|
| P2-L1 - `csvimport` engine (Task 9, W) | Sonnet 5.5 | `apps/api/internal/csvimport` | not started |
| P2-L2 - Courses, sections, staff, enrollments, imports, my-sections (Task 10, D) | Opus 5.5 design, Sonnet 5.5 build | `docs/design/P2-L2.md`, `apps/api/internal/course` | not started |
| P2-L3 - Web `courses` and `my-sections` features (Task 11, D) | Opus 5.5 design, Sonnet 5.5 build | `docs/design/P2-L3.md`, `apps/web/src/{features/courses,features/my-sections,routes}` | not started |

## Phase 3 - End to end
| Lane | Model | Scope (disjoint) | Status |
|------|-------|------------------|--------|
| P3-L1 - Serve web from the API, Playwright flow, CI `e2e` job (Task 12, W) | Sonnet 5.5 | `apps/api/internal/httpx/static.go`, `apps/web/e2e`, `ci.yml` | not started |
Notes P0-L1: review found 9 defects (shutdown did not drain, panics lost their request id, `DecodeJSON` accepted trailing data, plain-text 404s, unvalidated cursor id and client request id); all fixed before landing. `httpx.WithUserID`/`WithUniversityID` fill a per-request struct and return nothing - callers must not reassign ctx. Middleware order is RequestID, Log, Recover (innermost).

Notes P0-L3: Tailwind 4 (no `tailwind.config.ts`), React 18.3, TypeScript ~5.9 and jsdom ^26 pinned (typescript-eslint and Node 22.11 limits). `client.ts` uses an absolute base URL (`origin + /api/v1`). `unwrap` derives the error code from the status when a body has no envelope.

Notes P0-L4 (design): the contract wins over the Go structs printed in the implementation plan where they differ (nested `university` in `/me`, `Section` always embeds `course`, `term`, `staff`, staff add returns 202 with the invitation, imports return 200 with the applied rows).

Notes P1-L1 (design): Microsoft sign-in needs the `email` and `xms_edov` optional claims in the app registration. Secure cookies work on `http://localhost` in Chromium and Firefox, not Safari.

## Backlog
- Spec §4 "the platform CLI uses a separate role for `universities` administration": not built; CLI uses `UNICOURSE_DATABASE_URL`. Add a dedicated role when a non-engineer runs the CLI (raised by orchestrator).
- Whoever consumes `?next=` on sign-in must accept same-origin paths only (raised by P0-L3 review).
- Per-IP magic-link limit is in memory and per process; move to Postgres when more than one API process runs (raised by P1-L1).
- Enable branch protection with required checks `api`, `web` once the repo plan allows (CLAUDE.md).

## Journal
- 2026-10-05: Spec 1 and its implementation plan merged (#4). Owner started `/orchestrate` on a 30-minute loop and delegated decisions overnight (D8). PLAN.md filled from the implementation plan. Docker Desktop started for local Postgres. Launched P0-L1, P0-L3 workers and P0-L4, P1-L1 designers in parallel.
- 2026-10-06: P0-L1 (#6) and P0-L3 (#7) landed after review fixes and orchestrator verification. P0-L4 and P1-L1 designs done; D9-D16 ruled. Launched P0-L2 and P0-L4 workers and P1-L2, P1-L4 designers.
