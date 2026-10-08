# P18-L4a-review - review of P18-L4a (quiz grading and questions routes)
Status: done
## Doing
Finished. Verdict: approve. Three low-severity findings, none blocking.
## Found
- Gates on an ac03aa5 snapshot (git archive): gofmt clean, go vet clean, go test -race on TestAutoGrade|TestNormalize|TestQuestions ok, whole quiz package ok.
- Mutations (accepted answers not normalised, out-of-range index, control chars, keys need a submitted attempt, lock on any attempt, closed 409) each fail a test. The `len(chosen)==0` guard in isRight is unreachable for validated questions (no test fails without it).
- Low: question.go:257 first audit `before.totalPoints` is "0", after an empty PUT it is "0.00" (use `coalesce(sum(points), 0.00)`).
- Low: question_test.go:314-318 says "closed quiz" but uses `up`, so closed + unsubmitted attempt is untested (the no-attempt closed case is).
- Low: role matrix has no student enrolled in another section (plain member `other` takes the same path).
- to P18-L4b: autoGrade needs questions with keys (Answers non-nil, Correct set); call it on loadQuestions output before any hideKeys. The questions PUT takes the assessment row `for update`, so attempt start should take it `for share` to serialise with the lock check.
## Requests
## Blockers
