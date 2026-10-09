# P19-L2-review - review of P19-L2 (migration 0011, assessment fields, quiz deadline, student questions GET)
Status: done
## Doing
Review complete: approve, no blocking findings.
## Found
- DDL is the design's §1 text exactly; grade, pg, quiz and app tests pass; gofmt and vet are clean; typecheck passes; types.ts and Postman regenerate with no diff.
- Nits: AssessmentCreate says "null or omitted" but its schema is not nullable; the listQuestions description says `started_at` instead of `startedAt`; tx_test and deadline_test landed in the same commit as the code they test, not before it; no PATCH test for `lateUntil` alone at or before closesAt (the same code path is tested on create).
- grade package hit the 10 min go test timeout on pg_advisory_lock while other suites ran; it passes with -timeout 40m (118 s).
## Requests
none
## Blockers
none
