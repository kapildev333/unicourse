# P10-L1b-review - review of P10-L1b materials/weeks/download
Status: done
## Doing
Review complete. Verdict: approve. One low-severity finding and two optional test gaps.
## Found
- Gates: gofmt clean, vet clean, go test ./... -count=1 all ok with 0 SKIP, -race on content/blob/tenant ok, npm run generate leaves types.ts clean, typecheck ok.
- Low: blob.Commit can fail at the dir Sync after the rename has happened. receive then calls tmp.Remove() (that path no longer exists), so the renamed blob stays as an orphan until prune-files runs (after one hour). Fix: on a Commit error, also call d.Files.Discard(uni, id) (material.go:312/318-323).
- Checked and correct: After runs only after commit and never on error; deleteSection collects ids under the `for update` row lock (no race with an upload's second tx); the 409 path keeps files; the upload removes the temp on every error path; a client disconnect gives 400 or 499 with the temp removed; renumber SQL; a weekId from another section, a non-uuid weekId or an uppercase weekId gives validation_failed on weekId; the mail filter is limited to the tenant by RLS; prune-files skips .tmp and recent files and aborts before deleting if any university scan fails.
- Test gaps (optional): no test that a rolled-back week or material delete keeps its files (only the section 409 is tested); no test for the second-tx failure after the rename.
## Requests
## Blockers
