# Spec 7: Demo seed and root README

Written 2026-10-07 by the orchestrator under D8 delegation ("choose the best or recommended"); the owner confirms by leaving the decisions in §12 standing. Brainstormed on the architectural path: intent, approaches with trade-offs, design by section, self-reviewed for placeholders, contradictions and scope.

## 1. Intent

The owner wants two things when the repository is opened from GitHub:

1. A demo dataset that can be loaded into a fresh local database with one command, so every page of the app is full and every role can be tried without building data by hand. "Fixtures" means the dataset is a readable file that can be edited, not code.
2. A root `README.md` that gives a newcomer everything: what the product is, what it looks like, how to run it, how to load the demo, how to sign in as each persona, how to test, where the specs, plans and designs are, and how the work is organised.

Success: on a clean clone with Docker and Go and Node installed, following the README from top to bottom ends with the demo university open in the browser as any of the personas, in under ten minutes, with no step that needs knowledge outside the README.

Who it is for: the owner demonstrating the product, a new engineer joining, and the orchestrator's own verification runs (a seeded database beats a hand-built one for browser passes).

## 2. Out of scope

- Production data, migration of real universities, multi-university demo sets.
- Replacing the capture seed (`apps/web/e2e/screens.ts`) or the e2e suite's own seeding; they keep working as they are (D108). Making them call the seed command is backlog.
- A `.env` loader in the API (environment only, Spec 1 §9 stands); the README gives a copy-paste block instead.
- Localisation of the dataset; one English-language university.
- A Makefile, task runner or dev container. Two app READMEs plus the root one are the tooling.

## 3. Approaches considered

| Approach | Fidelity | Needs | Verdict |
|---|---|---|---|
| A. `unicourse-admin seed-demo` in Go, writes through the real HTTP handlers in process (`app.NewRouter` over `httptest`, session cookies created straight in `sessions`), exactly as `internal/grade/fixture_test.go` and `internal/content/fixture_test.go` already do | Full: every validation, audit row, pending table and blob write is the API's own | Postgres and `UNICOURSE_FILES_DIR`; no running API | **Chosen** |
| B. `unicourse-admin seed-demo` writing plain SQL as the owner role | Low: duplicates invariants (audit rows, composite FKs, score rules); drifts at every migration | Postgres only | Rejected: the test fixtures show the SQL route needs re-deriving every rule |
| C. A Node script driving a running API over HTTP (lift the capture seed out of `screens.ts`) | Full | Running API, Playwright, magic-link scraping from the log | Rejected: three moving parts for a one-command demo; ties a dev task to the e2e harness |

Fixture format: one JSON file, not Go literals (the owner asked for editable fixtures), not CSV (only sections and rosters have a CSV importer; the rest would need a second format), not YAML (no parser in the allowed modules, D2).

## 4. The command

```
UNICOURSE_OWNER_DATABASE_URL=... UNICOURSE_FILES_DIR=... UNICOURSE_BASE_URL=... unicourse-admin seed-demo [--slug northfield] [--reset]
```

- Loads the embedded fixture (`apps/api/internal/seed/demo.json`), creates the university with the fixture's name, slug (override `--slug`), allowed domain (`<slug>.edu`), timezone and locale, then writes everything else through the real handlers.
- Exit 2 with a one-line reason when the slug already exists and `--reset` is absent; usage and validation errors exit 2 before any database connection, like the other commands.
- `--reset` deletes the university with that slug (one `delete from universities where slug = $1` as owner; every tenant table cascades) and its files directory `<files>/<university_id>/`, then seeds. It never touches another university.
- On success prints the base URL, then one line per persona: role, display name, email, and what they see ("instructor of CS101, CS210, MATH201"). Sign-in is the real flow: the README tells the operator to request a magic link on `/sign-in`, which the dev mailer prints in the API log (`UNICOURSE_SMTP_URL` empty).
- Idempotent per slug with `--reset`; running it twice without `--reset` is the exit 2 above.
- No mail is queued by the seed: announcements are posted with `email: false`; invitations for the pending persona are created the way `create-university` does (no outbox row, D28).
- Does not run migrations; a database behind the binary fails with the same message as the API (run `migrate` first).

## 5. The dataset

Northfield University (`northfield`, `northfield.edu`, `America/New_York`, `en-US`), the same world as the capture seed so screenshots and demo agree (D105):

- Departments: Computer Science (CS), Mathematics (MATH), Economics (ECON), Biology (BIO).
- Terms: the current academic year (`Academic Year <year>`, Jan 1 to Dec 31, so it is "current" on any date) and `Spring <year+1>`. Term names and dates in the fixture carry `<year>` and `<year+1>` placeholders that the seed substitutes with the current year; everything else is literal.
- Courses and sections: CS101 Introduction to Computer Science (4 cr), CS210 Data Structures and Algorithms (4), MATH201 Linear Algebra (4), ECON110 Principles of Microeconomics (3), BIO150 Cell Biology (4); one section `001` each in the academic-year term with the two weekly meetings and rooms of the capture seed; CS101 also has a section `002` in Spring (empty, shows an open card and the term filter).
- People and roles: Renata Quinn (admin, `registrar@northfield.edu`), Owen Park (department admin of CS, `owen.park@`), Elena Marsh (instructor: CS101, CS210, MATH201), David Mensah (instructor: ECON110, BIO150), Sofia Alvarez (TA on CS101), eight students (Amara Okafor, Priya Raman, Tomasz Zielinski, Daniel Reyes, Mei-Ling Chou, Jonas Berg, Fatima Al-Sayed, Lucas Moreau) enrolled as in the capture seed (Amara in three of five), and Ingrid Larsen as a pending invitation never signed in. Every member has a display name (users row with `display_name` set); students enter through the roster import so `pending_enrollments` and the roster paths are exercised.
- Grading: the US plus/minus 4.0 scale, written through `PUT /grade-scale` only when the new university has no rows (as the capture seed does), two published assessments per section with weights 40/60 and the capture seed's deterministic scores, one missing final score per section; CS101 finalised with one override and a reason so Final grades, GPA and the audit log have content; CS210 left open.
- Content: every section has a description; CS101 has three announcements (one from the TA), three weeks with notes, and the three files from `apps/web/e2e/fixtures` (`syllabus.pdf`, `slides.pptx`, `code-samples.zip`) attached as materials; the other sections have one announcement and one week each.
- Audit log: whatever the writes above produce (no synthetic rows).

The fixture file lists all of this literally; the design (§10) fixes its exact shape. Names, emails and codes must stay those of the capture seed where both have them.

## 6. Fixture file

- `apps/api/internal/seed/demo.json`, embedded with `embed`, validated on load (unknown keys rejected, every reference by code or email resolved before the first write), decoded with `encoding/json` into Go structs that mirror the API request bodies where one exists (so a fixture edit that breaks a rule fails with the API's own field error).
- A second fixture is possible later by adding a `--fixture <path>` flag; not built now (YAGNI, §13).
- The file is the single source of the dataset: no names or numbers in Go code except the placeholder substitution.

## 7. Root README

`README.md` at the repository root, in this order, each section short and copy-paste ready:

1. One paragraph: what UniCourse is (multi-tenant platform for universities: departments, terms, courses, sections, staff, rosters, schedule, grading, section content) and the stack in one line.
2. Screenshots: eight 1440-wide captures, four views in light and dark pairs, committed under `docs/screens/` (D107): My sections (student, schedule view), Catalog, CS101 Materials (the capture script has no section Home capture; amended 2026-10-07 after P11-L2), Gradebook (instructor view). Taken on the `seed-demo` dataset so the README shows what a newcomer gets; `npm run screens` seeds its own university, so until it gains a `seed-demo` mode (§11) the captures are taken with a throwaway Playwright script outside the repo; total under 2 MB (PNG, left as captured).
3. Quick start: prerequisites (Docker, Go 1.27, Node 22), then the exact commands: compose up, env block, migrate, `seed-demo`, run API, run web, open `http://localhost:5173`, request a magic link for a persona, copy it from the API log. One block per step, nothing the reader must infer.
4. Demo personas: the table printed by `seed-demo` (role, name, email, what to look at).
5. Tests and gates: the Go gates, the web gates, e2e, screens, and the one-integration-suite-at-a-time rule.
6. Repository layout: the tree from `CLAUDE.md` with one line each, plus `docs/screens/`.
7. Documentation index: every spec with its one-line topic and its plan, the design documents directory, `PLAN.md` as the live plan, `CLAUDE.md` as the conventions.
8. How the work is organised: the orchestration workflow in five sentences (plan, lanes, worktrees, PRs with green ci, PLAN.md decisions), linking to `.claude/commands/orchestrate.md`.
9. Deployment notes in one paragraph linking to `apps/api/README.md` (single binary serving `dist/`, database roles, files directory, migrate then roll).
10. Status: which specs are implemented, what is pending (link to the backlog section of `PLAN.md`).

Rules: no claims the README lane did not run itself (every command in §7.3 is executed on a clean database before the PR); no copy that drifts from `apps/api/README.md` (link rather than repeat the env table; the quick start may inline the minimal env block); plain Markdown, GitHub-flavoured, no HTML except the screenshot table if alignment needs it.

`apps/web/README.md` gains the `screens` script and a link up to the root README; `apps/api/README.md` gains a `seed-demo` section written by the seed lane (the README lane links to it).

## 8. Authorization and safety

- The command runs as the owner role (like every `unicourse-admin` command, D48) and uses the API's handlers with real session cookies for personas, so every write is authorised the way a user's would be (the admin creates departments, the instructor creates assessments, the TA posts an announcement). A fixture that asks a persona to do what their role cannot fails with the API's 403.
- `--reset` is the only destructive path and is scoped to one slug. The README says in one line that the seed is for local databases.
- No new environment variables. `UNICOURSE_FILES_DIR` is required because materials are written.

## 9. Testing

- `internal/seed`: a Go test seeds the test database from the embedded fixture through the same code path and asserts counts per table (universities 1, departments 4, courses 5, sections 6, memberships, section_staff, enrollments, assessments 10, scores, final_grades for CS101, announcements, weeks, materials 3 with files on disk), the personas' display names, the pending invitation, and that a second run without `--reset` exits 2 while `--reset` leaves the same counts and a different university id. Runs under the shared advisory lock like the other database tests; skipped under `-short` only if it exceeds ten seconds (it should not).
- `cmd/unicourse-admin`: usage and flag validation tests in the existing table (`seed-demo -h`, bad slug, missing env) that need no database.
- README: the lane runs the quick start verbatim on a dropped-and-recreated local volume and pastes the transcript into the PR; the orchestrator repeats it.
- Test-driven as every lane: the counts test and the usage test are written first and fail before the command exists.

## 10. Internal layout

```
apps/api/internal/seed/        seed.go (Load, Run), demo.json, seed_test.go
apps/api/cmd/unicourse-admin/  main.go gains the seed-demo case and help text; main_test.go the usage rows
apps/api/README.md             "Load the demo" section
README.md                      root README (§7)
docs/screens/                  six PNGs
apps/web/README.md             screens script, link to root
docs/design/P11-L1.md          the designer's rulings: fixture schema, persona-to-write mapping, error handling, output format
```

No new Go module, no new npm dependency, no migration, no contract change.

## 11. Deferred, with triggers

- `--fixture <path>` for a second dataset: when someone needs a demo that is not Northfield.
- The capture script and the e2e suite seeding through `seed-demo`, and a Materials capture in `screens.ts`: when either seed step changes for the third time, or when the README captures need retaking.
- A JSON Schema for the fixture: when a non-engineer edits it.
- A `make dev` or dev container: when the README quick start exceeds one screen.

## 12. Decisions needing the owner's call

Recorded in `PLAN.md` as D104 to D108, each "owner to confirm":

- D104: CLI command, embedded JSON fixture, writes through the in-process real handlers (approach A).
- D105: dataset is the capture seed's Northfield University, extended with Phase 10 content and a finalised section.
- D106: `--reset` scoped to one slug; sign-in stays the real magic-link flow, links read from the API log.
- D107: eight screenshots (four views, light and dark) committed under `docs/screens/`.
- D108: capture script and e2e keep their own seeding.
