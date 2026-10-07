# P7-L1b-review - review of P7-L1b (gradebook, finalize, overrides, reopen, /me/grades)
Status: done
## Doing
Verdict: approve (no blocking defects). Reviewed 4b8c0cf against ad36559.
## Found
- Gates (run by me): gofmt -l . empty; go vet ./... ok; go test ./internal/grade/... ./internal/app/... -count=1 ok (34 tests PASS, no SKIP); npm run generate leaves no diff.
- Probes (scratch copy, real handlers + SQL), all as designed: 89.995 -> B+ stored 90.00 pts 3.30; exactly 93 -> A; score edit after finalize leaves final_grades, gradebook percent moves (90 -> 50) with final still 90.00 B+; 0-weight-only scores -> percent/letter/mean/median omitted, letters all 0; weights 40+59 -> 400 weights "is 99.00"; course credits 4.5 copied to every row incl. incomplete; graded+letter C override -> points 2.00; dup userId / null status / bad uuid field paths; PATCH withdrawn clears percent/letter/points, keeps credits; user not in section and non-uuid -> 404; real university-B section -> 404 on all 5 routes; TA/stud/daout reopen -> 403; /me/grades 4.0 A + 2.5 C + withdrawn -> gpa 3.23, credits 6.5 (withdrawn excluded); unpublished score 7.77 never in the body; stats skip withdrawn.
- Mutations: letter on rounded percent in finalize -> TestFinalizeBoundaryPin fails; weights check dropped -> 2 tests fail; c.credits in GPA -> TestMyGradesGPA fails; blocked dropped -> TestMyGradesPublishedOnly fails; letterFor `<` -> 3 fail; no publish -> TestFinalize fails; stats ignore final -> fails. SURVIVED: finalize credits replaced by constant 3.0; gradebook letter on rounded percent (gradebook.go:188).
- D66 confirmed: finalize copies c.credits into every final_grades row; GPA reads fg.credits; test asserts a course edit does not move the GPA.
- Nits / escalations: see final report (credits test gap, gradebook rounded-letter test gap, finalize audit letters not asserted, PATCH 409 missing from contract, /me/grades section credits live vs snapshot, unreachable status-0 return in overrideFinal, vacuous cross-tenant check in TestMyGradesShape).
## Requests
- to orchestrator: decide whether MyGradesSection.credits should be fg.credits when a final exists (D66 consistency), and add 409 to updateFinalGrade in the contract (L1c or follow-up).
## Blockers
none
