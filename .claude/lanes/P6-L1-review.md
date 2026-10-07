# P6-L1-review - review of P6-L1 (a+b) landing fca075c..7461bd8
Status: done
## Doing
Review complete: approve; two low-severity findings for the gate and one for touch sizing.
## Found
- Gates green (lint, typecheck, test 27 files/176 tests, build, e2e 2/2). The worktree's node_modules symlink points at main's install, which lacks lucide-react and the fonts, so typecheck fails until the packages are installed. I ran the gates on a private `npm ci` and then restored the symlink. Orchestrator: verify on a real install.
- Contrast: all 72 measurements in design §1.2 re-measured from index.css values, 0 mismatches. Derived shades match their stated mixes.
- Gate gap (low): `no-literal-colors.test.ts:10` misses stacked variants (`md:dark:bg-canvas`, `hover:dark:x`). Fix: add ':' to the lookbehind class, `/(?:^|[\s'"`{:])dark:/` (0 false positives on today's tree). Arbitrary named colours (`bg-[white]`, `text-[black]`) also pass, which is a spec loophole rather than a listed item.
- Nit: Tailwind scans the gate test, so the sample '"dark:bg-surface"' emits a dead `.dark\:bg-surface` rule in dist CSS. Fix: split the sample string ('"dark' + ':bg-surface"').
- Low: on a coarse pointer the theme buttons become 28x44 inside an h-9 (36 px) group: they overflow its border and stay 28 px wide (spec §7 asks for 44 px).
- For Tasks 2-5: sign-in-page.tsx `shadow-sm` now renders nothing (default shadows removed), as expected until P6-L2. Loading is now a div, so never render it inside a <p>.
## Requests
- to orchestrator: decide whether to fix the `:dark:` gate gap before Tasks 2-5 (a one-character change).
## Blockers
