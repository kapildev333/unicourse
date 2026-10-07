# P10-L1a-review - review of P10-L1a (section content API foundation)
Status: done
## Doing
Review complete. Verdict: approve. Two things must happen before the merge: rebase onto main, then re-run the gates.
## Found
- Branch base is c044dd4. main is 5 commits ahead (P9-L8 with 0006, openapi.json, types.ts and /me PATCH; P9-L1a and P9-L1b web). `git merge-tree` merges cleanly, but every gate ran on the stale base. Rebase, run `npm run generate`, and re-run TestMigrate (expects 25/20) and the web gates.
- Migration 0007 matches the design SQL line by line. The contract only adds to main's contract (checked semantically against the merge base). types.ts is byte-identical to a fresh openapi-typescript run.
- No L1b code is in the diff (no After, weeks, materials, upload, download or prune-files). The README Files section mentions prune-files ahead of L1b, which design section 8 asks for.
- For P10-L1b: blob.Commit (blob.go:114) renames without fsyncing <root>/<uni>. After a power cut, a committed materials row can point at a rename that was lost. Fix: open the directory and call Sync() after the rename.
- Spec-level, for the orchestrator: deleting a membership leaves its enrollments and section_staff rows, so a removed member still gets announcement mail (mail.go:22-26). This matches spec section 7 as written; the gap is in the existing data model.
## Requests
- to orchestrator: rebase lane/P10-L1-content-api onto main and re-run the gates before the PR; decide on the removed-member mail question; carry the Commit dir-fsync into the P10-L1b brief.
## Blockers
