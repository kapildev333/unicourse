# P21-L4-review - review of P21-L4 (web bell, notifications page, mail opt-out)
Status: done
## Doing
Review finished at 03e889e. Verdict: approve, with one layout risk the orchestrator has to check in a browser.
## Found
- Gates on a git-archive snapshot of HEAD: lint 0, typecheck 0, vitest 608/608 (no-literal-colors 2/2), build ok, and the regenerated routeTree.gen.ts is identical to the committed one.
- Mutation check: removing any one of these rules fails a test: count hidden at 0, 99+, mark read on link, mark all, settings PATCH only on change, checkbox disabled while loading, bell max-md:w-10. Removing the explicit aria-current does not fail a test, because TanStack Link sets it already.
- Risk F1: with pointer-coarse (a real phone), ThemeSwitch is about 142 px wide. Logo plus bell, theme and account then come to about 361 px, against the 342 px content width at 390, so the right group may wrap to its own row (D93/D129 one-row bar). A desktop resize to 390 uses a fine pointer, so it will not show this. Check it with touch emulation (hasTouch/isMobile).
## Requests
- to orchestrator: in the browser pass at 390, emulate touch (pointer: coarse) and confirm the header row 1 still holds the wordmark, bell, theme and account.
## Blockers
