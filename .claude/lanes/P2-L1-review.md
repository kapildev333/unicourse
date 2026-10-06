# P2-L1-review - review of csvimport engine
Status: done
## Doing
Review finished: changes needed (2 small code fixes, 2 test tightenings).
## Found
- Gates green: gofmt clean, vet clean, build ok, go test ./... -race -count=1 all ok. Fixture bom-crlf.csv committed with efbbbf and 0d0a.
- Invalid UTF-8 (Windows-1252 Excel export) or NUL in a cell passes Parse; Postgres rejects it with 22021, unmapped -> 500 in classifier/applier. Fix in Parse: reject non-UTF-8/NUL file as {file, ...}. Extends design §7 (orchestrator call).
- Whitespace-only or short trailing line ("   " or "," in a 3-col file) aborts the whole file with "line n: wrong number of fields" instead of being skipped as an empty record (§7 intent, Review Focus 4).
- Test looseness: csvimport_test.go:244 ignores the count query error (passes vacuously); :131 ignores Parse error.
- Nits: empty Key rows become "duplicate of line N" instead of the classifier's message; Apply's sort is redundant (PreviewRows already in line order).
- 5 MB / 119k rows: parse 29 ms, 72 MB alloc, preview 19 ms, apply 8 ms; 50 MB body: only 5 MiB consumed. Engine sufficient for §8.1.
## Requests
## Blockers
