# P9-L6-review - review of P9-L6 (entry pages, one search, separators, e2e selectors)
Status: done
## Doing
Finished; verdict request changes (one medium finding), reported to orchestrator.
## Found
- D80: all e2e hunks are §5.7 items (helpers.ts:47 label; screens.ts:97 and :143 label; screens.ts:215 separator; staffing.spec.ts:38 label, :140-141 and :159-160 and :386-387 separators, :212-213 searchbox + Enter). No other e2e edit.
- MEDIUM app-shell.tsx:65-66 `key={q ?? ''}` remounts the header input whenever q changes, so after Enter (on the Catalog or from any page) focus drops to <body>; reproduced in a scratch jsdom test (activeElement is body). Fix: drop `key`, make the input controlled with a `[text, seenQ]` state that follows q (the pattern the old catalog.tsx used).
- LOW sign-in-page.tsx:83 and :107 `writeLast` runs before the request succeeds, so a 404/429 attempt still becomes "Last used". Fix: call it after `unwrap` resolves (before `location.assign`, and after `post(v)`).
- NIT app-shell.test.tsx:90 title says "is empty elsewhere" but never checks another route.
- NIT magic-link-sent.tsx:45-50 inline link classes are a one-off recipe (consistent with ~15 existing inline link class strings, so no new debt beyond them).
- Verified OK: magic link trims and posts {email,next}, same Check your email page; providers send email/slug exactly as before; invite route untouched and prefill works (screen 03); resend timer cleared on unmount and disabled synchronously (no double post); localStorage guarded both ways; "Last used" makes the accessible name "Email me a sign-in link Last used", e2e uses substring match so unaffected; brand marks currentColor + aria-hidden; no literal colours in diff; no " — " separators left in web source, lone dash placeholders intact.
- Gates (in worktree): lint ok, typecheck ok, test 46 files / 334 passed, build ok, e2e 2 passed, screens 1 passed with empty WIDE list.
## Requests
## Blockers
