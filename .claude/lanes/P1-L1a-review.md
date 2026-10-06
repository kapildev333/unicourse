# P1-L1a-review - review of P1-L1a auth primitives
Status: done
## Doing
Finished. Verdict: approve. Low findings only, none blocking.
## Found
- Gates (apps/api, test DB): gofmt -l clean, go vet clean, go build clean, go test ./... -count=1 -race all ok; all 17 auth tests ran, none skipped.
- Probes: Go strings.ToLower vs Postgres lower() (en_US.utf8, libc) over all 132,191 letters U+0080..U+2FFFF: 0 mismatches, so NormalizeEmail output always passes `email = lower(email)` on this DB. The invitations_by_email policy is `email = app.email AND app.university_id IS NULL`; `allowed` runs in WithUserTx("", email), so the policy applies (TestMagicSendInvitedExternalEmail confirms it).
- Consume classification is race-free: under READ COMMITTED the losing UPDATE waits for the winner's commit, and the follow-up SELECT then sees used_at (the race test gives 1 win and 9 "link already used").
- Low: NormalizeEmail has no length cap (a 500-char local part passes). Low: smtpMailer has no dial or IO deadline, and http.Server has no WriteTimeout, so a stalled SMTP server hangs the request. Low: NewSMTPMailer accepts a host without a port, which fails on every send instead of at startup.
## Requests
- to P1-L1c: httpx request log (middleware.go:94) logs r.URL.Path, which puts the magic-link token in the logs on GET/POST /api/v1/auth/magic-link/{token}. Design section 4 says the plaintext never reaches the logs. Redact the path segment for that route.
- to orchestrator: timing and SMTP-failure oracle on POST /auth/magic-link. A not-allowed email returns after one SELECT. An allowed one does an INSERT plus a synchronous SMTP send, and a mail failure gives 500 only for allowed emails. This reveals which domains are allowed and which emails have pending invitations. If it matters, decide in PLAN (send async).
## Blockers
None.
