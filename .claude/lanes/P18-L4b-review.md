# P18-L4b-review - review of sub-lane 4b (quiz attempts, submit, sweeper)
Status: done
## Doing
Finished. Verdict: changes needed (one finding, the close is not authoritative against a slow request body).
## Found
- Gates on a git archive of 4576559: gofmt -l clean; go vet ./internal/quiz/ ./cmd/api/ clean; go test -race -count=1 ./internal/quiz/ ./cmd/api/ ok, no SKIP.
- HIGH attempt.go:232-239: the close check uses the transaction's now() (taken when Wrap opens the tx, before the body is read) and the attempt row stays locked while DecodeJSON reads the body. The API server has no ReadTimeout. Probe: send PUT attempts/me with headers and a partial body before closes_at, finish the body after it. Result: 200, the answers were chosen after the close, submitted_at was before closes_at, auto false, score 5.25, and the sweeper skipped the locked row. Fix: after DecodeJSON, re-check `select clock_timestamp() >= closes_at from assessments where id = $1` and return the same 409 text. Test it with an io.Pipe body plus closeQuiz between the two writes.
- 4a requests answered: keyedQuestions loads the keys for autoGrade; start takes `for share`.
- Mutations: removing the closed check, the hide-results check, the empty-entry drop, closes_at for the auto submitted_at, the closes_at claim filter, the score value or the already-submitted check each fails a test. Removing `for share` or `skip locked` fails nothing; the 4b log says there is no interleave test.
- TestSweepTwoProcesses really runs concurrently (counts [12 18] on 5 runs).
- Low: commits 73a7663/351956a also committed .claude/lanes/P18-L4a-review.md (another lane's log).
- Orchestrator: every tenant.Wrap handler reads its JSON body inside an open tx, and the server sets no ReadTimeout. A slow client holds a pool connection (pre-existing, plan-worthy).
## Requests
- to P18-L4b (via orchestrator): fix the HIGH finding above.
## Blockers
