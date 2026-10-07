# P8-L1-review - review of screens.ts capture script
Status: done
## Doing
Nothing; verdict approve with findings, reported to orchestrator.
## Found
- lint and typecheck green in apps/web (e2e/ included). No new dependencies; CI runs `npm run e2e` = `--project chromium`, whose default testMatch excludes screens.ts.
- Dark via emulated prefers-color-scheme hits `html:not([data-theme])` tokens, identical to `[data-theme=dark]` (theme.test.ts enforces). Header switch shows "System" in every shot; harmless for the benchmark.
- Gaps: no instructor schedule capture (brief asks list+schedule for student and instructor); skeleton capture is timer-raced (8 s delay vs 4 full-page shots); 04 file names "other-university" in the role slot.
- Product gap confirmed: enrollment CSV accepts `displayName` but names are never shown (people render as emails).
## Requests
## Blockers
