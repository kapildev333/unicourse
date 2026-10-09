# P19-L4-review - review of P19-L4 (per-attempt time limit)
Status: done
## Doing
Review of f97d4ce (range 387afdc...HEAD) finished. Verdict: approve. No blocking findings.
## Found
- Every attempt path now uses deadlineSQL: putAttempt (both the now() check and the clock_timestamp() re-check), the auto-submit submitted_at, and the sweeper's claim and order. No closes_at comparison is left in attempt.go, submit.go or sweeper.go except startAttempt's w.Closed, which the design keeps on purpose.
- Gates on a snapshot of f97d4ce all pass: gofmt, vet, quiz -race (37.7 s), app contract test, typegen and Postman regenerated with no diff, web typecheck.
- Mutation results: the clamp, submitted_at, untimed deadlineAt, the sweeper claim, the lock and removing the re-check each fail a test.
- Two mutations survive. (a) Changing the live re-check's clock from clock_timestamp() to now() fails nothing: closeQuiz sets the close 1 hour back, so now() is already past it. This gap was already there in Phase 18. (b) Dropping the sweeper's order by deadline fails nothing: one Sweep keeps going until no attempts are left.
- If the row lock is removed, TestAttemptPutBodyArrivesAfterClose hangs until the 40 m timeout instead of failing (its spin loop has no time bound). This is also from Phase 18.
- Web vitest could not run from the scratch snapshot: the symlinked node_modules hit an ESM require error in html-encoding-sniffer. This is a problem with my setup, not with the lane's code.
## Requests
none
## Blockers
none
