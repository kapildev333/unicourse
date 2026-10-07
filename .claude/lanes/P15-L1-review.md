# P15-L1-review - review of magic-link confirm page CSP form-action
Status: done
## Doing
Finished. Verdict: changes needed (README only); code approved.
## Found
- handler.go formAction/origin correct: scheme://host[:port] of WebBase, path stripped, same origin keeps 'self', nil Magic / unparsable falls back to 'self'. Only operator config reaches the header, no request input. Other directives unchanged.
- Only <form> in internal/ is the confirm page; OIDC callback and sign-in errors are plain 302s with no CSP.
- README.md:63 new sentence is wrong: without UNICOURSE_WEB_ORIGIN, WebBase = BaseURL, so the redirect lands on http://localhost:8080/northfield/me (404 JSON envelope when no WebDir), not on the confirm page.
- Gates: gofmt -l . clean; go vet ./... clean; go test ./internal/auth/... -count=1 -race ok (39s).
## Requests
- P15-L1 worker: fix README.md:63 sentence (see Found).
## Blockers
