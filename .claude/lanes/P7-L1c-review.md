# P7-L1c-review - review of P7-L1c (scores CSV import, gradebook CSV export, csvimport Update, tenant.Raw)
Status: done
## Doing
Verdict: approve (no blocking defects; nits below). Reviewed 4a5dfba (5ef760e plus the test-expectation fix) against base 4b8c0cf.
## Found
- 5ef760e alone was red: TestGradebookCSV expected an unquoted `'=HYPERLINK(""x"")` cell, but encoding/csv quotes any field holding `"`. 4a5dfba fixes only the expectation. The export output is correct CSV.
- Gates at 4a5dfba (both test URLs exported): gofmt -l . empty; go vet ./... ok; go test ./... -count=1 -timeout 30m: all 14 packages ok, 0 SKIP. apps/web npm run generate: no diff. (My first full run at 5ef760e was killed at the 600 s shell limit while grade waited on the shared-DB lock. Not a product issue.)
- Probes, run as handler tests on a scratch copy: BOM + CRLF + `Email,Points` header -> exists/add. Points `abc`, `1e2`, `+5`, `.5`, `5.`, `0x10` -> "points must be a number"; `-1`, `200.01` -> "between 0 and 200.00"; `1.234` -> "at most 2 decimals"; `200` ok; ` 5 ` is trimmed and ok. Over 5 MiB -> 400 file "file too large". A TA commit of update+add returns 200, and the single score.set audit lists only the applied rows. Re-preview gives exists/exists, and that commit adds no audit row. Roster preview JSON still has counts {add,error,exists,invite} with no update key.
- Export: `text/csv` -> `text/csv; charset=utf-8` with a BOM and the `'` escape on `=SUM(A1)` and `-Quiz` header names. `*/*`, empty and `TEXT/CSV` -> JSON. `text/csv;q=0.1, application/json` -> CSV, and so do `application/json, text/csv` and `text/csv-ish`: it is a substring match, the ceiling the design marked with ponytail.
- Mutations (scratch copy, all caught): unregistering the preview route -> contract test "not registered in Go"; an undocumented route -> "in Go but not in openapi.json"; treating members as enrolled -> TestScoresImport fails at line 7; dropping the BOM -> TestGradebookCSV fails.
- Nits (optional):
  1. tenant/middleware.go Raw branch: `_, err := w.Write(raw.Body); return err`. Design 3.2 says `return nil`. If the client disconnects mid-download, httpx.WriteError logs a spurious "internal error" and calls WriteHeader a second time (superfluous). Fix: `w.Write(raw.Body); return nil`, which matches WriteJSON.
  2. app/contract_test.go keeps an empty `pending` map and diff's pending parameter. P2-L2c deleted its pending list once it was empty. Could go, but keeping it costs nothing.
  3. No scores-import test sends a BOM/CRLF file or an oversize file (csvimport's own tests cover both). Fine.
## Requests
none
## Blockers
none
