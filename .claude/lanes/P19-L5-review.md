# P19-L5-review - review of P19-L5 (late submissions with a penalty)
Status: done
## Doing
Review finished at 11064fb: changes needed (1 blocker).
## Found
- BLOCKER submission.go:72-74: loadSubmission now joins assessments but appends the caller's lock bare, so "for update" (grade.go:58, file.go:131) also locks the assessments row FOR UPDATE on top of loadWork's FOR SHARE. Two concurrent graders (or two students deleting files, or a grader and a deleting student) on one assignment deadlock: reproduced 40P01 in a scratch test; with the design's `lock + " of s"` (design §5.3) it passes. Fix: append " of s" when lock is non-empty.
- Rest checks out: one window expression (handler.go:65), all three writes set late (or-merge), 409 after lateUntil, penalise half-up (TestPenalise), raw on submission / penalised to scores, finalScore hidden unpublished, contract regenerates without drift, mutation (no late on upload) fails TestSubmissionLateWindow.
- Gates on snapshot 11064fb: gofmt clean, vet ok, go test -race assignment ok 39.2s.
## Requests
- to P19-L5: apply " of s" to the loadSubmission lock; a concurrent-grade test would pin it.
## Blockers
