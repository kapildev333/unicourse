# P16-L9-review - review of a11y.spec.ts (axe sweep, D125)
Status: done
## Doing
Nothing; verdict approve, two nits.
## Found
- Every page and dialog in design A.8 (E-A11Y-01..06) is scanned in light and dark with the design's tags, and the filter is exactly serious/critical. Each dialog is awaited by its accessible name.
- Probe (scratch copy, deleted afterwards): html[data-theme] is null on every scan, and the html background is rgb(247,249,250) in light and rgb(28,29,31) in dark. There are 0 skeletons and 0 running animations at scan time, and dialog[open] = 1 on every dialog step. An injected <img> without alt failed the case with "image-alt: img", so the assertion does catch violations.
- The animation wait is correct: `.catch` swallows only rejections from cancelled animations, and running ones are still awaited. It is not a sleep, and there are no infinite animations in the app.
- package.json has one exact-pinned dev dependency, and the lock adds only @axe-core/playwright and its nested axe-core 4.13.0.
- Nit: a11y.spec.ts:54 waits on `body`, but the confirm page has an h1 ("Sign in to UniCourse"), so the default h1 wait would work.
- Nit: a11y.spec.ts:7,10 uses `let P = seedDemo(slug)` in beforeAll, where design 0.2 says `const P = personas(slug)` at module top. It works the same way.
## Requests
none
## Blockers
