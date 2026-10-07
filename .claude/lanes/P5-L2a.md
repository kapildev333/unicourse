# P5-L2a - schedule helpers, term picker, My sections list/schedule (web)
Status: done
## Doing
Finished; gates green (lint, typecheck, 145 tests, build). Awaiting orchestrator.
## Found
- `src/lib/schedule.ts` exports exactly the design's signatures: overlaps, clashesFor, currentTerm, classCode, formatMeeting, toMinutes, assignLanes, dayShort, dayLong, type Slot/Meeting.
- `features/my-sections/term-picker.tsx` exports `useTermChoice(slug, termId?)` and `TermPicker` (L2b reuse). Note line is rendered by the page, not the picker; L2b copies the `<p>` from the design.
- `my-sections.tsx` exports `useMySections(slug)` (key `[slug,'my-sections']`) and `MySectionsPage({slug, termId, view, onSearchChange})`.
- Deviation 1: `defaultTermId` in `features/courses/queries.ts` NOT rewritten on `currentTerm` (outside my scope); behaviour is identical, `useTermChoice` uses it as is. Whoever owns courses/ (L2c) may do the refactor.
- Deviation 2: the "Browse the catalog" link targets `/$slug/catalog`, which does not exist until L2b. It is written with a cast (`as unknown as { to: '/' }`) so typecheck passes now; L2b should drop the cast in `my-sections.tsx` once the route exists (it then typechecks plainly).
- `routeTree.gen.ts` unchanged by my work (search validation only).
## Requests
- to P5-L2b: remove the cast on the "Browse the catalog" link (see above).
## Blockers
