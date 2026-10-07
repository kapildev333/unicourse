# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this directory is

UniCourse: a multi-tenant platform for universities to host courses, manage classes and sections, and link professors and TAs. Product/startup, not a toy. No product code yet, so no build, lint, or test commands; add them here as soon as the stack exists. ## Layout and naming

```
apps/api/     Go API — net/http standard library only (no framework, router, ORM or DI); allowed modules: pgx, x/oauth2, go-oidc, x/crypto
apps/web/     React + Vite + TypeScript web app
docs/specs/   design specs, YYYY-MM-DD-<topic>.md, approved before any plan or code (not docs/superpowers/specs/)
docs/plans/   implementation plans, one per spec, same file name; each task is one lane (not docs/superpowers/plans/)
docs/design/  designer-lane output, <LANE-ID>.md, written before the worker lane builds that task
.claude/      agents, lane logs, /orchestrate
```

One convention per layer, no exceptions:

| Layer | Convention | Example |
|---|---|---|
| Directories, non-Go files | lowercase kebab-case | `apps/web/src/features/my-sections/` |
| Go packages | one lowercase word, no underscores, named for what they hold | `internal/course`, `internal/httpx` |
| Go identifiers | Go standard (MixedCaps); exported only when another package needs it | `course.Section`, `pg.WithTx` |
| TS files | kebab-case; React components PascalCase export in kebab-case file | `section-table.tsx` exports `SectionTable` |
| Database | snake_case, plural tables, `<table>_id` foreign keys, `created_at`/`updated_at` on every table | `section_staff.section_id` |
| HTTP routes | `/api/v1/<plural-kebab-noun>`, ids as path params, no verbs in paths | `GET /api/v1/sections/{id}/staff` |
| JSON fields | camelCase, matching the Go struct tag and the TS type exactly | `universityId` |
| Env vars | `UNICOURSE_` prefix, UPPER_SNAKE | `UNICOURSE_DATABASE_URL` |
| Domain terms | one word per concept everywhere: university (tenant), department, term, course (catalog), section (course in a term), staff (instructor or TA on a section), enrollment (student in a section) | never "class", "tenant", "org", "teacher" in code |

## Orchestration workflow

The only shared tooling is a plan-driven multi-agent workflow under `.claude/`:

- `/orchestrate <source documents>` (`.claude/commands/orchestrate.md`) makes the main session the orchestrator. It reads `PLAN.md`, or proposes one from the source documents and stops for discussion before launching anything.
- `PLAN.md` at the root is currently an unfilled template (placeholders in `<angle brackets>`): a decisions table, method rules, phases cut into lanes, backlog, and journal.
- The orchestrator is the main session (Fable 5.1). Lanes are sub-agents with disjoint directory scopes, run in this order per piece of work:
  - `designer` (Opus 5.5) writes the design document when a brief needs design judgement; no product code.
  - `worker` (Sonnet 5.5) implements the brief against that design; stops and reports rather than making an open design choice.
  - `reviewer` (Opus 5.5) reviews the landing read-only against brief and design, re-running gates itself.

Shared memory between lanes lives in `.claude/lanes/<LANE-ID>.md` (format in `.claude/lanes/README.md`): each lane writes only its own file, reads the others at start and finish, and addresses requests to other lanes there. The orchestrator reads all of them, relays urgent requests (SendMessage to a running lane, otherwise in the next brief), and folds anything durable into `PLAN.md`.

Rules that span these files:

- Only the orchestrator edits `PLAN.md`. Lanes escalate plan-worthy decisions in their report.
- Lanes work only inside the paths their brief lists; anything needed outside is reported, not edited.
- One directory has one owner at a time, and only one integration suite runs at a time (orchestrator included).
- The orchestrator verifies every landing itself (re-run suites, then exercise the real database/HTTP/browser) and records the evidence in the lane's status cell. A lane's own report is never verification.
- The orchestrator does not write features itself while a lane could.
- Nothing lands on `main` except through a pull request with green `ci`. GitHub does not enforce this server-side yet (branch protection and rulesets need GitHub Pro or a public repo), so the rule is process-enforced by the orchestrator; enable protection with required checks `api` and `web`, linear history and admins included as soon as the plan allows.

## Git and GitHub

- One branch per lane, `lane/<LANE-ID>-<slug>`, in its own worktree under `.worktrees/<LANE-ID>` (gitignored) so parallel lanes never share a checkout. Non-lane changes use `chore/<slug>`.
- Lanes commit on their branch and never push. The orchestrator pushes, opens the PR from `.github/pull_request_template.md`, waits for `ci`, squash-merges with branch deletion, removes the worktree.
- `ci` (`.github/workflows/ci.yml`): `api` job runs gofmt, `go vet`, `go test` against a Postgres service; `web` job runs lint, typecheck, tests, build. Both no-op until the app is scaffolded, so the required checks stay green on an empty tree.
- Dependabot updates Go modules, npm and Actions weekly; its PRs go through the same `ci`.
- Not used on purpose: GitHub Projects and Issues for planning (`PLAN.md` is the single plan; mirroring it is double bookkeeping), Releases (nothing deploys yet), Wiki and Discussions.

## UI conventions (Spec 3, binding for every web change)

The web app follows `docs/specs/2026-10-06-visual-design-system.md` and the design `docs/design/P6-L1.md`; read both before touching `apps/web`. The look is Udemy-like, chosen by the owner (D57): white top bar with a catalog search, bold DM Sans headings, 4 px radius, cards for Catalog and My sections, tables for admin.

- Colours come only from the semantic tokens in `apps/web/src/index.css` (`canvas`, `surface`, `line`, `line-strong`, `line-control` (secondary button border), `ink`, `ink-muted`, `accent`, `accent-hover`, `accent-soft`, `on-accent`, `focus`, `danger`, `success`, `warning` and their `-soft` backgrounds). Dark mode is the same tokens redefined under `html[data-theme="dark"]` and `prefers-color-scheme`, so feature code never uses `dark:` variants, hex colours, `bg-white`/`text-white`, Tailwind palette utilities or colour functions. `apps/web/src/no-literal-colors.test.ts` fails the web gate on any of them.
- Purple is the single accent and sits only on actions, links, the current location and the card cover band. Status colours (danger, success, warning) mean error, success, caution and nothing else.
- Type is DM Sans alone (bundled through `@fontsource-variable/dm-sans`; no runtime request to Google): `h1` is 32 px bold by base rule, so pages set no size or weight on it; section headings are `text-xl font-bold`; no serif, no mono, no `font-semibold`.
- Icons are Lucide (`lucide-react`), 16 px by the provider in `routes/__root.tsx`, always `aria-hidden` next to visible text; icon-only controls carry `aria-label` and `title`.
- Pages compose the shared components in `apps/web/src/components/` (`Button` with `primary`/`secondary`/`ghost`/`ghost-neutral`/`ghost-danger` (row actions muted at rest, red on hover), `ButtonLink` for a link styled as a button, `Badge`, `Card`, `Dialog` + `DialogFooter` with Cancel first, `EmptyState`, `ErrorBanner`, `FormField` + `inputClass`/`inputSizedClass`/`fileInputClass`, `ListCount`, `Loading` skeleton with `table` and `cards` variants, `PageHeader`, `SegmentedControl`, `Disclosure`, `StatusMessage`, `Table` with `stack` for phone rows, `TabNav` + `tabLink`, `EntryLayout`, `ThemeSwitch`) and never define a one-off recipe. A new pattern goes into `components/` with a test and a line in the design document.
- Default Tailwind colours and shadows are switched off; `rounded-md`/`rounded-lg` are 4 px; the only shadows are `shadow-card` on card hover and `shadow-dialog`.
- Every page is checked in light and dark before a PR (the theme switch is in the header; `localStorage` key `unicourse:theme`), and every text pair stays at or above 4.5:1 and borders at 3:1 (the design's §1.3 script measures them).
- The e2e suite is the regression arbiter for copy and roles; only the Catalog and My sections selectors were allowed to change when those pages became cards (D60).
