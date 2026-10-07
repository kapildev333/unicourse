# P7-L1b - gradebook, finalize / override / reopen, GET /me/grades
Status: done
## Doing
Landed on lane/P7-L1b-gradebook-final (5 commits on top of ad36559); awaiting orchestrator verification. Contract unchanged, `npm run generate` leaves no diff.
## Found
- review fixes applied: 5b3fba2
- Differences from the design, for P7-L2 and later lanes:
  - `final_grades.credits` (L1a ruling) is copied from the section's course at finalize, on every row; `FinalGrade.credits` is in every response. The GPA reads `fg.credits`, so editing a course's credits afterwards does NOT move past GPAs (the design test said it would; the test now asserts the opposite and that editing the final grade's own credits does).
  - JSON names are the contract's: `Gradebook{assessments, students, stats}`, `GradebookStudent{userId,email,displayName,scores,percent?,letter?,final?}`, `GradebookStats{mean?,median?,letters}`; `MyGrades{terms[{term,gpa?,credits,sections}],cumulativeGpa?,cumulativeCredits}`, section `{section,course,credits,final?,running?,assessments}`.
  - `applyOverride` takes the matrix (scale inside) instead of a separate scale; `validateOverride` takes the row's current status.
  - PATCH final-grades: neither status nor letter set -> 400 field `""` (`no fields to update`), also when only a reason is sent.
  - `running` in a gradebook row is omitted (with percent) when the scale has no row for the percent.
  - Overriding a no-longer-enrolled student back to `graded` recomputes from no scores (0 percent); ponytail comment in `overrideFinal`.
- Letter is decided on the unrounded percent (92.995 -> A-, stored 93.00), tested in TestLetterBoundaries and TestFinalizeBoundaryPin.
- Contract pending set now holds only the two L1c import routes.
## Requests
- to orchestrator: nothing outside scope was needed.
## Blockers
none
