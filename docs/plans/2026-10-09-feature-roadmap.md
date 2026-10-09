# Feature Roadmap, Phase 21 "Stay on top" — Implementation Plan

> **For agentic workers:** executed by `/orchestrate` (orchestrator + `designer` → `worker` → `reviewer` lanes, one branch and worktree per lane). Global constraints, gates and git rules are those of `docs/plans/2026-10-08-quizzes-and-assignments.md`. D146/D148 apply: local gates, no CI. This plan covers only the roadmap's next phase. Later phases get their own spec and plan once the owner picks them.

**Goal:** roadmap §4 Phase 21. Members see what is due across their sections (R1), get in-app notifications with an announcement-mail opt-out (R2), subscribe to a private calendar feed (R3), and read every date in the university's time zone and locale (R4).

**Spec:** `docs/specs/2026-10-09-feature-roadmap.md` §2 R1-R4. It is the spec of record for this phase ("Spec 13 by reference") once the owner says "go". **Design:** `docs/design/P21-L1.md`.

**Owner answers this plan assumes** (roadmap §6; the recommended defaults apply unless the owner rules otherwise, recorded from D157):
- **[Q1]** start here;
- **[Q8]** three in-app kinds; per-post announcement mail plus a per-member opt-out; no digest;
- **[Q9]** a secret token per member per university, rotatable, never logged;
- **[Q12]** Upcoming is a panel on My sections, and `/{slug}` still lands on `/me`.

**Hot spots (serial, rebase in landing order):**
- `migrations/0012`: P21-L2 only (one migration for notifications, the opt-out and calendar tokens);
- `openapi.json`, `types.ts`, Postman: P21-L2, then P21-L3;
- `internal/app/app.go` registration: P21-L2, then P21-L3;
- `components/app-shell.tsx` and `account-menu.tsx`: P21-L4;
- `lib/format.ts`: P21-L5.

## Global constraints (in addition to the base plan)
- Go stdlib plus D1 modules; no new npm dependency (no calendar or ICS library: the feed is hand-written RFC 5545 text, with CRLF line endings and 75-octet folding).
- New tables: `university_id`, forced RLS, composite tenant FKs (D19), `updated_at` trigger (D20).
- Fan-out is one `insert ... select` inside the producer's request transaction; no goroutines.
- The calendar token route sits outside `tenant.Wrap`, like magic links. It resolves the token to (user, university), runs under that tenant's RLS, answers 404 for unknown or revoked tokens, and its route pattern is what gets logged (D27, D32).
- Web follows the Spec 3 tokens and shared components; the bell is an icon-only control with `aria-label` and `title`; checked in light, dark and at 390 px.

## Review focus
1. A notification reaches only members who can read its source: enrolled students and the section's staff. It never reaches the author, and it never reaches students for an unpublished grade.
2. Upcoming and the feed list only work the member can do (enrolled, not submitted, window open or due within 14 days) and never leak quiz keys or other students' state.
3. The calendar token is a bearer credential. It is read-only, rotation invalidates the old URL, it is revoked with the membership, and it is absent from the request log.
4. Times show in the university's zone. A non-UTC browser against a UTC university shows UTC, with the zone named.

## Tasks
- **Task 1 (P21-L1, designer, Opus 5.5).** Design for R1-R4:
  - DDL and RLS for `notifications`, the mail opt-out, and `calendar_feeds`;
  - notification kinds and producers (announcement create; quiz or assignment create; published or finalize), plus read and mark-read routes and the unread count;
  - `GET /me/upcoming` shape and SQL; the feed format (VEVENT with RRULE bounded by term dates, TZID from the university, deadlines as events) and token routes;
  - `university.timezone`/`locale` on `GET /me`;
  - web pages, copy and e2e case names, including how e2e handles the UTC university versus the runner's zone (helpers create `--timezone UTC`);
  - file ownership per lane.

  Scope: `docs/design/P21-L1.md`.
- **Task 2 (P21-L2, worker, Sonnet 5.5, after 1).** Migration `0012`; new package `internal/notification` (list, count, mark read, opt-out). Producer calls are inserted in `internal/content/announcement.go` (which also honours the opt-out when building mail), `internal/grade/assessment.go` (publish), `internal/grade/final.go` (finalize), and the quiz and assignment create path in `internal/grade`. Contract, `types.ts`, Postman.
  Scope: `migrations/0012*`, `internal/notification/**`, the named functions only, `internal/app/app.go`, contract files.
- **Task 3 (P21-L3, worker, Sonnet 5.5, after 2 lands).** `GET /me/upcoming` in `internal/grade`; calendar feed and token routes; timezone and locale on `GET /me` in `internal/auth/handler.go`; contract, `types.ts`, Postman.
  Scope: `internal/grade/upcoming*`, a new `internal/calendar/**`, `internal/auth/handler.go` (the `university` struct and query), `internal/app/app.go`, contract files.
- **Task 4 (P21-L4, worker, Sonnet 5.5, after 2; parallel with 3).** Header bell with unread count; a `/{slug}/notifications` page with mark-read; the opt-out toggle in the account menu.
  Scope: `features/notifications/**`, `routes/$slug/notifications.tsx`, `components/app-shell.tsx`, `components/account-menu.tsx`, and their tests.
- **Task 5 (P21-L5, worker, Sonnet 5.5, after 3).**
  - Upcoming panel on My sections.
  - A "Calendar feed" dialog: copy the URL, rotate.
  - `formatDate`/`formatDateTime` take the university's zone and locale, and so does `features/quiz/window.ts`.
  - Run the full e2e suite, because displayed times may shift.

  Scope: `features/my-sections/**`, `lib/format.ts` (+test), `features/quiz/window.ts`.
- **Task 6 (P21-L6, worker, Sonnet 5.5, after 4 and 5).** e2e for:
  - the notification on announcement and on release, and mark-read;
  - Upcoming for a student and for staff;
  - calendar feed fetch, rotation and 404;
  - the timezone display.

  Scope: a new `e2e/notifications.spec.ts`, a new `e2e/upcoming.spec.ts`, and `e2e/helpers.ts` only if the design says so.
- **Reviews:** one Opus 5.5 reviewer per worker lane (P21-L2-review … P21-L6-review), as in Phases 18-19.
- **Close:** full Go, web and e2e suites on main; PLAN.md status PR; release `v0.21.0`.

## Marked for the owner
- **[Q1, Q8, Q9, Q12]** above. A different answer to Q8 or Q12 changes Tasks 2, 4 and 5 before Task 1 starts.
- Out of this phase on purpose: due-date reminder mail (needs a scheduler), digests, web push (needs a deployment target, Q3), notification pruning. Add a CLI prune when the table grows.
