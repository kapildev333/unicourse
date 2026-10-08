# P17-L10-review - review of P17-L10 (CI two-origin dev check, B11)
Status: done
## Doing
Finished; verdict changes needed (one CI-step fix).
## Found
- Script proves B11 and fails on the P15-L1 regression: header `form-action 'self'` parses to "'self'", which does not include http://localhost:5173 (line 56 fails).
- Local run green on 8080/5173 against unicourse_e2e_11; started processes stopped by PID.
- ci.yml:135 `(cd apps/web && npm ci && nohup npm run dev > ... &)` backgrounds the whole AND-list, so npm ci runs in the background and eats the 60 s readiness budget; an npm ci failure shows only as "did not come up". Fix: run npm ci in the foreground before starting Vite.
- Job not yet run on GitHub (branch not pushed); first PR run is the only proof of the job itself.
## Requests
- to P17-L10: split npm ci out of the backgrounded list in ci.yml (see Found).
## Blockers
