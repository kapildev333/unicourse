# P14-L2-review - review of P14-L2 web shared backlog
Status: done
## Doing
Review finished: changes needed (one a11y fix, one phone-row risk for the browser pass).
## Found
- Gates re-run in .worktrees/P14-L2/apps/web: lint clean, typecheck clean, vitest 58 files / 416 passed.
- Branch base is c685720; `git diff main..HEAD` also shows main's c1ccf1a (PLAN.md, .claude/agents) reversed. The landing (`main...HEAD`) is in scope.
- account-menu.tsx:76 aria-label="Account" while the visible text at md+ is the user's name: fails WCAG 2.5.3 Label in Name.
- Phone header row (D93): about 52 px more controls in row one; at 390 px with coarse pointer (ThemeSwitch 3 x 44 px) the controls group probably wraps under the wordmark. The orchestrator should check at 390 px with touch emulation.
- C6 audit total not delivered (API has no total); "N or more" is used instead. No Phase 14 lane adds `total`; this needs an orchestrator decision.
## Requests
- to orchestrator: confirm the phone row at 390 px with touch emulation; decide on the C6 total (API lane or accept "or more").
## Blockers
