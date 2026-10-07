# P4-L1-review - review of P4-L1 (mail outbox, sub-lanes a and b)
Status: done
## Doing
Verdict: changes requested (one test gap; product code is correct as built).
## Found
Defects:
1. apps/api/internal/auth/magic_test.go:207-224 (TestMagicSendQueuesUrgentMailAndWakes) and handler_test.go:485 do not reliably catch Wake firing before commit. Mutation (Wake moved inside the WithUserTx closure) failed 1 of 5 runs of `-run Magic -count=5` (only TestMagicLinkDoesNotWaitForMailer, by timing luck). Wake-after-commit is the one ordering the design insists on (§0.6, Wake doc). Fix: in the Wake counter, also read `select count(*) from outbox_mails` on db (another connection) and assert it was 1 at wake time; an in-tx wake sees 0, deterministically.
Nits:
- outbox_test.go:234 TestLockedRowIsSkipped: with `skip locked` removed it hangs until the 10 min go test timeout instead of failing (TestTwoSendersNeverDoubleSend still passes then, since plain FOR UPDATE also prevents double sends). Give Drain a 2 s context there so the mutation fails fast.
- outbox/mailer.go:35 ponytail comment ("move to a queue ... when volume or latency matters") is stale now the queue exists; replace with the §8 ceilings (one send at a time per process; at-least-once, a SIGTERM or crash between SMTP accept and commit re-sends). No ceiling is marked in outbox.go itself.
- auth/magic.go:24-26 doc comment line 25 is unwrapped (~170 cols).
- pg/tx.go:25 WithPlatformTx doc still lists only universities, users, sessions, magic_links; outbox_mails is now a platform table too (outside lane scope; for the next lane touching pg).
- outbox/mailer_test.go TestLogMailer does not assert subject (design §6 says it does). Line itself is byte-identical to main (rename, only `package` changed).
- handler_test.go:505-509 the "failing mailer still gives 202" half never runs s2, so it asserts nothing beyond the first half.
- cmd/unicourse-admin: `newMailer`/`failMailer` names survive while they now enqueue; harmless.
- Timing gap measured (MagicLinks.Send, 200 runs, local): allowed median 1.16 ms vs not-allowed 0.74 ms; the gap is mostly the commit WAL flush on the writing path, so "sub-millisecond" in design §8 can be a few ms on slow disks. Still the documented ceiling, nothing worse (rate limit runs before Send on both paths; no SMTP in the request).
Probes run (scratch copy of apps/api, not the worktree): rollback/commit-failure atomicity (deferred constraint trigger: Send errors, 0 magic_links, 0 outbox rows, 0 wakes); 3 senders on 2 separate pools over 200 rows: 200 sends, 200 distinct, 0 rows left; backoff schedule from DB clock 30,60,...,1920 s, 8th attempt gives up, failed_at set, body '', log "mail gave up"; urgent sent first behind 50 due non-urgent; explain over 50k rows uses the partial index, 0.02 ms; trigger outbox_mails_set_updated_at present; no RLS, table count 18 in migrate_test.
## Requests
## Blockers
