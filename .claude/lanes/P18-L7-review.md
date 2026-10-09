# P18-L7-review - review of P18-L7 (web assignments)
Status: done
## Doing
Finished; verdict changes needed (two small fixes), reported to the orchestrator.
## Found
- Gates green in the worktree: lint, typecheck, vitest src 512/512 (no-literal-colors included), build (routeTree.gen.ts unchanged after build).
- M1 submissions-grading.tsx:187/189-195: dirty stays true after a successful save when the typed score is not in fmtNum form ("17.50", "017") or the feedback has outer whitespace (the server trims), so leaving the panel raises "Discard unsaved changes?". Fix: on success setScore(fmtNum(s.score)) and setFeedback(s.feedback ?? '').
- M2 submission-form.tsx:267: the copy 'validation_failed:body' replaces every body 400, so the API's "a submission needs text or a file" (stale file list, e.g. the last file removed in another tab) shows "Use at most 20,000 characters of plain text.". Fix: map by message (only "must be at most 20000 characters" gets the copy) or show the server sentence otherwise.
- L1 brief vs design: grade PATCH always sends feedback (design 8 says {score, feedback}; the brief asked for feedback only when changed). Orchestrator's call.
- Info: Score label is "Score for <name>" (design 8 says "Score"); e2e must use the longer label.
- 6a's request to P18-L7 (drop `as never` in section-tabs.tsx) done; my-grades.tsx left to 6b as the worker noted.
## Requests
## Blockers
