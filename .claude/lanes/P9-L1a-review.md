# P9-L1a-review - review of P9-L1a (shared components, Task 1a)
Status: done
## Doing
Finished: verdict changes needed (one regression).
## Found
1. HIGH apps/web/src/components/dialog.tsx:44 `sm:max-h-none` drops the UA cap (max-height calc(100% - 2em - 6px), overflow auto) on desktop. Measured in Chromium at 1440x900 with 60 preview rows in a `wide` dialog: main = 862 px tall, footer at y 856 (scrolls); branch = 3027 px tall, top at y -1050, footer at y 1938, unreachable (fixed, no scroll). Hits csv-import, scores-import and finalize previews with many rows. Fix: delete `sm:max-h-none` (keep `max-h-[calc(100dvh-2rem)]` at every width).
2. LOW dialog.tsx:64 sticky DialogFooter sticks at the dialog's padding edge, so scrolled content shows through the 24 px bottom padding under the buttons (390 screenshot). Fix: `sticky -bottom-6 -mb-6 pb-6` on the footer.
3. LOW index.css stack rules target only `td`; a `th scope="row"` in tbody (finalize-dialog.tsx:162, scale-editor.tsx:46, gradebook.tsx:478) stays a table cell inside a block row if those tables ever pass `stack`. Fix: add `.ui-table-stack tbody th` beside `td` in the stack selectors, or document "td only" in the Table comment.
4. NIT index.css: `td::before` label inherits bold in the first cell (label "Name" renders bold); add `font-weight: 400` to the ::before rule. `td:last-child { width: 100% }` is a no-op on a block td; delete.
5. NIT index.css:72 `--color-line-control` in the prefers-dark block is indented 2 instead of 4. card.test.tsx:3 second import line from '@tanstack/react-router'; merge with line 1.
Verified OK: token values equal ink (light) and line-strong (both dark blocks); contrast via §1.3 script: light 16.87 on surface, dark 3.88 on surface, 4.86 on canvas (claim matches). Stack CSS scoped to .ui-table-stack only. Card: one main link + sibling footer link, no nesting; inset focus ring not clipped. Dialog Close: type=button, aria-label+title, focus ring from Button, last in DOM so showModal focus lands on the first control; close returns focus (test). No dark:/hex/palette in feature code. Scope: only brief files plus orchestrator commit (theme.test.ts, CLAUDE.md).
Gates (mine): lint ok; typecheck ok; test 41 files 257 passed; build ok; e2e 2 passed (port free, no bind collision).
## Requests
- to orchestrator: send finding 1 (and 2) back to P9-L1a before landing.
## Blockers
