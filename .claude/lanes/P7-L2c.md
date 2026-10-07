# P7-L2c - grading web: finalize, override, reopen, import, export
Status: done
## Doing
Landed on lane/P7-L2-grading-web; awaiting orchestrator verification. Gates in apps/web green: lint, typecheck, test (242), build.
## Found
- Export file name is `<course>-<section>-gradebook.csv` (the brief), not design's `-grades.csv`; test asserts the brief's name.
- `OverrideDialog` lives in `finalize-dialog.tsx` (design §7); no `override-dialog.*` files were created.
- Final column renders only once the section is finalized; Finalize button is disabled with a visible reason (loading scale, no students, weights != 100).
- Finalize server override errors map to rows via `m.variables` (the sent overrides, in order).
- Import scores button shows for `access.scores` when there is at least one assessment (admins: none, per D65 / §3.9).
## Requests
- none
## Blockers
