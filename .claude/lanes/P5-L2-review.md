# P5-L2-review - review of P5-L2 web landing (L2a, L2b, L2c) at 9469fa4
Status: done
## Doing
Finished. Verdict: approve (two small non-blocking defects, nits below).
## Found
Gates (apps/web, run by me): lint clean; typecheck clean (app + e2e); vitest 24 files / 162 tests passed; build ok (368.50 kB js).
Probes (scratch copy, 18 extra tests, all pass): overlaps boundaries incl. touching, 1-minute overlap, other weekday, identical, self; clashesFor once per section with 3 overlapping meetings, self excluded; currentTerm inclusive bounds / between terms / after all terms -> undefined, defaultTermId -> latest past; My sections after every term shows "No term is in session today. Showing Fall." with the picker on it; grid 10:00-11:15 -> gridRow 121 / 196 at 08:00 start, 600 rows default, 07:30-19:10 stretches to 07:00-20:00 (780 rows, 13 hour labels), clashing Monday blocks in columns 1 and 2 with border-dashed, touching Wed block plain; section/h3/ol per weekday; catalog clash only against the picked term, 11:15/11:15 touching no note, Load more sends cursor and termId, "Showing 3", button gone; "No sections in Fall 2026"; catalog renders when /me/sections fails; real-timer debounce (100 ms gaps) = one request; editor remove+add -> meetings[1].endsAt lands on the 2nd sent row and is focused; PATCH only-meetings body; labels + type=time on every row; nav order per role (member: My sections, Catalog only).
Mutations: overlaps `<=` -> 2 tests fail (schedule.test, catalog.test); editor `j = i` -> 1 test fails (section-dialog.test).
D23/D40: no toasts, no dark classes, no fetch outside api/, only existing colour roles, package.json/lock untouched.
Own-role badge on detail IS in the design (§0 #15, §5.3), not scope creep. L2a's cast request to L2b: done (no cast left).

Defects (non-blocking):
1. apps/web/src/features/courses/queries.ts:33-40 - defaultTermId not rebuilt on currentTerm (design §1 last para, §8). L2c's reason ("not a pure replacement") is wrong: the design adds the latest-starting fallback on top. Two copies of the same in-session/next rule. Fix: `return (currentTerm(terms, today) ?? [...terms].sort((a, b) => a.startsOn.localeCompare(b.startsOn)).at(-1))?.id`.
2. apps/web/src/features/courses/meetings-editor.tsx:47 with section-dialog.tsx:99 - a `meetings` (20-row) error has no aria-invalid control and hides the banner, so useFocusFirstError focuses nothing (probe: focus stays on "Create section"). role="alert" still announces it and the UI cap makes it near-unreachable. Fix: give the <p> `tabIndex={-1}` and a ref focused when fields.meetings appears, or let the banner render when the only field is `meetings`.

Nits:
- Formatting churn: section-dialog/section-list/section-detail `git diff --stat` 101/84/25 vs `-w` 73/52/25; plus ~45 more lines are pure 80-column re-wraps (useCursorList call, DELETE call, ErrorBanner, term select, number input, onlyDept, Course hint, detail api.GET) against a house width of ~120 (168 lines >100 cols in courses/admin). Read every hunk: hides no behaviour change, but revert the re-wraps or add a prettier config so it stops recurring.
- catalog.tsx:28-34 - if `q` leaves the URL from outside (nav "Catalog" click on /catalog?q=x) the stale `text` is re-pushed after 300 ms, so search cannot be cleared that way. Design §0 #10 accepted the input keeping text; sync `text` from `q` when `q` changes externally if it bothers anyone.
- catalog.test.tsx:102 - debounce test fires keystrokes synchronously and waitFor stops at the first +1, so it would not catch a later extra request. The probe with 100 ms gaps passes; consider that form.
- section-dialog.tsx:39 - edit mode still fetches /courses it never shows.
- validation_failed with empty details (body too large) now also shows "Add at most 20 meeting times." (existing useFieldErrors behaviour for every copy key).
## Requests
- to orchestrator: defects 1-2 are small follow-ups for a courses/ owner; no PLAN.md decision needed.
## Blockers
none
