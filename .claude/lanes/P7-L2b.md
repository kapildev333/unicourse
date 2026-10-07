# P7-L2b - grading web: My grades, Gradebook core
Status: done
## Doing
Landed on lane/P7-L2-grading-web; awaiting orchestrator verification. Gates in apps/web green: lint, typecheck, test (225), build.
## Found
- For L2c, hooks and props to extend in `features/grades/gradebook.tsx` (marked `L2c:` comments): `PageHeader` `action` (Finalize / Reopen; `finalized = students.some(s => s.final)` already computed, as is `total`/`totalsHundred`), toolbar right-hand div (Import scores with `access.scores`, Export CSV with `access.view`), Final `th`/`td` after Letter and the tfoot cells (add a cell per extra column; `colSpan={assessments.length}` stays), `setMessage` (StatusMessage) and `setPageError` (ErrorBanner) for results. `refreshGrades = useGradesRefresh(slug)`; `scale` from `useGradeScale`; `access = gradebookAccess(...)` incl. `finalize`.
- `queries.ts` now has `useSection`, `useGradebook` (casts: the contract also declares text/csv for this path, so unwrap types it string | Gradebook), `useMyGrades`, `putScores`, `Section`/`ScoresPut` aliases. `downloadGradebookCsv` is L2c.
- Cell state lives in `gradebook.tsx` (`cells` + `cellsRef` via `patch`); `save(a, entries, note)` is the single save path.
- Deviations: Grade scale link in the stats footer goes to `/{slug}/grade-scale` (L2a's route, D65), not admin/grade-scale. Paste "didn't fit" note is appended to the Saved announcement (a separate announce would be overwritten by it). No separate "Final" column, no Finalize button yet (L2c). Stats footer is omitted when there are no assessments (the empty state replaces the table).
- `npx prettier` is not a project tool; I used `--no-semi --single-quote --print-width 120 --trailing-comma all` to match the code style.
- Route files: `routes/$slug/grades.tsx`, `routes/$slug/sections_.$id.grades.tsx`; routeTree.gen.ts regenerated.
## Requests
- to orchestrator (E5): nav entry to /grades and a Gradebook link on section detail are Phase 6 files, not added.
## Blockers
