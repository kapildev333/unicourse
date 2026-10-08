# P16-L2-review - review of the e2e harness lane P16-L2
Status: done (verdict: changes needed, one small fix)
## Doing
Review finished; report sent to the orchestrator.
## Found
- helpers.ts:45 + :147 `mails(logLines())` skips the first mail written after the snapshot: the API log ends in "\n", so `logLines()` is the line count + 1 and `slice(n)` starts one past the new line (signIn hides this with `before - 1`). A "no mail sent" assertion would pass vacuously. Fix: `logLines = () => ...split('\n').length - 1` (index of the next line) and `latestMailLink(email, before)` in signIn; nothing else in e2e/ calls logLines. Until fixed, area lanes (L3 auth, L4 admin) must pass `logLines() - 1`.
- Port 18112 was held by an orphaned API from a concurrent `playwright test` run in this same worktree (two runs share e2e/.tmp, so one teardown pid is lost). Not killed; gates run at 18113. Two runs in one worktree also delete each other's .tmp binaries (my first demo run failed in seedDemo because of it).
- Everything else matches §D.2 and §A.1: personas == demo.json, pageAs never signs out, regex escaping right, invite links `/invite/<token>?email=` match the `'/invite/'` path, sectionIds shape and limit 200 match the API.
## Requests
- to orchestrator: relay the mails/logLines fix (or the `logLines() - 1` rule) to P16-L3 and P16-L4 before they code against mails().
## Blockers
