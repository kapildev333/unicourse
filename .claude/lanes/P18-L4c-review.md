# P18-L4c-review - review of sub-lane 4c (quiz attempts list, review, role matrix, wiring, contract)
Status: done
## Doing
Finished. Verdict: changes needed (contract map bodies say 1 entry max; nested unknown fields in review body silently accepted).
## Found
- Gates in the worktree at caaa315: gofmt -l clean; go vet ./... clean; go test -race -count=1 ./internal/quiz/ ./internal/app/ ok (26 quiz tests, 3 app tests, no SKIP); web npm run typecheck clean; types.ts regenerated into scratch is identical; Postman regenerated 3x is byte-identical to the committed file.
- HIGH openapi.json AttemptPut.answers and AttemptReview.grading carry minProperties 1 and maxProperties 1: the contract says a student may save one answer and a review may touch one question. The code accepts {} and any number. Only min=max=1 makes openapi-to-postmanv2 stable (tried 0..100, 1..100, 0..0, schema-level example, no bounds: all vary per run). Real fix is in apps/web/scripts/postman.mjs (normalise key_N maps), outside 4c's scope: orchestrator decision.
- MEDIUM review.go:107 tenant.Opt[map[string]markIn] decodes with plain json.Unmarshal, so DisallowUnknownFields is lost inside grading: {"grading":{"<q>":{"points":1,"feedbak":"x"}}} is 200 and the feedback is dropped; {"feedback":5} reports field "body" not "feedback".
- Mutations: enrollment join, unsubmitted 409, staff 403, partial credit correct, sum over all questions, graded_by each fail a test. Removing "for update" fails nothing (no concurrent-review test).
- Low: Question.position minimum 0/example 0 but positions are 1-based; PATCH Errors omit "must not be null" and "must not contain NUL"; 356c45c committed .claude/lanes/P18-L4b-review.md (another lane's log); a "wip" commit.
## Requests
- to orchestrator: decide the Postman stability fix (postman.mjs owner) so the two map schemas can drop min/maxProperties.
## Blockers
