# P18-L3-review - review of P18-L3 (migration 0010, assessment kinds in grade)
Status: done
## Doing
Finished; verdict approve with two small follow-ups (contract text, test strength).
## Found
- DDL matches design section 1 and E4; RLS/force/tenant_isolation/trigger on all four new tables.
- Gates green: gofmt clean, go vet clean, go test grade/pg/app ok; web typecheck ok, vitest grades 86/86; types.ts and Postman regenerate with no drift.
- openapi listAssessments description still says students see "published assessments only"; now quiz/assignment rows are visible unpublished.
- POST/PATCH "Errors:" lines omit kind, description, opensAt, closesAt.
- TestUpsertScore cannot fail if graded_at = now() is dropped from the conflict update (at2.Before(at) is false when equal).
- Sub-microsecond window (e.g. opensAt ...00.0000001Z, closesAt ...00.0000004Z) passes Go check, rounds equal in Postgres, 23514 -> 500.
## Requests
## Blockers
