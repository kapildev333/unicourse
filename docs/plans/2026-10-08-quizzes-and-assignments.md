# Quizzes and Assignments — Implementation Plan

> **For agentic workers:** executed by `/orchestrate` (orchestrator + `designer` → `worker` → `reviewer` lanes, one branch and worktree per lane). Each task is one lane. Global constraints, gates and git rules are those of `docs/plans/2026-10-05-platform-core-and-staffing.md`. Tests need `UNICOURSE_TEST_DATABASE_URL` and `UNICOURSE_TEST_OWNER_DATABASE_URL` (D48); the API process needs `UNICOURSE_FILES_DIR` (Spec 6). Web unit tests on the owner's Node 22.11 need `NODE_OPTIONS=--experimental-require-module`.

**Goal:** staff create quizzes (multiple choice and fill in the blank, a window with automatic submission at the close, automatic grading, staff review) and assignments (text and files, due date, hand grading) on a section; every one is a gradebook column (Spec 11 §1).

**Architecture:** `assessments.kind` extends Spec 4's table; two new Go packages `internal/quiz` and `internal/assignment` with one migration `0010`; a sweeper goroutine submits attempts at the close; two web feature directories and two section tabs; e2e files of their own.

**Spec:** `docs/specs/2026-10-08-quizzes-and-assignments.md` — every task cites its sections. **Visual rules:** Spec 3; the taste rubric (D78) for the web design lane. **Decisions:** D137-D146 in PLAN.md.

**Coexistence (D145, D146).** Seven Phase 17 PRs (#153-#159) land first on local gates while GitHub Actions is disabled; migration numbers `0008` and `0009` are theirs. Hot spots this phase: `migrations/0010` P18-L3; `routes.go`, `openapi.json`, `types.ts` additive, P18-L3 then L4 then L5 (each branches from the previous head and rebases on landing); `section-tabs.tsx` P18-L6a (L7 branches from it); `cmd/api/main.go` P18-L4b; `cmd/unicourse-admin/main.go` P18-L5; `features/grades/*` P18-L6a.

## Global Constraints

- Go 1.27 standard library; pgx, x/oauth2, go-oidc, x/crypto only (D1). No new npm dependency.
- Every new table: `university_id`, forced RLS with `tenant_isolation`, composite tenant FKs (D19), `created_at`/`updated_at` with the trigger (D20; `TestEveryUpdatedAtTableHasTrigger`), `(university_id, id)` unique where referenced.
- Contract conventions D16; D46 (unknown JSON keys rejected, explicit camelCase tags); audit rows in the request transaction; every time comparison in SQL with `now()` (D30).
- Points and scores are `numeric` strings handled with `math/big` (Spec 4, D68); never floats.
- Authorization only through `authz.CanEditContent` (staff) and an enrollment check (students), Spec 11 §4; check order section → role → state → body.
- Web: Spec 3 tokens and shared components, Lucide, no literal colours, no toasts, dialogs trap focus, live regions for countdown and save status; every page checked in light and dark and at 390 px.
- Test-driven: failing test → implementation → pass → commit, per step.

## Review Focus

1. **The close is authoritative and happens once** — a `PUT …/attempts/me` at or after `closesAt` is 409 and changes nothing; the sweeper submits each unsubmitted attempt of a closed quiz exactly once across two processes (`skip locked`), writes the `scores` row and `autoSubmitted`; an open quiz's attempts are never touched. Pinned in Task 4b.
2. **Students never see keys or scores early** — `correct`, `answers`, `grading`, `score`, `feedback` are absent from every student response until `published`; questions are absent before `opensAt`. Pinned in Task 4a/4c and Task 6b.
3. **Grading is exact** — set equality for choices, normalised equality for blanks, decimal sums; review points are bounded by the question's points and the recompute matches the gradebook cell. Pinned in Task 4a and 4c.
4. **The gradebook is unchanged for manual assessments** — Spec 4 tests still pass; `manual` rows reject a window; a quiz rejects `maxPoints`; `dueOn` is derived in the university timezone. Pinned in Task 3.
5. **Files behave as materials do** — allowlist, limit, attachment headers, blob removal after commit, `prune-files` coverage. Pinned in Task 5.

## File Structure

```
apps/api/migrations/0010_quizzes_and_assignments.sql
apps/api/internal/grade/{assessment.go,assessment_test.go}          kind, description, opensAt/closesAt, dueOn derivation, maxPoints rule, student listing
apps/api/internal/quiz/{grading.go,grading_test.go}                  pure grading
apps/api/internal/quiz/{question.go,attempt.go,submit.go,sweeper.go,review.go,handler.go,fixture_test.go} + tests
apps/api/internal/assignment/{submission.go,file.go,grade.go,handler.go} + tests
apps/api/internal/app/app.go                                         quiz.Register, assignment.Register
apps/api/cmd/api/main.go                                             sweeper
apps/api/cmd/unicourse-admin/main.go                                 prune-files reads submission_files
apps/api/openapi.json · apps/web/src/api/types.ts · docs/postman/*
apps/web/src/features/quiz/{queries.ts,quiz-list.tsx,quiz-dialog.tsx,question-editor.tsx,take-quiz.tsx,countdown.ts,results.tsx,attempts-review.tsx} + tests
apps/web/src/features/assignment/{queries.ts,assignment-list.tsx,assignment-dialog.tsx,submission-form.tsx,submissions-grading.tsx} + tests
apps/web/src/features/section/section-tabs.tsx                      + Quizzes, Assignments
apps/web/src/features/grades/{assessment-dialog,gradebook,my-grades}.tsx   D144 edits
apps/web/src/routes/$slug/sections_.$id.{quizzes,assignments}*.tsx
apps/web/e2e/{quizzes,assignments}.spec.ts
apps/api/internal/seed/demo.json · docs/seed/fixture.schema.json · apps/api/internal/seed/*.go
docs/design/P18-L1.md (API and data) · docs/design/P18-L2.md (web)
```

## Phase 18 — Quizzes and assignments

### Task 1 (P18-L1, D): API and data design
Spec 11 §3, §4, §5, §10. Decide: the `0010` DDL; the Go types and JSON shapes; the grading function signature; the sweeper's loop, claiming query and university iteration; where the `scores` upsert lives (a small exported function in `grade`, called by `quiz` and `assignment`, is the expected answer); the student listing rule's SQL; the review and grade recompute; the submission-file upload reuse of `blob` and the `content` streaming helper (move the helper into `blob` if both need it); the test names per Spec 11 §7. Scope: `docs/design/P18-L1.md`.

### Task 2 (P18-L2, D): web design with the taste rubric
Spec 11 §6, Spec 3, the taste-skill §4 and §9 (D78, exceptions D100 and D101) and the Udemy/Canvas references of Spec 5. Decide every screen's layout, states, copy, components used, the countdown behaviour, the question editor's form model, the review page's in-place editing, phone layout, and the two tabs' placement. No new component without a test and a line in the design. Scope: `docs/design/P18-L2.md`. Runs in parallel with Task 1 from the spec's shapes; differences are relayed by the orchestrator.

### Task 3 (P18-L3, W): migration 0010 and assessment kinds (after Task 1)
Spec 11 §3 `assessments` row, §5 `Assessment` paragraph, §7 `grade`. Steps: failing tests for kind/window validation per kind, `maxPoints` rejected for quiz, `dueOn` derivation (New York 23:30 case), `PATCH` cannot change kind, student listing rule; then the migration (all four tables, so Tasks 4 and 5 share one file), the `grade` changes, `openapi.json` `Assessment` and request schemas, `types.ts`, Postman. Scope: `apps/api/migrations/0010_*.sql`, `apps/api/internal/grade/assessment*.go` (+ the shared SELECT), `apps/api/internal/pg/*_test.go` (table list), `openapi.json`, `types.ts`, `docs/postman/*`.

### Task 4 (P18-L4, W, after Task 3): `internal/quiz`
Sub-lanes on one branch from Task 3's head: **4a** pure grading (tests first) and questions `GET`/`PUT` with validation, `maxPoints` recompute, lock after an attempt; **4b** attempts start/read/save/submit, the submit function, the `scores` upsert, the sweeper with the two-process test, `cmd/api/main.go` wiring; **4c** staff attempts list and review, student visibility rules, role matrix, RLS test extension, audit rows, `routes.go`, `openapi.json`, `types.ts`, Postman. Scope: `apps/api/internal/quiz/**`, `apps/api/internal/app/app.go`, `apps/api/cmd/api/main.go`, `apps/api/internal/grade/score.go` (the exported upsert only, as the design names it), `routes.go`, `openapi.json`, `types.ts`, `docs/postman/*`.

### Task 5 (P18-L5, W, after Task 3, in parallel with Task 4): `internal/assignment`
Spec 11 §3 submissions and files, §5 submission routes, §7 `assignment`. Steps: submission `PUT` window tests → implementation; files upload/remove/download reusing `blob`; grade with the `scores` upsert; role matrix; RLS; audit; `prune-files`; contract. Branches from Task 3's head; rebases over Task 4's landing (additive hot spots). Scope: `apps/api/internal/assignment/**`, `apps/api/internal/app/app.go` (one line), `apps/api/cmd/unicourse-admin/main.go` (prune-files), `routes.go`, `openapi.json`, `types.ts`, `docs/postman/*`, `apps/api/internal/blob/**` only if Task 1's design moves the streaming helper there.

### Task 6 (P18-L6, W, after Tasks 2 and 4): web quizzes
Sub-lanes: **6a** tabs, quiz list with states per role, quiz dialog, question editor, D144 gradebook edits; **6b** take page (autosave, countdown, submit, results view); **6c** review page. Each step test-first with Vitest against contract-shaped responses; light and dark and 390 px checked. Scope: `apps/web/src/features/quiz/**`, `apps/web/src/routes/$slug/sections_.$id.quizzes*.tsx`, `routeTree.gen.ts`, `apps/web/src/features/section/section-tabs.tsx` (+ test), `apps/web/src/features/grades/{assessment-dialog,gradebook,my-grades}.tsx` (+ tests).

### Task 7 (P18-L7, W, after Tasks 2, 5 and 6a): web assignments
Assignment list and dialog, student submission page (reusing `features/section/upload.ts` and `files.ts`), grading page. Scope: `apps/web/src/features/assignment/**`, `apps/web/src/routes/$slug/sections_.$id.assignments*.tsx`, `routeTree.gen.ts`.

### Task 8 (P18-L8, W, after Tasks 6 and 7): demo seed and end to end
Spec 11 §7 end to end and §10 seed line: `demo.json` gains one open quiz with two submitted attempts and one assignment with two submissions (one graded) on CS101, written through the real handlers (D104); the fixture schema follows; `quizzes.spec.ts` and `assignments.spec.ts` assert the §1 criterion (quiz closes 60 s after creation; the test observes the automatic submission). Scope: `apps/api/internal/seed/**`, `docs/seed/fixture.schema.json`, `apps/web/e2e/quizzes.spec.ts`, `apps/web/e2e/assignments.spec.ts`, `apps/web/e2e/helpers.ts` only for a new exported helper the two files share.

### Close
Full Go suite and `npm run e2e` on main, the orchestrator's real-HTTP pass (start, autosave, close, sweeper, review, files) and a browser pass of every screen in light and dark, PLAN.md status PR, release `v0.18.0`.
