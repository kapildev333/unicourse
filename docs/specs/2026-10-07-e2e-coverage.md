# Spec 9: End-to-end coverage with Playwright

Written 2026-10-07 by the orchestrator from the owner's ask ("Design playwright AI automation test cases for our code base and do all e2e test cases"), in an unattended `/loop` run: decisions in §9 are the orchestrator's, each "owner to confirm" (D122 to D126). Test-only change: nothing under `apps/web/src` or `apps/api` changes except where a test exposes a real defect, which is then a separate fix lane.

## 1. Intent

Every user-visible flow of UniCourse, for every role, is exercised by a Playwright test that runs on `npm run e2e` and in CI, so a regression anywhere in the product fails `ci` before it reaches `main`. The cases are designed from the specs (Spec 1 to Spec 8), the README persona table and the running UI by the designer lane, written down as a catalogue with stable IDs, and then implemented as plain `@playwright/test` files. "AI automation" means the cases are derived and built by the lanes; the runner is Playwright alone (D122).

## 2. What exists

- `apps/web/e2e/` runs one API binary serving the built web app on port 18180 against `unicourse_e2e` (D33: never reset, unique slugs, domains and emails per run; sign-in links read from the API's mail log). `workers: 1`, retries 1 in CI, traces on failure.
- Three spec files, all green on `main` (v0.14.1): `staffing.spec.ts` (one flow: two universities, department, term, course, section, staffing, roster import, schedule clashes, grading end to end), `content.spec.ts` (description, announcements, weekly materials, downloads, access), `theme.spec.ts` (dark theme survives reload and sign-out).
- `screens.ts` (not in `npm run e2e`) seeds a full Northfield university through the HTTP API with `page.request` and a `Sec-Fetch-Site: same-origin` header, and captures screens.
- `unicourse-admin seed-demo --slug <s>` loads the Northfield dataset (14 personas, five courses, two terms, CS101 finalised, one pending invitation) under any slug; emails use `<slug>.edu`.
- CI `e2e` job (`ci.yml`): Postgres service, `npm run e2e`, 20-minute timeout, report and API log uploaded on failure.
- Rulings that still hold: D60 and D80 (the existing spec files change only by an explicit ruling), D99 (new areas get their own spec file), D108 (the capture script keeps its own seeding).

## 3. Deliverables

1. **Test-case catalogue** `docs/design/P16-L1.md`: every case with an ID, the role, preconditions, steps, the assertion in the words of the spec it covers, and the spec file that implements it; the three existing files are mapped to IDs too, so the catalogue is the complete inventory. Gaps the designer finds that no test can cover (OIDC with a real provider, SMTP delivery, the 100 MB upload boundary) are listed as such with the reason.
2. **Harness** (`e2e/helpers.ts`, `e2e/env.ts`, `playwright.config.ts`, `global-setup.ts`): the port comes from `UNICOURSE_E2E_PORT` (default 18180) so parallel lanes and the orchestrator can each run the suite on one machine; `api`, `upload` and a `seedDemo(slug)` helper live in `helpers.ts` (moved out of `screens.ts`, which imports them); `signIn` accepts the page a sign-in lands on when a `next` redirect is in play.
3. **Spec files**, one per area, each creating its own university or `seed-demo` slug per run, so any file runs alone and the files are independent of each other and of order:

| File | Area | Roles |
|---|---|---|
| `demo.spec.ts` | `seed-demo` smoke: every README persona signs in and sees exactly the "Sees" column of the README table (the README is the oracle) | all |
| `auth.spec.ts` | sign-in page validation, unknown address gives the same "Check your email" (no enumeration), used, expired and malformed magic-link tokens, sign-out, a deep link while signed out returns to it after sign-in, wrong slug, a member of A on B's URLs, invitation accept (valid, used, expired), the 5-per-email limit message | member, outsider |
| `admin.spec.ts` | departments create, rename, delete and the blocked delete; terms create, edit, status, delete; members list, role change, remove; invitations list, revoke, pending state; audit log entries for each of the above with filters; a department admin's narrower view | admin, department admin |
| `courses.spec.ts` | course create, edit, delete; CSV import preview errors and commit for courses and sections; section create with meeting validation (end before start, overlapping rows), edit, delete; staff add by email, pending invitation remove, staff remove; roster remove; Participants tab per role | admin, department admin, instructor, TA, student |
| `grades.spec.ts` | grade scale editor validation and save; assessment create, edit, delete; weights not totalling 100 block Finalize with the reason; score cell validation (over max, not a number, blank); scores CSV preview errors; publish; finalize; override; unfinalize; My grades with unpublished count; student never sees the gradebook; TA permissions as specified | instructor, TA, student |
| `catalog.spec.ts` | Catalog search, department and term filters, Clear filters, Enrolled badge, empty states ("Set up terms" for an admin, none for a student); My sections list and schedule, term picker, Today and the phone agenda at 390 px | admin, student |
| `content-edit.spec.ts` | announcement edit and delete; week rename, reorder if specified, delete; material rename, move, delete; upload rejected for a disallowed extension; student sees none of the editing controls | instructor, TA, student |
| `a11y.spec.ts` (D125) | `@axe-core/playwright` sweep of every page per role in light and dark; serious and critical violations fail | all |

4. **CI**: the `e2e` job timeout rises to 30 minutes; `workers` stays 1 until the suite exceeds 10 minutes on CI, then 2 (D126).

## 4. Rules for every case

- The oracle is the spec text (Spec 1 to Spec 8) or the README persona table; the catalogue cites the section. A case that asserts behaviour no spec names is a finding for the orchestrator, not a test.
- Selectors are accessible roles and labels (`getByRole`, `getByLabel`, `getByText` with `exact` where a substring would be ambiguous); no `data-testid`, no CSS classes, no `waitForTimeout`.
- Seeding: through `seed-demo` for persona-based reads (D124), through `page.request` against the HTTP API for writes the test does not itself assert, and through the UI only where the UI flow is the assertion.
- Each spec file owns its university (slug prefix per file plus `runId`), and D33 holds: the database is never reset.
- A failing case must point at the product: the assertion message names the role, the page and the expected copy.
- Phone layout is checked inside the area's file at 390 px where the design has a phone variant (stacked tables, agenda), not in a separate project.

## 5. Harness details

- `UNICOURSE_E2E_PORT` sets `UNICOURSE_ADDR`, `UNICOURSE_BASE_URL` and `baseURL`; `global-setup` still refuses a taken port. Each worktree has its own `e2e/.tmp`, so API logs never mix.
- `seedDemo(slug)` runs `unicourse-admin seed-demo --slug <slug>` with `e2eEnv` and returns the persona emails (`<local>@<slug>.edu`).
- `api(page, method, url, data)` and `upload(page, url, csv, commit)` are the `screens.ts` functions, unchanged, exported from `helpers.ts`.
- `signIn(page, email, slug, landing?)`: `landing` defaults to `/${slug}/me`.

## 6. Gates

- `npm run lint`, `npm run typecheck` (covers `e2e/` through `tsconfig`), `npm test -- --run`, `npm run build`.
- `npm run e2e -- <file>` for the lane's file on its own port, green twice in a row (flakiness is a defect).
- The orchestrator runs the whole suite before each PR and records the wall-clock time in PLAN.md; CI `e2e` green on the PR.
- `npm run screens` still passes after the harness lane (it imports the moved helpers).

## 7. Out of scope

- OIDC against a real provider, SMTP delivery, the 100 MB upload boundary (unit-tested in Go with `-short` off), visual snapshot comparison, performance budgets.
- Playwright MCP, `playwright-agents` or any LLM-in-the-loop runner (D122).
- Changes to the three existing spec files beyond `screens.ts` importing the moved helpers (D60, D80).
- Product fixes: a defect a test exposes is reported, the case is marked `test.fixme` with the finding ID, and a fix lane follows under its own PR.

## 8. Internal layout

```
apps/web/e2e/
  env.ts helpers.ts global-setup.ts global-teardown.ts   harness (P16-L2)
  demo.spec.ts                                            P16-L2
  auth.spec.ts admin.spec.ts courses.spec.ts              P16-L3, L4, L5
  grades.spec.ts catalog.spec.ts content-edit.spec.ts     P16-L6, L7, L8
  a11y.spec.ts                                            P16-L9 (D125)
docs/design/P16-L1.md                                     catalogue
```

## 9. Decisions needing the owner's call

Recorded in `PLAN.md` as D122 to D126, each "Orchestrator 2026-10-07, owner to confirm":

- D122: "AI automation" is read as cases designed and built by the lanes; the runner is plain `@playwright/test`, no Playwright MCP or LLM runner and no new runtime dependency.
- D123: Phase 16 structure: catalogue first, harness lane next, six area lanes in parallel on disjoint spec files, accessibility last; `UNICOURSE_E2E_PORT` makes parallel e2e runs possible.
- D124: new spec files may seed through `seed-demo --slug` or the HTTP API (amends D108 for the e2e suite only; the capture script is unchanged).
- D125: `@axe-core/playwright` as a pinned dev dependency for one accessibility sweep; the lane is dropped if declined.
- D126: CI `e2e` timeout 30 minutes; `workers` 1 until the suite exceeds 10 minutes on CI.
