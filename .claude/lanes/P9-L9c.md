# P9-L9c - Phase 9 closing audit, group c (grading, states, dark, phone): docs/design/P9-L9c.md
Status: done
## Doing
Done: docs/design/P9-L9c.md written (not committed; orchestrator commits)
## Found
- Width: 71 of 71 AFTER `*-390.png` are 390 wide (BEFORE 46 of 67; the 60-63 set measured 542, not 550).
- Contrast (index.css after tokens, P6-L1 §1.3 functions): dark ink-muted on surface 5.63, dark line-control on surface 3.88; success/warning soft pairs 5.39-7.77. All pass.
- Still below 4: 62 (h)6 phone 3, 65 (j)4 hidden grades 2, 66 (g)7 phone 3, (m)4 tables collapse 3.
- Cross-cutting CSS (affects every Table, all groups): `.ui-table th` paints body row headers canvas (two-tone column, sampled in 62); `.ui-table tbody tr:last-child td` skips the row-header th (stub hairline under the last row, 60/63/64/73).
- `font-variant-numeric: tabular-nums` in index.css is a no-op (DM Sans has no tnum, per P9-L1c).
- Answer to P9-L9a and P9-L9b: (m)2 scored n/s app-wide as you suggest; your dark scores folded into the app-wide (l) row (none below 4). My stack fixes for 62 and 66 depend on P9-L9b F1.
- Correction for P9-L9a R4: student display names do show in the grading captures (60-65: "Amara Okafor" over the email), filled by the roster CSV import; staff (elena.marsh etc.) still show as emails.
## Requests
- to P9-L9b: seen in your captures, 71/43 Term select option uses an en dash range and is cut at 288 px (`features/courses/section-list.tsx:255`); 46 stacked import preview labels rows "email"/"displayName" (raw CSV keys) and meeting times use an en dash. Yours to score.
## Blockers
