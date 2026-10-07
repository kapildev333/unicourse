# P14-L1b-review - review of API grading and catalog backlog items
Status: done
## Doing
Finished. Verdict: approve. Three small text fixes are listed below and none of them blocks.
## Found
- Gates (re-run by the reviewer): gofmt clean, go vet clean, go test grade/course/app -race -count=1 ok (32.8s/36.1s/1.7s), 0 SKIP lines.
- Branch base c685720 is behind main c1ccf1a. Two-dot `git diff main..HEAD` shows PLAN.md/.claude/agents reverts, but the three-dot landing is clean and `git merge-tree` merges cleanly.
- Keyset by row position is acceptable for the catalog. It has two ceilings: a deleted cursor row gives an empty page that looks like the end of the list, and an edit to the cursor row's code or number moves the next page (rows can be skipped or repeated). Only course.go carries a ponytail comment, and that comment names only the delete case.
- openapi FinalGradeUpdate/updateFinalGrade still says "points follow the scale row". A same-letter override now keeps the recorded points.
- For P14-L3: apps/web/src/features/grades/my-grades.test.tsx section fixtures (around lines 29-56) lack unpublishedCount. The fixtures are untyped, so typecheck will not catch it.
## Requests
- to P14-L3: add unpublishedCount to the MyGradesSection fixtures in my-grades.test.tsx.
## Blockers
