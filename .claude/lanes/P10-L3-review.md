# P10-L3-review - review of Spec 6 content e2e
Status: done
## Doing
Finished: verdict changes needed (two small spec edits), lint and typecheck green; e2e not run (orchestrator owns the port).
## Found
- content.spec.ts:32 proves "emailed" only via the Emailed badge, which is `len(mails) > 0` (announcement.go:139-140): one recipient (the TA) is enough, so "emailed to the roster" is not asserted. Plan Task 3 asks to read the mail from the stdout mailer log (outbox/mailer.go:23 logs to/subject/text). Fix: expect.poll over apiLog for a "mail" line with to === students[0] and subject containing annA.title.
- content.spec.ts:183-190 student write-control checks miss "Edit <week|title>" and "Move ..." buttons; one regex /^(Add week|Edit |Rename |Move |Delete |Upload files)/ on both pages closes it.
- Nits: double blank line 148-149; student does not see announcement bodies asserted; prettier is not a gate (staffing.spec also fails it).
- Scope clean: only content.spec.ts, three fixtures, the lane log. Fixtures are tiny, self-authored, extension-allowlisted.
## Requests
- to P10-L3: the two fixes above.
## Blockers
