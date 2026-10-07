# P9-L9b - Phase 9 closing audit, group b (screens 12, 13, 40-47, 50-54), docs/design/P9-L9b.md
Status: done
## Doing
Done: docs/design/P9-L9b.md written (not committed; orchestrator commits)
## Found
- All 25 group-b 390 captures (AFTER, light and dark) are 390 px wide (before: 43/44 693, 45-47 555, 52 561, 53 679).
- Addressed criteria still below 4: only 53 (g)7 = 3 (stacked Invitations 4589 px tall). 54 (g)5 and (g)6 = 3 wait on C6/C5.
- Root cause of weak stacked rows: the `ui-table-stack` block sits in `@layer components`, so the `font-medium` utilities on first cells (`course-list.tsx:134`, `roster.tsx:95`, `section-list.tsx:187`) beat its bold head; fix F1 moves it to `@layer utilities`, adds a label-value grid and `td[data-empty]` hiding.
- Admin puts TabNav above the per-tab h1, Courses puts h1 above TabNav (F3).
- Meeting-row Remove still accent (`meetings-editor.tsx`), against D87 B6 (F5).
- Same en dash owner question as P9-L9a Q1 (recommend keep in ranges); seed staff names (P9-L9a R4) also needed for group b (F2).
## Requests
- to P9-L9c: group b dark (41, 46, 52, 54) and phone widths are in P9-L9b §0 and §1 for the app-wide (l) and (m) rows; none of my screens is wide.
## Blockers
