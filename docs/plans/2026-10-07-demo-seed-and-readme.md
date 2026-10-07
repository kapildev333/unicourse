# Demo Seed and Root README — Implementation Plan

> **For agentic workers:** executed by `/orchestrate` (orchestrator + `designer` → `worker` → `reviewer` lanes, one branch and worktree per lane). Each task is one lane. Global constraints, gates and git rules are those of `docs/plans/2026-10-05-platform-core-and-staffing.md`. Tests need `UNICOURSE_TEST_DATABASE_URL` and `UNICOURSE_TEST_OWNER_DATABASE_URL` (D48) and, for the seed test, a writable `UNICOURSE_FILES_DIR` (a `t.TempDir()`).

**Goal:** one command loads a full demo university into a local database, and the root README takes a newcomer from clone to a signed-in demo (Spec 7 §1).

**Architecture:** `internal/seed` loads an embedded JSON fixture and writes it through the real handlers in process (`app.NewRouter` over `httptest` with session cookies minted in `sessions`), the pattern of `internal/grade/fixture_test.go`; `unicourse-admin seed-demo` is the entry point. The README documents what the seed prints.

**Tech Stack:** as Spec 1 §3; no new Go module, no new npm dependency, no migration, no contract change.

**Spec:** `docs/specs/2026-10-07-demo-seed-and-readme.md`; every task cites its sections.

## Global Constraints

- Go 1.27 standard library; allowed modules only (Spec 1 §3). `encoding/json` and `embed` for the fixture.
- Every seed write goes through a handler; the only direct SQL is: university row (as `create-university` does), users with display names, memberships, sessions, the `--reset` delete, and the "slug exists" check. The designer may add a direct write only when no handler exists for it and says so in the design.
- Fixture is the single source of names and numbers (spec §6); `<year>` placeholders substituted at load.
- No mail queued (spec §4); no outbox rows after a seed run.
- Test-driven: failing test, implementation, pass, commit, for each step. Gates: `gofmt -l .`, `go vet ./...`, `go test ./... -count=1 -race` on `internal/seed` and `cmd/unicourse-admin`; the README lane runs the quick start verbatim and pastes the transcript.
- Naming per `CLAUDE.md`: package `seed`, command `seed-demo`, flags `--slug`, `--reset`; domain words only (never "class", "tenant", "teacher") in the fixture and README.

## Review Focus

- Fidelity: every write the API would audit is audited; a persona's write is made by that persona's session.
- `--reset` cannot touch another university or another university's files.
- The fixture round-trips: editing a name in `demo.json` changes the seeded data with no Go change.
- README: every command was run as written; no section repeats `apps/api/README.md` where a link does.

## File Structure

```
apps/api/internal/seed/seed.go        Load(fixture []byte) (Fixture, error); Run(ctx, db, files, baseURL, Fixture, Options) (Report, error)
apps/api/internal/seed/demo.json      the Northfield dataset (spec §5)
apps/api/internal/seed/seed_test.go   counts, personas, reset, no outbox rows
apps/api/cmd/unicourse-admin/main.go  seed-demo case, help text
apps/api/cmd/unicourse-admin/main_test.go usage rows
apps/api/README.md                    "Load the demo" section
README.md                             root README (spec §7)
docs/screens/*.png                    six captures
apps/web/README.md                    screens script, link to root
```

## Phase 11 — Demo seed and root README

### Task 1 (P11-L1, D): seed command

Spec §3 to §6, §8, §9, §10. Designer writes `docs/design/P11-L1.md`: the fixture JSON schema (every key, with the request body it maps to), the order of writes and which persona makes each, how sessions are minted and cleaned up (delete the seed's sessions at the end so no cookie outlives the command), the `--reset` sequence (files directory after the row delete, within what transaction boundary), the printed report format, error handling (first failing write aborts; what is left behind and the message that says to rerun with `--reset`), and the test list. Worker builds it test-first in `internal/seed` and `cmd/unicourse-admin`, adds the README section, pastes gates. Reviewer checks the Review Focus list and re-runs gates.

Scope: `docs/design/P11-L1.md`, `apps/api/internal/seed/**`, `apps/api/cmd/unicourse-admin/**`, `apps/api/README.md` (one new section only). Not `apps/api/internal/app/app.go` unless `Deps` needs a field; report, do not edit, anything else.

### Task 2 (P11-L2, W): root README and screenshots

Spec §7, §9 (README row). Starts once `docs/design/P11-L1.md` is on its branch (the command name, flags and report format are fixed there) and lands after Task 1 so the quick start can be run verbatim. Worker: write `README.md`; run `npm run screens` on a seeded database, pick the six captures named in spec §7.2, copy to `docs/screens/`; update `apps/web/README.md`; run the quick start from a dropped volume and paste the transcript in the lane log. Reviewer: read the README as a newcomer, re-run the quick start, check links and the screenshot sizes.

Scope: `README.md`, `docs/screens/**`, `apps/web/README.md`. Nothing under `apps/api`.
