# P18-L5-review - review of P18-L5 (internal/assignment: submissions, files, grading)
Status: done
## Doing
Finished; verdict approve, with low-severity follow-ups.
## Found
- Gates green: gofmt clean, go vet clean, go test -race assignment ok; content, app, unicourse-admin, grade, course ok; web typecheck ok; types.ts and Postman regenerate with no drift.
- Material refactor onto content.ReceiveFile/ServeFile is behaviour-equivalent (headers, limit, allowlist, temp cleanup unchanged; content tests green).
- Mutation check: 14 of 15 mutants killed (window, empty-submission delete, 2x cap, grade hiding, PUT projection, assessment and section blob discard, submittedAt move, graded_by, other-user 403, feedback cap, list editor check, limit boundary). Survivor: dropping up.Temp.Remove() in assignment/file.go receive (phase-2 failure path has no test).
- Low: putSubmission counts files before the upsert without a row lock; a concurrent last-file delete can leave a body '' submission with no files (fix: upsert first, then count, 400 rolls back).
- Low: PATCH grades a former (unenrolled) student's submission and upserts a scores row; grade.putScores rejects unenrolled users. Orchestrator to decide whether to require enrollment.
- Process: tests were not committed before the code (d2ef297 holds code and submission tests together; file/grade/authz/prune tests come after).
- Nits: download with a non-uuid userId as staff says 404 "submission" not "file"; SubmissionFileUpload has additionalProperties false while the description says other parts are ignored.
## Requests
## Blockers
