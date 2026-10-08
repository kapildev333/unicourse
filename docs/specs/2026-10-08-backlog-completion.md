# Spec 10: Backlog completion

Written 2026-10-08 by the orchestrator from the owner's ask ("Please complete the backlog") in an unattended `/loop` run. The ask is the trigger for every bounded item that PLAN.md listed as "gated on a trigger" (D118); decisions in §9 are the orchestrator's, each "owner to confirm". Items that need the owner or an external resource stay listed (§8).

## 1. Intent

The backlog empties: every bounded defect and improvement listed in PLAN.md on 2026-10-08 is built, tested and landed, so the next phase starts from a clean list.

## 2. Items and what "done" means

| # | Item (PLAN.md backlog) | Done when |
|---|---|---|
| B1 | Per-IP limits in memory per process | Buckets for `POST /auth/magic-link` and `POST /auth/oidc/{provider}/start` live in Postgres (migration `0008`), so N API processes share one 300-per-hour count per client address; unit test with two limiter instances on one database; the per-email limit moves the same way |
| B2 | Outbox delivery one mail at a time | `UNICOURSE_OUTBOX_SENDERS` (default 1) runs that many sender goroutines, each claiming with `FOR UPDATE SKIP LOCKED`; a test with 2 senders delivers 20 mails exactly once |
| B3 | Pickers read one page of 200 | `GET` departments, terms and courses accept `q` (case-insensitive prefix or substring on code and name, contract updated); the department picker, term select and course picker in the web show a search field when the list has more than 200 rows (or always, if simpler) and query with `q` |
| B4 | CLI runs as the owner role | A `unicourse_platform` role (migration `0009`, `docker/initdb.sql`) can run `create-university` but not `migrate`; the CLI reads `UNICOURSE_PLATFORM_DATABASE_URL` for `create-university` when set; README and `docs/` say which role runs what |
| B5 | Only `X-Forwarded-For`; IPv4-mapped CIDR never matches | `Forwarded: for=` (RFC 7239) is read with the same trust walk; `::ffff:a.b.c.d` client addresses and prefixes are unmapped before matching; unit tests for both |
| B6 | Invitation 409 reveals membership state | One generic 409 message for "already a member" and "pending invitation" ("This email already has access or a pending invitation."), web copy follows |
| B7 | 100 MB boundary test out of CI | The upload limit comes from `UNICOURSE_MAX_UPLOAD_BYTES` (default 104857600); the boundary test runs in CI with a small limit (`-short` no longer skips it) |
| B8 | Spec 7 §11 deferred items | `seed-demo --fixture <path>` loads another dataset; `docs/seed/fixture.schema.json` describes the fixture (a Go test checks `demo.json` against the schema's required keys, no new dependency); a root `Makefile` with `make dev` (Postgres up, migrate, seed-demo, API and web dev servers); `screens.ts` seeds through `seed-demo` (D108 trigger taken) |
| B9 | Upload allowlist copied in the web | The API exposes the allowed extensions and the byte limit in the contract (`GET /api/v1/upload-policy`, public to signed-in members); the web reads it instead of its copy |
| B10 | `config` accepts a malformed `UNICOURSE_WEB_ORIGIN` | Startup fails with a clear message unless the value is `scheme://host[:port]` with no path |
| B11 | Two-origin dev setup not covered by CI | A CI step starts the API on 8080 and the Vite dev server on 5173, requests a magic link through the proxy and checks the confirm page's CSP `form-action` lists the web origin |
| B12 | P16-L1 findings F3, F4, F5, F6, F10 | F3: meetings in one section may not overlap on the same day (API 400 field error on `meetings`, editor message); F4: the section Remove dialog says "This section has final grades and cannot be deleted."; F5: course conflict copy says "already exists in this university"; F6: a department admin on `/admin/departments` or `/admin/terms` sees no create, edit or delete controls; F10: the audit "Changed by" filter lists names (email as the fallback) |
| B13 | Web index chunk 546 kB; upload progress; ESLint 10 | Routes are code-split (`lazy` per feature) so the index chunk is under 300 kB; material upload uses XHR with a byte-level progress bar; ESLint 10 stays gated until `eslint-plugin-jsx-a11y` peers on it (checked 2026-10-08: `^9` only) |

## 3. Out of scope (stay in the backlog with the owner)

Branch protection (D120, plan), OpenSpec (D121, owner says "openspec"), release build artifacts (D115, needs a deployment target), the owner's local Node version (22.11 below the 22.22.2 floor).

## 4. Constraints

- Stdlib-only API (D1); no new Go module. No new web runtime dependency; dev dependencies only if a lane's brief allows one.
- Every API change updates `apps/api/openapi.json` and the contract test; every web change keeps the e2e suite green (89 cases); new behaviour gets a unit test first.
- Spec 3 UI rules hold for every web change; the e2e suite is the arbiter for copy, so a copy change (B6, B12) updates the e2e assertion in the same PR under D80's rule (the lane names the exact selector).
- Hot spots stay single-lane per the Method: `routes.go`, `cmd/api/main.go`, `migrations/` (numbers assigned per lane), `openapi.json`, `ci.yml`, `package.json`.

## 5. Gates

Go: gofmt, vet, package tests then the full suite (no SKIP). Web: lint, typecheck, unit tests (CI), build, the e2e files the change touches on the lane's own port and database, the full suite on main at phase close. Real-process checks by the orchestrator for B1, B2, B4, B5, B10, B11.

## 6. Internal layout

Design `docs/design/P17-L1.md` (API shapes and migrations for B1, B2, B3, B4, B5, B7, B9, B10, B12-F3); worker lanes per the plan.

## 7. Testing

Per item in §2; the designer names the test for each.

## 8. Needs the owner

§3 items, and confirmation of the decisions below.

## 9. Decisions needing the owner's call

Recorded in PLAN.md as D130 to D134, each "Orchestrator 2026-10-08, owner to confirm":

- D130: "Please complete the backlog" is the trigger for every bounded backlog item (B1 to B13); owner-gated items stay (§3).
- D131: F3 rule: meeting rows in one section may not overlap on the same day.
- D132: invitation 409 copy is generic; the upload policy is a contract route `GET /api/v1/upload-policy`; the upload limit is `UNICOURSE_MAX_UPLOAD_BYTES`; outbox parallelism is `UNICOURSE_OUTBOX_SENDERS`.
- D133: a `unicourse_platform` database role for `create-university`; the CLI uses `UNICOURSE_PLATFORM_DATABASE_URL` when set, else the owner URL as today.
- D134: migration numbers: `0008` rate buckets (P17-L2), `0009` platform role (P17-L7).
