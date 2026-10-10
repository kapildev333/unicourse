<div align="center">

# UniCourse

**A multi-tenant course platform for universities, built on the Go standard library and React.**
Departments, terms, courses and sections. Rosters, staffing, grading and GPA. Announcements and weekly materials. Light and dark.

[![ci](https://github.com/kapildev333/unicourse/actions/workflows/ci.yml/badge.svg)](https://github.com/kapildev333/unicourse/actions/workflows/ci.yml)
![Go](https://img.shields.io/badge/Go-1.27-00add8?logo=go&logoColor=white)
![net/http only](https://img.shields.io/badge/API-net%2Fhttp%2C%20no%20framework-00add8)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-pgx-4169e1?logo=postgresql&logoColor=white)
![React](https://img.shields.io/badge/React-Vite%20%2B%20TypeScript-61dafb?logo=react&logoColor=black)
![Playwright](https://img.shields.io/badge/E2E-Playwright-2ead33?logo=playwright&logoColor=white)
![OpenAPI 3.1](https://img.shields.io/badge/OpenAPI-3.1-6ba539?logo=openapiinitiative&logoColor=white)

<img src="docs/screens/my-sections-light.png" alt="My sections, schedule view" width="900">

</div>

## Highlights

- **One tenant per university.** Every URL is scoped (`/northfield/...`); departments, terms, courses, sections and people never leak across tenants.
- **Real roles.** University admin, department admin, instructor, TA, student and invited users, enforced server-side on every route.
- **Passwordless sign-in.** Magic links out of the box; Google and Microsoft OIDC wired in.
- **Grading that holds up.** Weighted components, scores, overrides, finalisation, GPA, all with an audit trail.
- **Section home.** Announcements and weekly materials with file uploads, for students and staff.
- **Contract first.** OpenAPI 3.1 checked against the routes in CI, a generated Postman collection, typed web client.
- **No framework on the API.** `net/http`, `pgx`, `x/oauth2`, `go-oidc`, `x/crypto`. That is the whole allow-list.
- **Tested end to end.** Go race tests against Postgres, Vitest on the web, Playwright through a real browser, API and database.
- **Light and dark**, with a documented design system.

<details>
<summary><b>More screenshots</b> (light left, dark right)</summary>

Northfield University, the demo dataset loaded by `seed-demo`: Catalog and Materials as the student Amara Okafor, the Gradebook as the instructor Elena Marsh.

| Light | Dark |
|---|---|
| ![My sections, schedule view](docs/screens/my-sections-light.png) | ![My sections, schedule view](docs/screens/my-sections-dark.png) |
| ![Catalog](docs/screens/catalog-light.png) | ![Catalog](docs/screens/catalog-dark.png) |
| ![Section materials](docs/screens/section-home-light.png) | ![Section materials](docs/screens/section-home-dark.png) |
| ![Gradebook](docs/screens/gradebook-light.png) | ![Gradebook](docs/screens/gradebook-dark.png) |

</details>

## Quick start

Prerequisites: Docker, Go 1.27, Node 22.22.2 or newer. The demo is for local databases only.

One command, from the repository root:

```
make dev
```

It starts Postgres, applies the migrations, replaces the demo university (`seed-demo --reset`), then (installing the web dependencies first on a fresh clone) runs the API on :8080 and the web app on :5173 until you press Ctrl-C (that stops both; Postgres keeps running, stop it with `docker compose -f apps/api/docker-compose.yml down`). Then open <http://localhost:5173> and sign in as in step 6 of the fold below (the only step you need after `make dev`). `make test` runs the API and web gates and `make e2e` the Playwright suite.

<details>
<summary><b>Step by step, without make</b></summary>

Run the API steps from `apps/api`:

1. Start Postgres (creates the roles and databases on a fresh volume):

   ```
   cd apps/api
   docker compose up -d
   ```

   If port 5432 is taken (a local Postgres or an older UniCourse container), stop that first; compose reports "port is already allocated".

2. Set the environment (once per shell):

   ```
   export UNICOURSE_DATABASE_URL='postgres://unicourse:unicourse@localhost:5432/unicourse?sslmode=disable'
   export UNICOURSE_OWNER_DATABASE_URL='postgres://unicourse_owner:unicourse_owner@localhost:5432/unicourse?sslmode=disable'
   export UNICOURSE_BASE_URL=http://localhost:8080
   export UNICOURSE_WEB_ORIGIN=http://localhost:5173
   export UNICOURSE_SESSION_SECRET=dev-secret-dev-secret-dev-secret-00
   mkdir -p data/files
   export UNICOURSE_FILES_DIR=$PWD/data/files
   ```

3. Create the tables, then load the demo university:

   ```
   go run ./cmd/unicourse-admin migrate
   go run ./cmd/unicourse-admin seed-demo
   ```

   Add `--reset` to replace an existing demo, or `--fixture <path>` to load your own university from a JSON file instead of the built-in one (format: [docs/seed/fixture.schema.json](docs/seed/fixture.schema.json); material files are read from the fixture's directory). Details: [Load the demo](apps/api/README.md#load-the-demo).

4. Run the API (leave it running; its log shows the sign-in links you request):

   ```
   go run ./cmd/api
   ```

5. In a second terminal, from the repository root, run the web app:

   ```
   cd apps/web
   npm install && npm run dev
   ```

6. Open <http://localhost:5173>, go to Sign in, and request a magic link for a persona email below, for example `amara.okafor@northfield.edu`. No mail is sent: copy the link from the API log line with `"msg":"mail"` and open it in the browser, then press **Continue signing in** on the confirmation page. You land on `http://localhost:5173/northfield/me`; if you land on `http://localhost:8080/...` instead (a 404), the API was started without `UNICOURSE_WEB_ORIGIN` (the env block above sets it).

</details>

## Demo personas

`seed-demo` prints this table. Every persona signs in with a magic link as above. Start with **Amara Okafor** (student) or **Elena Marsh** (instructor).

<details>
<summary><b>All 14 personas</b></summary>

| Role | Name | Email | Sees |
|---|---|---|---|
| admin | Renata Quinn | registrar@northfield.edu | admin of Northfield University |
| department admin | Owen Park | owen.park@northfield.edu | department admin of CS |
| instructor | Elena Marsh | elena.marsh@northfield.edu | instructor of CS101, CS210, MATH201 |
| instructor | David Mensah | david.mensah@northfield.edu | instructor of ECON110, BIO150 |
| TA | Sofia Alvarez | sofia.alvarez@northfield.edu | TA on CS101 |
| student | Amara Okafor | amara.okafor@northfield.edu | student in CS101, CS210, MATH201 |
| student | Priya Raman | priya.raman@northfield.edu | student in CS101, CS210, MATH201, BIO150 |
| student | Tomasz Zielinski | tomasz.zielinski@northfield.edu | student in CS101, CS210, MATH201 |
| student | Daniel Reyes | daniel.reyes@northfield.edu | student in CS101, CS210, MATH201, BIO150 |
| student | Mei-Ling Chou | mei-ling.chou@northfield.edu | student in CS101, CS210, ECON110 |
| student | Jonas Berg | jonas.berg@northfield.edu | student in CS101, MATH201, ECON110 |
| student | Fatima Al-Sayed | fatima.alsayed@northfield.edu | student in CS101, MATH201, ECON110 |
| student | Lucas Moreau | lucas.moreau@northfield.edu | student in CS101, ECON110, BIO150 |
| invited | Ingrid Larsen | ingrid.larsen@northfield.edu | invitation pending, never signed in; TA on CS210 at first sign-in |

CS101 is finalised (with one override); CS210 is open. Ingrid's invitation expires after 14 days; `--reset` gives a fresh one.

</details>

## Explore the API with Postman

The API is described by `apps/api/openapi.json` (OpenAPI 3.1, checked against the routes in CI). A Postman
collection generated from it lives in `docs/postman/` (`npm run postman` in `apps/web` regenerates it; CI fails
when it is stale).

1. In Postman, import `docs/postman/unicourse.postman_collection.json` and `docs/postman/unicourse.postman_environment.json`, and select the "UniCourse local" environment.
2. With the API running and the demo seeded, send **auth / magic-link / Request a magic link** (it uses `{{email}}`, Elena Marsh by default).
3. Copy the token from the link in the API log (`.../auth/magic-link/<token>`) into the environment variable `token`.
4. Send **auth / magic-link / {token} / Consume a magic link**. Its test stores the session in `session`, and every later request sends it.
5. Try **universities/{slug} / me / My sections** or any other request. Path ids are blank: paste an id from a previous response.

## Tests and gates

API, from `apps/api` (needs `UNICOURSE_TEST_DATABASE_URL` and `UNICOURSE_TEST_OWNER_DATABASE_URL`, see [apps/api/README.md](apps/api/README.md)):

```
gofmt -l .
go vet ./...
go test ./... -count=1 -race
```

Web, from `apps/web`:

```
npm run lint
npm run typecheck
npm test -- --run
npm run build
```

End to end (Playwright, real API binary and Postgres, one-time `npx playwright install chromium`): `npm run e2e` in `apps/web`. Screenshots for review: `npm run screens` writes captures to `apps/web/e2e/.tmp/screens/`. Run only one integration suite at a time (they share the test databases). Details: [apps/web/README.md](apps/web/README.md).

## Repository layout

```
apps/api/     Go API: net/http only; allowed modules pgx, x/oauth2, go-oidc, x/crypto
apps/web/     React + Vite + TypeScript web app
docs/specs/   design specs, approved before any plan or code
docs/plans/   implementation plans, one per spec; each task is one lane
docs/design/  design documents per lane, written before the worker builds
docs/screens/ README screenshots (light and dark pairs)
.claude/      agents, lane logs, /orchestrate
```

Naming and domain terms (university, department, term, course, section, staff, enrollment) are in [CLAUDE.md](CLAUDE.md).

## Documentation index

| Spec | Topic | Plan |
|---|---|---|
| [1](docs/specs/2026-10-05-platform-core-and-staffing.md) | Platform core, sign-in, courses, sections, staffing, rosters | [plan](docs/plans/2026-10-05-platform-core-and-staffing.md) |
| [2](docs/specs/2026-10-06-student-schedule.md) | Student course view and schedule | [plan](docs/plans/2026-10-06-student-schedule.md) |
| [3](docs/specs/2026-10-06-visual-design-system.md) | Visual design system, light and dark | [plan](docs/plans/2026-10-06-visual-design-system.md) |
| [4](docs/specs/2026-10-07-grading.md) | Grading, scores and GPA | [plan](docs/plans/2026-10-07-grading.md) |
| [5](docs/specs/2026-10-07-ui-ux-benchmark.md) | UI/UX benchmark and improvement plan | [plan](docs/plans/2026-10-07-ui-ux-benchmark.md) |
| [6](docs/specs/2026-10-06-section-content.md) | Section home, announcements, weekly materials | [plan](docs/plans/2026-10-06-section-content.md) |
| [7](docs/specs/2026-10-07-demo-seed-and-readme.md) | Demo seed and this README | [plan](docs/plans/2026-10-07-demo-seed-and-readme.md) |

Lane design documents are in [docs/design/](docs/design/). [PLAN.md](PLAN.md) is the live plan (decisions, phases, journal); [CLAUDE.md](CLAUDE.md) holds the conventions.

## How the work is organised

Work is plan-driven: `PLAN.md` lists numbered decisions and phases cut into lanes. Each lane is a sub-agent (designer, worker, reviewer) with its own directory scope, a branch `lane/<LANE-ID>-<slug>` and a git worktree under `.worktrees/`. Lanes commit and never push; the orchestrator re-runs the gates, exercises the real database, HTTP and browser, then opens the pull request. Nothing reaches `main` except through a pull request with green `ci`. Decisions go into the `PLAN.md` decisions table, and lanes share notes in `.claude/lanes/`. See [.claude/commands/orchestrate.md](.claude/commands/orchestrate.md).

## Deployment notes

One Go binary serves the API and the built web app (`UNICOURSE_WEB_DIR`), against Postgres with two non-superuser roles (owner for migrations, API role for the process) and one shared files directory (`UNICOURSE_FILES_DIR`). Run `unicourse-admin migrate` first, then roll the binaries. Environment table, roles, files and backups: [apps/api/README.md](apps/api/README.md).

## Status

Implemented: Spec 1 (platform core, sign-in, courses, sections, staffing, rosters), Spec 2 (student My sections and schedule), Spec 3 (design system, light and dark), Spec 4 (grading, scores, GPA), Spec 5 (UI/UX benchmark and the Phase 9 improvements), Spec 6 (section home, announcements, weekly materials), Spec 7 (demo seed and this README), Spec 8 (Postman collection generated from the OpenAPI contract). Also in progress: P4-L4 (live Google and Microsoft sign-in credentials). Pending work is in the [PLAN.md backlog](PLAN.md#backlog).

Each completed phase is summarised in the [releases](https://github.com/kapildev333/unicourse/releases).

## Contributing

This is a personal project and does not accept pull requests or issues. Fork it or download it freely.
