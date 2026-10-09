# P18-L8a-review - review of the P18-L8a demo seed landing
Status: done
## Doing
Review finished: approve, with low findings for the orchestrator to decide.
## Found
- All writes go through the real handlers as the right persona (Elena creates, students attempt and submit, Sofia reviews, Elena grades). No SQL on the assessment, attempt or submission tables. The file goes through the real multipart upload. Reset cascades and removes the blob directory (the test asserts 4 blobs, then 0).
- The schema walker change does not weaken anything: `additionalProperties: false` is not a map, so unknown keys still fail.
- Seed rows do not affect demo.spec.ts or the README persona table: CS101 is finalized, weight 0, and the running-grade "not yet published" line only shows on open sections.
- L1: the "attempt by a non-student" reject case uses invited Ingrid, so it hits the `invited` branch of ref. The new `!inRoster` check in validateWork's student() is untested (deleting it keeps the tests green).
- L2: duplicates are not rejected: two attempts by one student (runtime 409, partly seeded), two submissions by one student (silent upsert), two reviews of one student (silent last wins). A review of a student with no attempt in the fixture is not checked either.
- Nits: `as` is not checked to be staff on the section; windows are not checked to be open when attempts or submissions exist (runtime failure, not a Load error); the graded-submission count does not assert graded_by = Elena.
## Requests
none
## Blockers
none
