# P8-L3c - grading, states, dark mode and phone width comparison (docs/design/P8-L3c.md)
Status: done
## Doing
Done; docs/design/P8-L3c.md written (306 lines), not committed (orchestrator commits).
## Found
- For P8-L3d: 11 of 37 light-390 captures are wider than 390 (21/23 755, 43/44 693, 45-47 555, 52 561, 53 679, 60-63 550); measured with sips. Overlaps P8-L3a's schedule finding and P8-L3b's tables; suggest one cross-cutting "no page overflow" item with a scrollWidth check in the P8-L1 capture run.
- Gradebook at 390: sticky Student column (email names) fills the table, scores unreachable; same in Finalize dialog. Fix is a width cap on the sticky th.
- Gradebook hides missing scores: lucas.moreau 81.75% B- running, 32.70% F at finalize.
- Section detail add-staff row does not wrap at 390 (likely the 555 px overflow source); P8-L3b owns that screen.
- Copy changes that may hit e2e selectors: Finalize "Computed: {letter}" -> "Keep {letter}", "Running %"/"Running letter" headers, ErrorBanner default title.
- Skipped: Dribbble images, dark-390, 10/11/13 empty states. No tool contrast measurement.
## Requests
## Blockers
