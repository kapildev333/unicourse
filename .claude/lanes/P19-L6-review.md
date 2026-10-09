# P19-L6-review - review of P19-L6 server timeouts
Status: done
## Doing
Verdict sent: changes needed (test gap on ReceiveFile; csvimport 400 vs 499)
## Found
- Deadline semantics verified against go1.27.1 net/http: readRequest sets t0+ReadTimeout per request, the idle/header path resets it before each next request, so an upload's extended deadline does not leak across keep-alive. statusWriter implements Unwrap, so ResponseController reaches the conn.
- Scratch test (outside the worktree) through newHandler + real content.ReceiveFile: no extension -> 499 logged; 10 s extension -> 201. Works, but the lane's own test never calls ReceiveFile (its handler calls SetReadDeadline itself), so deleting the ReceiveFile call or the uploadErr passthrough fails no test.
- csvimport.uploadErr has no deadline passthrough: a CSV upload past 30 s answers 400 "invalid multipart upload: read tcp ...: i/o timeout" and logs 400, not 499.
- DecodeJSON second Decode (trailing-value check) maps a deadline to 400 "must contain a single JSON value".
## Requests
none
## Blockers
none
