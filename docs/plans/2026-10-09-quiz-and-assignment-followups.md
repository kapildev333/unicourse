# Quiz and Assignment Follow-ups — Implementation Plan

> **For agentic workers:** executed by `/orchestrate` (orchestrator + `designer` → `worker` → `reviewer` lanes). Global constraints, gates and git rules as `docs/plans/2026-10-08-quizzes-and-assignments.md`. D146/D148: local gates, no CI.

**Goal:** Spec 12 §1: regrade after a key change, late submissions with a penalty, per-attempt time limits, server read timeouts.

**Spec:** `docs/specs/2026-10-09-quiz-and-assignment-followups.md`. **Design:** `docs/design/P19-L1.md` (API, data and web, one document).

**Hot spots:** migration `0011` P19-L2; `openapi.json`/`types.ts`/Postman additive, P19-L2 then L3/L4/L5 rebase in landing order; `cmd/api/main.go` and `internal/config` P19-L6; `features/quiz/*` and `features/assignment/*` P19-L7; e2e files P19-L8.

## Review Focus
1. A regrade never touches `manual` grading entries, unsubmitted attempts or `final_grades`, and writes exactly one audit row per changed attempt (Task 3).
2. The effective deadline is one SQL expression used by the PUT, the student GET and the sweeper; a limit never extends past the close (Task 4).
3. The penalised number is written to `scores` with half-up rounding and the raw grade stays on the submission; a late write after `late_until` is 409 (Task 5).
4. The read timeout cuts a slow body on ordinary routes and the upload deadline extension keeps a slow 100 MB upload alive (Task 6).

## Tasks
- **Task 1 (P19-L1, D, Opus):** design for Tasks 2-5 and 7: DDL, the deadline SQL, the regrade algorithm and its `manual` flag, the penalty write, JSON shapes, web form models and copy, test names, the sub-lane file ownership. Scope `docs/design/P19-L1.md`.
- **Task 2 (P19-L2, W, after Task 1):** migration `0011`, `grade` assessment fields and validation, contract for `Assessment`, `types.ts`, Postman. Scope `migrations/0011`, `internal/grade/assessment*`, `pg` tests, `openapi.json` (Assessment schemas), `types.ts`, Postman.
- **Task 3 (P19-L3, W, after Task 2):** regrade in `internal/quiz` (questions PUT with ids, `manual` on review, regrade function, audit), contract. Scope `internal/quiz/{question,review,grading,submit}*`, `openapi.json` (questions and attempt schemas), `types.ts`, Postman.
- **Task 4 (P19-L4, W, after Task 2, parallel with 3 on disjoint files):** time limit in `internal/quiz` (`deadlineAt`, PUT/GET checks, sweeper claim at the deadline). Scope `internal/quiz/{attempt,sweeper}*`, contract (Attempt schema).
- **Task 5 (P19-L5, W, after Task 2, parallel):** late window and penalty in `internal/assignment`. Scope `internal/assignment/**`, contract (Submission schemas).
- **Task 6 (P19-L6, W, now, no design needed beyond Spec 12 §3):** server timeouts. Scope `internal/config/**`, `cmd/api/main.go`, `internal/httpx` (a deadline helper if needed), `internal/content/material.go` and `internal/assignment/file.go` (the one `SetReadDeadline` call each, coordinated: L6 lands before L5 branches, or L5 rebases), README env table.
- **Task 7 (P19-L7, W, after Tasks 1, 3, 4, 5):** web: dialogs, editor regrade state and confirm, countdown from `deadlineAt`, late states, grading penalty line. Scope `features/quiz/**`, `features/assignment/**` (+ tests).
- **Task 8 (P19-L8, W, after Task 7):** e2e E-QUIZ-11, E-QUIZ-12, E-ASG-09. Scope the two spec files.
- **Close:** full suite on main, PLAN.md status PR, release `v0.19.0`.
