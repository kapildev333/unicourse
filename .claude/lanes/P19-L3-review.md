# P19-L3-review - review of regrade after an answer-key change
Status: done
## Doing
Review finished: approve, one escalation, one test gap.
## Found
- Gates: gofmt clean, go vet ok, go test -race quiz ok (29s); types.ts and Postman regenerate with no drift.
- Escalation: a manual entry keeps its points even when the question's points drop below them (regradeMarks), so score can exceed max_points and re-saving that entry in review is 400. Spec says "keep", so orchestrator to rule (clamp vs keep).
- Test gap: removing the `for update` lock in regrade.go:33 fails no test (no concurrency test; design did not ask for one).
- Lock order checked: regrade = assessment for update -> attempts by id for update -> scores; putAttempt/review lock one attempt, sweeper uses skip locked, startAttempt waits on the assessment share lock. No cycle.
## Requests
none
## Blockers
none
