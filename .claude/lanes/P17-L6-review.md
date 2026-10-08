# P17-L6-review - review of P17-L6 (q on departments/terms, LikeParam move, picker search, overlap contract)
Status: done
## Doing
Review complete; verdict changes needed (one small web fix), report sent to orchestrator.
## Found
- API: SQL, LikeParam move, 400 path, cursor paging under q all correct; RLS untouched. Gates green (gofmt, vet, tenant+app tests no SKIP, lint, typecheck, vitest 158, postman and types regenerate clean). Merges clean onto origin/main (P17-L5 #149); overlap/20-row contract text matches meeting.go on main.
- department-picker.tsx: while searching, a pending or empty result shows "No departments yet. Create one under Departments." (wrong copy); found.error ignored.
- Term search is a no-op for terms past the first 200 on Section list too (section-list.tsx:73 falls back like useTermChoice), not only Catalog/My sections.
- P17-L5 landed without documenting the import row overlap message in importSections; still open.
## Requests
- to orchestrator: decide the Section list term fallback (accept as ceiling or pass the picked Term up); schedule the importSections overlap text.
## Blockers
none
