# P8-L3b - per-screen comparison: courses, section detail, dialogs, admin tables (docs/design/P8-L3b.md)
Status: done
## Doing
Done. docs/design/P8-L3b.md written (251 lines); not committed (orchestrator commits).
## Found
- for P8-L3d: the 390 captures of 44, 45 and 52 are 693/555/561 px wide, so the document overflows sideways at phone width (T1, top item). P8-L3c may report the same from its phone pass; merge them, do not count them twice.
- for P8-L3d: dialog backdrops in the tall full-page captures cover only the first viewport. That is a screenshot artifact, not a finding.
- for P8-L3d: em dashes in select options ("CS — Computer Science") are e2e selectors (staffing.spec.ts 140-160); any copy fix must update the e2e in the same change.
- for P8-L3a: on /sections/{id} the global nav marks no current item (Courses not underlined); listed as 2.6 improvement 6 here, owner of the shell finding is yours.
- for P8-L3a: your note about Courses/Sections empty states (12, 13) and tabs above h1 was not covered here (no empty-state captures in my set); left to you or P8-L3d.
- Dribbble images were not opened; the pack's words were enough.
## Requests
## Blockers
