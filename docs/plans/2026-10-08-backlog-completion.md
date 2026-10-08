# Backlog Completion — Implementation Plan

> **For agentic workers:** executed by `/orchestrate` (orchestrator + `designer` → `worker` → `reviewer`). Global constraints, gates and git rules are those of `docs/plans/2026-10-05-platform-core-and-staffing.md`. Each task is one lane.

**Goal:** every bounded backlog item in Spec 10 §2 is built, tested and landed (Spec 10 §1).

**Spec:** `docs/specs/2026-10-08-backlog-completion.md`; design `docs/design/P17-L1.md` (Task 1).

## Global Constraints

- Spec 10 §4. Hot-spot owners this phase: `migrations/0008` P17-L2, `migrations/0009` P17-L7, `cmd/api/main.go` P17-L3, `routes.go` and `openapi.json` P17-L6 then P17-L4 (branches from L6's head), `ci.yml` P17-L4 then P17-L10 (branches from L4's head), `apps/web/package.json` P17-L9.
- Each lane runs e2e on its own port and database (`UNICOURSE_E2E_PORT=171<nn>`, `unicourse_e2e_<nn>`), as in Phase 16.
- Test-driven; a copy change names the e2e selector it updates.

## Review Focus

- Correctness of the shared-state changes (B1 buckets under concurrency, B2 exactly-once delivery, B5 trust walk), contract and e2e consistency, Spec 3 for web changes, scope.

## Phase 17 — Backlog completion

### Task 1 (P17-L1, D): API shapes
Spec 10 §2 B1, B2, B3, B4, B5, B7, B9, B10, B12-F3: table and query shapes, env variables, contract additions, test names. Scope: `docs/design/P17-L1.md`.

### Task 2 (P17-L2, W): per-IP and per-email limits in Postgres (B1)
Scope: `apps/api/internal/auth/limiter*.go` (and callers in that package), `apps/api/migrations/0008_rate_buckets.sql`.

### Task 3 (P17-L3, W): outbox senders, `Forwarded`, IPv4-mapped prefixes, `WEB_ORIGIN` validation (B2, B5, B10)
Scope: `apps/api/internal/outbox/**`, `apps/api/internal/httpx/clientip*.go`, `apps/api/internal/config/**`, `apps/api/cmd/api/main.go`, README env table lines for the new variables.

### Task 4 (P17-L4, W, after L6): upload limit env, CI boundary test, upload policy route, web reads it (B7, B9)
Scope: `apps/api/internal/content/material*.go`, `apps/api/internal/blob/**`, `routes.go`, `openapi.json`, `apps/web/src/api/types.ts` (generated), `apps/web/src/features/section/material-dialogs.tsx` (+ test), `.github/workflows/ci.yml` (the api job's test flags only).

### Task 5 (P17-L5, W): overlapping meetings rule, generic invitation 409, F4, F5, F6, F10 (B6, B12)
Scope: `apps/api/internal/course/meeting*.go`, `apps/api/internal/tenant/invitation*.go`, `apps/web/src/features/courses/{meetings-editor,section-dialog,course-dialog,remove-dialog,section-list}.tsx` (+ tests), `apps/web/src/features/admin/{departments,terms,audit-log}.tsx` (+ tests), the e2e selectors the copy changes touch (named in the lane log).

### Task 6 (P17-L6, W): `q` on departments, terms, courses; picker search (B3)
Scope: the three list handlers in `apps/api/internal/{tenant,course}`, `openapi.json` (query params only), `apps/web/src/api/types.ts`, `apps/web/src/features/admin/department-picker.tsx`, the term select and course picker components (+ tests).

### Task 7 (P17-L7, W): platform CLI role (B4)
Scope: `apps/api/migrations/0009_platform_role.sql`, `apps/api/docker/initdb.sql`, `apps/api/cmd/unicourse-admin/**`, README (CLI section), `docs/` page if the design names one.

### Task 8 (P17-L8, W): seed `--fixture`, fixture schema, `make dev`, screens via seed-demo (B8)
Scope: `apps/api/internal/seed/**`, `apps/api/cmd/unicourse-admin/**` (the `seed-demo` flag only; coordinate with L7 by file: L8 owns `seed.go`/`seed_demo.go` lines, L7 owns `create-university`), `docs/seed/fixture.schema.json`, root `Makefile`, `apps/web/e2e/screens.ts`, README quick start.

### Task 9 (P17-L9, W): code-split routes, XHR upload progress (B13)
Scope: `apps/web/src/routes/**` (lazy imports), `apps/web/vite.config.ts`, `apps/web/src/features/section/material-dialogs.tsx` upload path (coordinate with L4: L9 branches from L4's head), `apps/web/src/lib/upload.ts` (new), `apps/web/package.json` if a chunk-size setting is needed.

### Task 10 (P17-L10, W, after L4): CI two-origin dev check (B11)
Scope: `.github/workflows/ci.yml` (one new step or job), `apps/web/scripts/two-origin-check.mjs` (new).

### Close
Full Go suite and `npm run e2e` on main, real-process checks (Spec 10 §5), PLAN.md status PR, release `v0.17.0`.
