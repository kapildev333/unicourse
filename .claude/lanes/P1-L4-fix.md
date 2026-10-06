# P1-L4-fix - flaky sign-in page web test
Status: done
## Doing
Committed.
## Found
Could not reproduce the failure locally (0 fails in 20 single-file runs, 22 full runs, 300-iteration stress of the same scenario) before the change. Hardened `mockApi` in src/test-utils.tsx: no Request.clone() (stream tee); the stub reads the body text once and gives the recorded call and each handler their own freshly built Request.
## Requests
## Blockers
