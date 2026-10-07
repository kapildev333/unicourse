# P11-L1-review - review of unicourse-admin seed-demo (lane/P11-L1-seed-demo)
Status: done
## Doing
Review complete. Verdict: approve (no blocking defects; nits below).
## Found
- Fidelity: every handler-backed write goes through app.NewRouter in process as the design's persona; direct SQL only for university, users, memberships, department_admins, invitation (+display_name), sessions. Sessions revoked in a defer with WithoutCancel, joined into err; failure test proves 0 sessions. Outbox 0 asserted.
- --reset: slug select + delete by that id in one WithPlatformTx; file removal filters on that id only. Correct by reading; no test seeds a second university to prove isolation (design did not ask).
- demo.json is byte-identical to design Appendix A; files/ byte-identical to apps/web/e2e/fixtures.
- Worker deviations accepted: run.go split (design's own ~600-line rule; seed.go+run.go = 911 lines); multipart filename = fixture file name (content.fileName derives content type from the extension, so "file" would be rejected).
- Nits: (1) run.go:433 `cmp` helper shadows stdlib package name and duplicates `cmp.Or` (go 1.27). (2) seed_test.go:89-92 "override outside roster" uses invited Ingrid, so it hits the invited branch, not the not-in-roster branch of seed.go:298. (3) seed_test.go:188 incomplete check does not pin Lucas Moreau. (4) seed_test.go:304 Walk error ignored. (5) run.go:469-473 csv Write/Flush errors unchecked (cw.Error()); harmless on bytes.Buffer.
## Gates (apps/api, test DB env set)
gofmt -l . -> empty; go vet ./... -> exit 0; go test ./internal/seed/... ./cmd/unicourse-admin/... -count=1 -race -> ok seed 4.868s, ok unicourse-admin 7.506s
## Requests
- to orchestrator: optional follow-up test that --reset leaves a second university's rows and blobs intact.
## Blockers
none
