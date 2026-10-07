# P4-L1a - mail outbox table and package
Status: done
## Doing
Finished; gates green. Callers still send synchronously (L1b switches them).
## Found
- Exported surface for P4-L1b: `outbox.Mail{To, Subject, Text string; Urgent bool}`, `outbox.Enqueue(ctx, tx *sql.Tx, mails ...Mail) error`, `outbox.NewSender(db, mailer, log) *Sender` (fields `Poll`, `Backoff`, `MaxAttempts`, `DB`, `Mailer`, `Log`), `(*Sender).Run(ctx)`, `.Wake()`, `.Drain(ctx) error`. `outbox.Mailer`, `LogMailer`, `NewSMTPMailer` moved unchanged (log line identical).
- Migration `0003_mail_outbox.sql`; `migrate_test.go` now counts embedded migration files and expects 18 tables, `outbox_mails` in the no-RLS list.
- `tenant/invitation.go` was not in the brief's file list but used `auth.Mailer`; renamed to `outbox.Mailer` there (rename only) so the tree compiles.
- `lastURL`/`urlRe` now live in `auth/magic_test.go` (still log-buffer based; L1b changes to the DB read).
- `outbox_mails` bodies are not logged; `LogMailer` is the exception.
## Requests
## Blockers
