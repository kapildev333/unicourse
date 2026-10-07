# P9-L1a - shared components (Spec 5 Task 1a)
Status: done
## Doing
Finished; waiting for orchestrator.
## Found
Signatures (all in apps/web/src/components/):
- `Table`: new `stack?: boolean`; each `td` needs `data-label`. Scroll wrapper now inside a `min-w-0` div.
- `Loading`: `<Loading />` unchanged; `<Loading variant="table" columns={string[]} />`; `<Loading variant="cards" />`; `label?` on all.
- `Card`: new `footer?: ReactNode` (pass your own `<Link>`; rendered outside the main link). Band `h-16`; badge row `mt-auto pt-3`. The `<li>` now carries the border/hover; the main link is inside it.
- `Dialog`: always renders a "Close" icon button (last in DOM, top right so autofocus still lands on Cancel). Phone width/height classes; `DialogFooter` is sticky.
- `SegmentedControl<T extends string>({ label, options: {value,label}[], value, onChange })` (segmented-control.tsx).
- `Disclosure({ summary: ReactNode, children, className? })` (disclosure.tsx).
- `LoadMore`: new `showCount?: boolean` (default true).
- Token `line-control` (css + Tailwind `border-line-control`); secondary Button uses it. Ratios in docs/design/P6-L1.md.
- NEEDS OUTSIDE MY SCOPE: apps/web/src/lib/theme.test.ts line `expect(a).toHaveLength(19)` must become 20 (the dark block now has 20 declarations). I did not edit it; `npm test` is red on that one test until the orchestrator or a lane owning src/lib changes it.
## Requests
- to orchestrator: change theme.test.ts 19 -> 20; add line-control, SegmentedControl, Disclosure to CLAUDE.md UI conventions.
## Blockers
Gates: lint clean; typecheck clean; build ok; e2e 2 passed; test: 1 failed (theme.test.ts, above) | 256 passed (257).
