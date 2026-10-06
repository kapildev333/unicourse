# P0-L4-review - review of openapi.json, route registry, contract test
Status: done (changes needed)
## Doing
Review finished; verdict reported to orchestrator.
## Found
- Contract shape matches design §3-§5 and §8 exactly (43 ops, methods, params, bodies, success codes, schemas, required/optional, enums, no null, list wrappers, multipart file/checksum). types.ts is generator output (regen diff clean).
- Contract test hole: `app/contract_test.go:73` strips then re-adds `/api/v1`, so `r.Handle("GET", "/me", h)` passes the test but serves `/me`, not `/api/v1/me` (proved on a scratch copy).
- "Who:" descriptions diverge from spec §6 for sections (any member reads), imports (instructor for enrollments), enrollments (instructor delete, TA read missing), getSection (none). Domain lanes must take authz from spec §5/§6 + their design, not these lines, until fixed.
- deleteSection 409 is unreachable under D15 (all section children cascade).
- `app.NewRouter` is not mounted in `cmd/api/main.go`; the first domain lane must mount it under `/api/v1/`.
- P1-L1 lists callback error codes incl. `unauthenticated`; contract descriptions list forbidden, validation_failed, rate_limited, no_membership.
## Requests
- to orchestrator: rule on departments readable by any member (P1-L2 escalation) and on the items above before P0-L4 lands.
## Blockers
