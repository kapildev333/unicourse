# P8-L3a - Benchmark comparison: entry, shell, My sections, Catalog (docs/design/P8-L3a.md)
Status: done
## Doing
Done: docs/design/P8-L3a.md written (310 lines), not committed (orchestrator commits).
## Found
- For P8-L3d: top items are schedule overflow at 390 (capture 755 px wide, page scrolls sideways), Catalog Clear filters + quoted query, Enrolled badge on Catalog cards from `mine`, admin action on "No terms yet", one search field (header vs page Search), single sign-in field.
- Copy changes that hit selectors: e2e `staffing.spec.ts` line 212 `getByLabel('Search')` (one-search item), e2e `signIn` helper `getByLabel('Email', { exact: true })` (single sign-in field), department and term option labels if e2e selects by label.
- Capture artifact: 21-light Schedule button shows hover colour (pointer resting), not a bug.
- Direction questions (B): per-course colour (pack Q7), user menu (pack Q5), provider brand marks (D53), global search scope (pack Q4).
- C items: display names, invite details for signed-out visitors, catalog total count and sort param.
- Overlap: Courses/Sections empty states (12, 13) and tabs-above-h1 on Courses are noted for P8-L3b; phone and dark findings here are limited to 01, 20, 21, 30 (P8-L3c covers the rest).
## Requests
## Blockers
