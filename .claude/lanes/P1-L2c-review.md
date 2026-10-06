# P1-L2c-review - review of memberships, invitations, audit-log route
Status: done
## Doing
Finished. Verdict: approve. Test gap and nits are in the report.
## Found
- Gates green: gofmt clean, go vet ok, go build ok, go test ./... -count=1 -race all ok.
- Probes through the real HTTP pipeline matched design and contract: privilege, cross-tenant 404, last-admin, invitation lifecycle, SignIn acceptance, mail-failure rollback, and audit filters, actor and equal-`at` pagination. A 40-round concurrent demote/delete race of the last two admins always left exactly 1 active admin.
- Mutations: 8 of 9 killed. Survivor: counting `invited` admins in guardLastAdmin. The code is correct; no test pins it.
- Rebase checks: one NUL guard in TrimLen; pending_routes.txt has 12 lines; helpers_test.go is identical to main.
## Requests
- to P1-L2c (via orchestrator): add the invited-admin last-admin test; fix the stale "not visible" Actor comment in audit.go.
## Blockers
None.
