# P14-L1a-review - review of P14-L1a (API core backlog items)
Status: done (re-review 2026-10-07: approve)
## Doing
Finished; verdict changes needed (one line in httpx/error.go:77, one comment line).
## Found
- httpx/error.go:77 logs raw r.URL.Path. LogMiddleware logs only the route pattern so secret path params (magic-link {token}) never reach logs; this line breaks that rule. Not live today (consumeMagicLink's failRedirect turns ctx errors into a 500 httpx.Error first), but it leaks as soon as any handler with a secret in the path returns a ctx error. The request line already has status 499 + requestId, so drop "path" (test asserts on it: update error_test.go:100,108).
- Urgent mail now expires after 15 min, and the 15 is a separate literal from magic.go:45. Mail.Urgent doc (outbox.go:14) still says only "sent before other mail"; a future non-link urgent mail would be silently dropped. Fix: one comment line on Mail/Urgent naming the 15-minute give-up and magic.go:45.
- Nit: client abort during POST /auth/magic-link/{token} still logs "sign-in failed" ERROR + 500 via auth failRedirect default branch.
- Gates: gofmt empty, vet ok, httpx + outbox -race ok.
## Requests
none
## Blockers
none
## Re-review (9f13c66..a494fbd): approve
- Three fixes confirmed: error.go:77 logs requestId only (test asserts no path); outbox.go:14-15 and magic.go:45 cross-reference the 15-minute TTL; auth failRedirect returns ctx-done Canceled/ErrTxDone unchanged (oidc_test.go TestFailRedirectClientAbortIsReturnedUnchanged).
- audit Filter.where(): pure extraction, same conditions/order/$n for List and Count; Count numbers from $1, List appends cursor and limit after. entityId alone works (independent conditions); from >= / to < unchanged. Count needs no users join (conditions use a.* only); RLS on audit_log is forced.
- Contract additive (new response field); types.ts matches a fresh openapi-typescript run; postman collection unchanged against a fresh build; npm run typecheck ok.
- Nits: total not tested with an empty filter or with from/to (tenant/audit_test.go:66-75); one-line add: `total(t, e, adm, "")` equals len(get("")) . Scope widened to internal/audit, openapi.json, types.ts (orchestrator-approved per brief). P14-L1b also edits openapi.json (descriptions, other hunks): regenerate types.ts after whichever merges second.
- Gates: gofmt empty; go vet ok; go test -race -count=1 httpx outbox auth audit tenant ok; web typecheck ok.
