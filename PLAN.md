# UniCourse - orchestration plan

Spec: `docs/specs/2026-10-05-platform-core-and-staffing.md` · Implementation plan (task detail, TDD steps): `docs/plans/2026-10-05-platform-core-and-staffing.md`

## Decisions
| # | Decision | Status |
|---|----------|--------|
| D1 | Stack and constraints as spec §3: Go 1.27 stdlib `net/http`, pgx/x-oauth2/go-oidc/x-crypto only, PostgreSQL 16 with RLS, React 18 + Vite + TS strict + TanStack + Tailwind, npm | Agreed (owner, spec 2026-10-05) |
| D2 | Contract file is `apps/api/openapi.json`, not YAML (stdlib has no YAML parser; contract test reads it) | Agreed (owner, PR #4) |
| D3 | Web runtime/dev deps beyond the framework: `openapi-fetch`, `openapi-typescript` only (plan Task 3) | Orchestrator 2026-10-05, owner to confirm |
| D4 | Postgres roles: the image superuser stays `postgres`; the API/test role `unicourse` is `NOSUPERUSER NOBYPASSRLS` and owns `unicourse` and `unicourse_test` (superusers bypass RLS, so the plan's `POSTGRES_USER=unicourse` would void the tenancy test). Compose init script and a CI step create it | Orchestrator 2026-10-05, owner to confirm |
| D5 | `apps/api/README.md` and `apps/web/README.md` are gitignored by the owner, so lanes do not write them; run instructions live in the compose file header and `package.json` scripts | Orchestrator 2026-10-05 |
| D6 | Lane logs are written in the lane's own worktree (`.claude/lanes/<LANE-ID>.md`) and land with the lane's PR; lanes read each other's via `.worktrees/*/.claude/lanes/` | Orchestrator 2026-10-05 |
| D7 | A sub-agent has ~100k tokens, so plan tasks 5, 6, 8, 10, 11 are built as sequential sub-lanes (`<LANE-ID>a`, `b`, …) on the same branch, cut by that task's designer | Orchestrator 2026-10-05 |
| D8 | Owner is away overnight 2026-10-05 and delegated every decision ("don't wait for my decisions"); the orchestrator decides, records it in this table as "owner to confirm", and never changes usage limits, on-demand usage or credits | Agreed (owner, 2026-10-05) |
| D9 | How auth reads `memberships`/`invitations` across universities under single-policy RLS (spec §4 vs §5 `SignIn`, `/me`) | Open - P1-L1 designer proposes, orchestrator rules before P0-L2 writes the migration |
| D10 | Where pending staff assignments for not-yet-members live (`invitations.section_staff jsonb` vs `pending_section_staff` table; plan Task 10 Step 1) | Open - P1-L1 designer proposes (affects `0001_init.sql`) |

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
| P0-L1 - API skeleton and `httpx` (Task 1, W) | Sonnet 5.5 | `apps/api/{go.mod,cmd/api,internal/config,internal/httpx,docker-compose.yml,docker/}` | worker running |
| P0-L2 - `pg`, schema, RLS, `testdb`, `audit` (Task 2, W) | Sonnet 5.5 | `apps/api/{migrations,internal/pg,internal/testdb,internal/audit}`, `cmd/api/main.go`, `ci.yml` api job | not started (after P0-L1, D9, D10) |
| P0-L3 - Web skeleton and API client (Task 3, W) | Sonnet 5.5 | `apps/web/` | worker running |
| P0-L4 - `openapi.json` and contract test (Task 4, D) | Opus 5.5 design, Sonnet 5.5 build | `docs/design/P0-L4.md`, `apps/api/openapi.json`, `apps/api/internal/httpx/{routes,contract_test}.go`, `apps/web/src/api/types.ts`, `ci.yml` web job | designer running; worker after P0-L1 and P0-L3 land |

## Phase 1 - Platform core
| Lane | Model | Scope (disjoint) | Status |
|------|-------|------------------|--------|
| P1-L1 - Authentication (Task 5, D) | Opus 5.5 design, Sonnet 5.5 build | `docs/design/P1-L1.md`, `apps/api/internal/auth` | designer running |
| P1-L2 - Tenant scope, `authz`, departments, terms, memberships, invitations, audit route (Task 6, D) | Opus 5.5 design, Sonnet 5.5 build | `docs/design/P1-L2.md`, `apps/api/internal/{authz,tenant}` | not started |
| P1-L3 - `unicourse-admin` CLI (Task 7, W) | Sonnet 5.5 | `apps/api/cmd/unicourse-admin` | not started |
| P1-L4 - Web `auth` and `admin` features (Task 8, D) | Opus 5.5 design, Sonnet 5.5 build | `docs/design/P1-L4.md`, `apps/web/src/{features/auth,features/admin,routes}` | not started |

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

## Backlog
- Spec §4 "the platform CLI uses a separate role for `universities` administration": not built; CLI uses `UNICOURSE_DATABASE_URL`. Add a dedicated role when a non-engineer runs the CLI (raised by orchestrator).
- Enable branch protection with required checks `api`, `web` once the repo plan allows (CLAUDE.md).

## Journal
- 2026-10-05: Spec 1 and its implementation plan merged (#4). Owner started `/orchestrate` on a 30-minute loop and delegated decisions overnight (D8). PLAN.md filled from the implementation plan. Docker Desktop started for local Postgres. Launched P0-L1, P0-L3 workers and P0-L4, P1-L1 designers in parallel.
