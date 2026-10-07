# P6-L1a - Spec 3 Task 1a: packages, tokens, theme, switch, gate, e2e
Status: done
## Doing
Committed on lane/P6-L1-ui-foundation.
## Found
- vitest blanks `?raw` imports of `.css` (returns ""), so `theme.test.ts` reads `src/index.css` with `node:fs` (`/// <reference types="node" />`, path relative to cwd apps/web) instead of the design's `import css from '../index.css?raw'`.
- No `@testing-library/user-event` in the project: theme-switch test uses `fireEvent`.
- `@theme static`, `not-aria-pressed:`, `aria-pressed:` and the font `@import`s all compile as designed; no deviation.
- Gate test passes on whole tree; theme switch sits before the email span in app-shell.
## Requests
## Blockers
