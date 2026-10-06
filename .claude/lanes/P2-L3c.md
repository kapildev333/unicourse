# P2-L3c - Web: my sections and polish
Status: done
## Doing
Finished; awaiting orchestrator verification.
## Found
- `features/my-sections/my-sections.tsx` `MySectionsPage({ slug })`, wired into `routes/$slug/me.tsx`; `routeTree.gen.ts` unchanged (same route path). Terms with zero sections are not rendered; all-empty shows "No sections yet".
- Keyboard walk (by code reading + tests, no browser/API available; a real-browser pass is still owed): checked every screen of P2-L3 for keyboard reach (all controls are native button/a/select/input), visible focus (Button, inputs use focus-visible outline; links rely on the browser default ring, no `outline:none` anywhere in `index.css`), labels (every input goes through FormField, file input included), status in live regions (StatusMessage role=status, import status aria-live, errors role=alert), `th scope="col"` on all tables, text badges (RoleBadge/ActionBadge/invited chips), dialogs.
- Fixed: `Dialog` did not return focus to its trigger for conditionally mounted dialogs (CourseDialog, SectionDialog are unmounted rather than closed, so native close() focus restore never ran). `Dialog` now records the trigger before opening and refocuses it on close/unmount (test in `dialog.test.tsx`).
- No dead code or leftover casts found in L3a/L3b files (cast in courses-layout was already removed).
- Not done: in-browser tab walk (`npm run dev`) since no API runs here.
## Requests
None.
## Blockers
None.
