# P21-L2-review - review of P21-L2 notifications landing
Status: done
## Doing
Verdict approve; gates green on ab4b33b (gofmt, vet, go test notification content grade pg app, typecheck, no contract drift).
## Found
- DDL byte-identical to design §1. Producers match §2.1 recipients; no other writer of announcements/assessments.published/final_grades exists.
- types.ts and Postman regenerate with no drift; web typecheck green; gofmt/vet clean.
- Minor: pg tx_test cases notifications->sections and notifications->memberships are the same statement (B section + B user), so neither pins its own FK.
- Minor: notification_test TestListOwnOnly claims a B row but inserts none (dead `cb`); cross-university isolation rests on pg RLS tests.
## Requests
## Blockers
