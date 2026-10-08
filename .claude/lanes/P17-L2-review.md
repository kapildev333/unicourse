# P17-L2-review - review of P17-L2 (rate_buckets, migration 0008)
Status: done
## Doing
Review finished: approve.
## Found
- Diff matches design P17-L1 §1 line for line (plus the ruled set_updated_at trigger). Upsert is atomic; expires_at <= now resets; hits > max denies (6th of 5); prune by absolute expires_at is safe across buckets; fail closed returns a plain error (500).
- Gates: gofmt clean, go vet clean, go test -v ./internal/auth/... ./internal/pg/... ok, 0 SKIP.
- Real HTTP on unicourse_e2e_12 / 127.0.0.1:17112: a@ x5 202, 6th 429, b@ 202, " A@X.EDU " 429 (same normalised bucket). rate_buckets: magic-link-email a=7, b=1, magic-link-ip 127.0.0.1=8, window 3600 s. API log carries no email.
- Nit: with the trigger, `updated_at = $3` in the upsert's update set is overwritten by now(); harmless.
## Requests
## Blockers
