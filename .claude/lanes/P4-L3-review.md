# P4-L3-review - review of P4-L3 (client IP behind trusted proxies, OIDC start limit)
Status: done
## Doing
Verdict: approve. No defects.
## Found
- Matches docs/design/P4-L3.md §2-§9 line for line; §3 algorithm verbatim, all 22 table rows present; §4 messages exact; `oidc-ip:` bucket checked before `p.begin` (no cookie on 429); no openapi change (§7).
- Gates (apps/api, UNICOURSE_TEST_DATABASE_URL exported): gofmt -l . empty; go vet ok; go test ./... -count=1 all ok; go test ./internal/httpx/... ./internal/auth/... -race -count=1 ok.
- Real process 127.0.0.1:18091, UNICOURSE_TRUSTED_PROXIES=127.0.0.1: forwarded client A 300x202 then 429; client B 202; A with `1.1.1.1, 203.0.113.11, 127.0.0.1` and `9.9.9.9, 203.0.113.11` both 429 (spoof ignored); OIDC start 300x200 then 429 `{"error":{"code":"rate_limited",...}}`, no Set-Cookie; magic link for the OIDC-exhausted client 202 and OIDC for the magic-link-exhausted client 200 (separate buckets); `{}` 400 and unknown provider 404 after exhaustion. Log line `"clientIp":"203.0.113.99"`, raw XFF entry and email absent; without the env var `"clientIp":"127.0.0.1"`.
- Startup: `nope`, `0.0.0.0/0`, `::/0`, `fe80::1%eth0` exit 1 with the §4 texts; bare `127.0.0.1` starts.
- Extra spoof/edge probes (scratch copy): two header lines, empty hops, `[::1]:443`, zone in hop, mapped hops/remote, NUL, bad port, `[]` all as designed, never panic.
- Mutations killed: leftmost walk (rows 4,5,7,10,11), no trusted-chain skip (6,7), no Unmap (17,19), no untrusted-remote check (1,2), middleware ignoring trusted (TestLogMiddlewareClientIP), OIDC limit removed / shared `ip:` bucket (TestOIDCStartRateLimit), magic link on RemoteAddr (TestIPLimitPerClientBehindProxy). Not covered by a test: dropping ClientIPMiddleware from cmd/api newHandler (design accepts; real-process check covers it).
- Scope: only §12 paths plus the design doc and lane log. No dead code; no other RemoteAddr users remain.
- Nits (no change required): (1) config.go parseProxies: an IPv4-mapped CIDR such as `::ffff:10.0.0.0/104` is accepted but never matches (hops/remote are unmapped); fails safe (trusts nothing). Unmap or reject when `p.Addr().Is4In6()`. (2) clientip.go:37 the ponytail comment precedes the `clientIP:` doc text, so the doc comment no longer starts with the name; move it below. (3) config.go `fmt.Errorf("zone")` is a placeholder error whose text is never shown; `errors.New` or a bool is plainer.
## Requests
## Blockers
