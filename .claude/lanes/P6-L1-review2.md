# P6-L1-review2 - second review of P6-L1 landing (main...5cbec34)
Status: done
## Doing
Review complete: approve; three low findings, one spec/design tension to settle.
## Found
- Gates (run here, symlinked install): lint 0, typecheck 0, test 28 files / 179 passed, build ok, e2e 2/2 passed (staffing 6.2 s, theme 0.4 s).
- index.css is byte-identical (ignoring whitespace) to design §2; all 18 tokens match the §1.3 script (light once, dark twice). Re-measured from index.css: accent/canvas 4.58 / 6.06, line-strong/surface 5.08 / 3.88, warning/warning-soft 5.39 / 7.00 - all equal design §1.2.
- dist: no Serif, DM Sans woff2 only, no Google font hosts, theme-boot.js present, no dead `.dark\:` rule.
- Low: no-literal-colors.test.ts:7 misses `bg-[red]`, `text-[rebeccapurple]`, `bg-[color:white]`; :8 misses `hwb(`, `color-mix(`. Widened rules have 0 hits on today's tree.
- Low (a11y, 2.5.3): header search's name "Find a course" differs from its only visible text, placeholder "Search the catalog" (app-shell.tsx:44-45). Fix: placeholder "Find a course".
- Spec tension: card band `bg-accent-soft` is ~300x96 px of purple tint; spec §5.1 says purple never on backgrounds larger than a badge. Design §6 calls it badge-sized, which it is not. Orchestrator: amend spec §5.1 or make the band `bg-canvas`.
- For Tasks 3: catalog.tsx:142 and my-sections.tsx:127 `role="search"` divs are unlabelled next to the header's "Catalog" landmark; give them an aria-label (e.g. "Filters").
## Requests
- to orchestrator: settle the card-band colour vs spec §5.1; fold design §0 #1, #2, #4, #6, #17 deviations into the spec text.
## Blockers
