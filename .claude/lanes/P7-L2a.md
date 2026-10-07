# P7-L2a - grading web: grade logic, queries, Grade scale page
Status: done
## Doing
Landed on lane/P7-L2-grading-web; awaiting orchestrator verification.
## Found
- Route is `routes/$slug/grade-scale.tsx` at `/{slug}/grade-scale` (D65), not `admin/grade-scale` as in design §5. Outside AdminLayout, so members read the scale (read-only table plus the administrators-only note); admin edits. Design §0 #16 / E2 is thereby resolved; L2b's "scale in a details on My grades" is optional, the route works for members.
- `ScaleEditorPage` reads the role from `useMe()` (not `useUniversity`), so it needs no router in tests.
- For L2b: `queries.ts` exports aliases `GradeScaleRow, GradeScale, GradeScaleUpdate, Assessment, Gradebook, MyGrades`, `useGradeScale(slug)` (key `[slug,'grade-scale']`, staleTime 60s), `useGradesRefresh(slug)` (gradebook + my-grades + audit-log), `useScaleRefresh(slug)`. Not yet there (L2b/c): useGradebook, useSection, useMyGrades, putScores, downloadGradebookCsv.
- `grade.ts` exports: gradebookAccess/Access, round2, fmt2, fmt1, fmtNum, parseDecimal, parsePoints, letterFor, weightTotal, totalsHundred, finalPercent, validateScale/ScaleErrors, hasScaleErrors, sentence. `presets.ts`: usPreset, canadianPreset. `scale-editor.tsx`: ScaleEditorPage, ScaleTable({rows, caption}).
- Schema names matched the design; no aliasing needed. Typecheck was green on this branch (the L1a-reported audit-log and section-dialog.test failures did not occur here).
- Nav link to /grade-scale not added (Phase 6 files).
## Requests
## Blockers
