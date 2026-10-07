# P4-L1b - switch mail callers to the outbox
Status: done
## Doing
Finished; gofmt, vet, go test ./... and npm run e2e green.
## Found
- All callers enqueue; API process runs the sender (cmd/api/main.go); CLI captures URL and enqueues nothing (D28); /readyz unchanged.
- `tenant.env.mails()` and `course` mail counts now read `outbox_mails`.
- README: SMTP row extended. Diagnosis query is in docs/design/P4-L1.md §5 (not added to README).
- apps/web/node_modules is a symlink I created (untracked/ignored); remove it with the worktree.
- review fixes applied: 3439e3f
## Requests
## Blockers
