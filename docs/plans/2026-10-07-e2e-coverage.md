# End-to-end Coverage — Implementation Plan

> **For agentic workers:** executed by `/orchestrate` (orchestrator + `designer` → `worker` → `reviewer`). Global constraints, gates and git rules are those of `docs/plans/2026-10-05-platform-core-and-staffing.md`. Each task is one lane.

**Goal:** every user-visible flow, for every role, has a Playwright case that runs in `npm run e2e` and CI (Spec 9 §1).

**Architecture:** one catalogue (`docs/design/P16-L1.md`), one harness lane that makes spec files independent and the port configurable, one spec file per area built from the catalogue, one accessibility sweep last. No product code changes (Spec 9 §7).

**Tech Stack:** `@playwright/test` (installed), `unicourse-admin seed-demo` and the HTTP API for seeding; `@axe-core/playwright` only in Task 9 (D125).

**Spec:** `docs/specs/2026-10-07-e2e-coverage.md`.

## Global Constraints

- Nothing under `apps/web/src` or `apps/api`. A defect a case exposes is reported in the lane log with its catalogue ID, the case becomes `test.fixme('<ID>: <finding>')`, and the orchestrator opens a fix lane.
- The three existing spec files are not edited (D60, D80); `screens.ts` changes only its imports (Task 2).
- Every case carries its catalogue ID in the test or step title (`E-AUTH-03`), so the catalogue and the files can be diffed.
- Each lane runs its file on its own port: `UNICOURSE_E2E_PORT=181<nn> npm run e2e -- <file>`, where `nn` is the lane number (L3 = 18103, and so on). Never kill another process on a port; report it.
- Gates per lane: `npm run lint`, `npm run typecheck`, `npm test -- --run`, `npm run build`, the lane's file green twice in a row on its port.

## Review Focus

- Every catalogue ID for the lane's area is implemented or marked `fixme` with a finding; nothing in the file is outside its area.
- Assertions quote the spec's copy; selectors are roles and labels; no `waitForTimeout`, no test ids, no shared state across files.
- Seeding uses `seed-demo` or `page.request` unless the UI flow is the assertion (Spec 9 §4).
- Two consecutive green runs; the trace of a failure names the role and page.

## Phase 16 — End-to-end coverage

### Task 1 (P16-L1, D): test-case catalogue and harness design

Spec 9 §3.1, §4, §5. Scope: `docs/design/P16-L1.md`. Reads Spec 1 to Spec 8, the README persona table, `apps/web/src/features/**` and `routes/**`, the three existing spec files and `screens.ts`. Output: §A catalogue grouped by spec file with IDs `E-<AREA>-<nn>` (role, preconditions, steps, assertion with the spec section cited, seeding method); §B mapping of the existing files to IDs; §C uncoverable cases with reasons; §D harness changes as exact function signatures and the `UNICOURSE_E2E_PORT` wiring; §E an estimate of cases per file. Escalate copy or behaviour that no spec names.

### Task 2 (P16-L2, W): harness, `demo.spec.ts`, CI timeout

Spec 9 §3.2, §3.4, §5; design §D. Scope: `apps/web/e2e/{env.ts,helpers.ts,global-setup.ts,global-teardown.ts,demo.spec.ts}`, `apps/web/e2e/screens.ts` (imports only), `apps/web/playwright.config.ts`, `.github/workflows/ci.yml` (`e2e` job `timeout-minutes` only). Lands first; Tasks 3 to 9 branch from its head. Gates add `npm run screens` green and the existing three files green on the new harness.

### Task 3 (P16-L3, W): `auth.spec.ts`

Spec 9 §3.3 row auth; design §A `E-AUTH-*`. Scope: `apps/web/e2e/auth.spec.ts`.

### Task 4 (P16-L4, W): `admin.spec.ts`

Design §A `E-ADMIN-*`. Scope: `apps/web/e2e/admin.spec.ts`.

### Task 5 (P16-L5, W): `courses.spec.ts`

Design §A `E-COURSES-*`. Scope: `apps/web/e2e/courses.spec.ts`, new fixtures under `apps/web/e2e/fixtures/courses-*` only.

### Task 6 (P16-L6, W): `grades.spec.ts`

Design §A `E-GRADES-*`. Scope: `apps/web/e2e/grades.spec.ts`, fixtures `apps/web/e2e/fixtures/grades-*`.

### Task 7 (P16-L7, W): `catalog.spec.ts`

Design §A `E-CATALOG-*` (Catalog, My sections, schedule, phone agenda). Scope: `apps/web/e2e/catalog.spec.ts`.

### Task 8 (P16-L8, W): `content-edit.spec.ts`

Design §A `E-CONTENT-*` not already covered by `content.spec.ts` (§B). Scope: `apps/web/e2e/content-edit.spec.ts`, fixtures `apps/web/e2e/fixtures/content-*`.

### Task 9 (P16-L9, W, after D125): `a11y.spec.ts`

Design §A `E-A11Y-*`. Scope: `apps/web/e2e/a11y.spec.ts`, `apps/web/package.json`, `package-lock.json` (one pinned dev dependency). Runs after Tasks 3 to 8 land so the sweep covers the pages they seed. Dropped if the owner declines D125.

### Close

Orchestrator: full suite on `main` with the wall-clock time in PLAN.md, `workers` ruling per D126, release `v0.16.0`.
