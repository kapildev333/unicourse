# P14-L1c-review - review of seed, blob and CLI backlog items
Status: done
## Doing
Finished: verdict approve, nits only.
## Found
- RemoveUniversity: strict lowercase uuid via d.check, os.RemoveAll on root/<id>; a symlinked <files>/<id> is unlinked, target untouched (probed). Missing dir is nil. Seed reset keeps warning semantics (§5), test proves the other university's directory survives.
- prune-files: .tmp 24h by mtime, blobs 1h unchanged, output/exit codes consistent, README matches.
- cw.Error checked after Flush.
- Nits: blob_test.go ignores errors from Put/MkdirAll/WriteFile (`tmp, _ := ...; _ = tmp`); WalkTemp follows a symlinked .tmp dir and fails the whole run if .tmp is a file (operator-only, same trust as Put); main.go pruneFiles doc comment has an odd line wrap.
- Gates: gofmt -l clean, go vet clean, go test seed/blob/unicourse-admin -race pass.
## Requests
## Blockers
