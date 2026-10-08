# P17-L3-review - review of P17-L3 (outbox senders, Forwarded, web origin, upload limit config)
Status: done
## Doing
Review finished: approve; one design-level follow-up escalated, three nits.
## Found
- Diff matches design P17-L1 §2, §5, §6.1, §7 and README rows almost verbatim; no scope creep.
- Outbox: no double send possible (row locked FOR UPDATE SKIP LOCKED for the whole send, deleted or rescheduled in the same tx; cancel rolls back). TestSendersDeliverEachOnce passed 6/6 incl. -count=5; peak==2 only fails if goroutine 2 is starved ~1 s; keep it.
- to orchestrator (design §5.4, owner to confirm): mixed proxy chains regress. Upstream proxy sets X-Forwarded-For only, inner trusted proxy sets both (e.g. CDN -> HAProxy `option forwarded` + `forwardfor`): XFF "203.0.113.7, 10.0.0.3", Forwarded "for=10.0.0.3", remote 10.0.0.2 -> 10.0.0.2 (verified), so every client shares one rate-limit bucket; before this lane it was 203.0.113.7. Smallest fix: in clientIP, when one walk ends on a trusted address (all its hops trusted) use the other walk; disagree only when both end untrusted. A client can still only push a trusted address, so it cannot pick its key.
- nits: empty `for=` / `for=""` is skipped as blank, not "unknown" (doc comment says invalid token -> unknown); WEB_ORIGIN accepts `http://x:`, `http://x:0`, `http://x:99999` (empty/invalid port gives an invalid CSP form-action source); drainAll lacks the §2.4 round ceiling as a `ponytail:` comment.
## Requests
none
## Blockers
none
