# P6-L1-fix2 - stop header wrapping with DM Sans
Status: done
## Doing
Gates green (lint, typecheck, 177 tests, build). Orchestrator re-measures in Chrome.
## Found
Arithmetic (inner width = viewport - 64; gaps 2 x 16 = 32; nav 448 and cluster are shrink-0; only the name block shrinks):
- lg, 1024 px (inner 960), email hidden: cluster = 368 - 168 (email) = ~200. Name block at its cap = 95 wordmark + 25 separator/gaps + 192 = ~312. 312 + 448 + 200 + 32 = 992, 32 over, so the name truncates to ~160 and the row is exactly 960. Measured-case name (block 231): 231 + 448 + 200 + 32 = 911, fits with 49 spare.
- xl, 1280 px (inner 1088), email shown, measured case: 231 + 448 + 368 + 32 = 1079 <= 1088, fits.
- xl worst (name capped, email capped at 224, ~56 wider): cluster ~424; 312 + 448 + 424 + 32 = 1216, 128 over; name truncates to ~64 px. Still one row.
- Two or more memberships (coordinator decision): select is w-40 min-w-0 xl:w-56, so at lg cluster ~372, name block shrinks to the wordmark (~107): 107 + 448 + 372 + 32 = 959 <= 960.
## Requests
## Blockers
