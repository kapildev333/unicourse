# P14-L1a-review - review of P14-L1a (API core backlog items)
Status: done
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
