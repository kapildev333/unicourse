# P17-L2b-review - review of P17-L2b (UNICOURSE_SIGNIN_IP_LIMIT)
Status: done - approve
## Doing
Reviewed bb0eeaa (git diff 7092df0...HEAD, 6 files).
## Found
- No e2e test asserts the per-IP limit (only E-AUTH-12, per-email 5, unchanged); per-IP 429 is covered by Go handler_test.go (newHXLimit perIP, lines 224-263), so 100000 masks nothing e2e owned.
- Validation, error text and test mirror OUTBOX_SENDERS exactly; README row sits with the other optional numerics.
- Gates: gofmt clean, go vet ok, config + cmd/api tests pass (no SKIP), web typecheck ok.
## Requests
none
## Blockers
none
