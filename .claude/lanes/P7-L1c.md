# P7-L1c - scores CSV import, gradebook CSV export, csvimport Update, tenant.Raw
Status: done
## Doing
4 commits on lane/P7-L1c-import-export (on top of 4b8c0cf); awaiting orchestrator verification. Contract unchanged; contract pending set is now empty.
## Found
- For P7-L2c, import preview row (JSON, same envelope as the other imports): `{"checksum":"<sha256>","rows":[{"line":2,"action":"add|update|exists|error","message":"...","data":{"email":"stud@a.edu","points":"90"}}],"counts":{"add":n,"error":n,"exists":n,"update":n}}`. No `invite` key. `data` is the raw trimmed cells keyed by lowercased header (`email`, `points`). `update` message is `"80.00 → 90.00"` (old stored, new canonical). Error messages: `invalid email`, `unknown email`, `not enrolled in this section`, `points <decimal message>` (e.g. `points is required` for an empty cell, `points must be between 0 and 200.00`), `duplicate of line N`. Commit with errors: 400 validation_failed, details `line:<n>`.
- CSV header line: `email,displayName,<assessment names by position>,percent,letter,final` after a UTF-8 BOM; file name `gradebook.csv`; selected by `Accept` containing `text/csv`. Text cells starting with `= + - @ tab CR` get a leading `'`.
- Nits applied: PATCH `dueOn: ""` is collected with other field errors (reported last in details, after position); numeric-string ponytail note moved to the decode site in putScores.
- `exists` rows and update-only/add-only commits: one `score.set` audit row only when something is applied.
## Requests
none
## Blockers
none
