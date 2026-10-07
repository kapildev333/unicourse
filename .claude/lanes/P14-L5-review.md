# P14-L5-review - review of web dependency bumps
Status: done
## Doing
Finished. Verdict: changes needed (one fix: the stated Node floor).
## Found
- The Node floor is wrong. jsdom 30.1.2, @asamuzakjp/css-color, @asamuzakjp/dom-selector and w3c-xmlserializer need `^22.22.2 || ^24.15.0 || >=26`, and undici 8.11.2 needs `>=22.19`. So `engines: >=22.12` (apps/web/package.json:6) and "Node 22.12 or newer" (README.md:18, apps/web/README.md:9) understate it. Fix: `>=22.22.2` and "Node 22.22.2 or newer". CI `node-version: 22` resolves to the newest 22.x, so CI is fine.
- Tests: main also has 424 tests in 60 files (432 in the brief was a miscount), so the count did not change.
- Build: main gives a 489.47 kB JS chunk (145.90 kB gzip) with no warning. The worktree gives 546.32 kB (160.54 kB gzip) and the >500 kB warning. The growth is React 19's react-dom. It is not a gate failure, but the warning is new.
- For the orchestrator: when D1 is amended, PLAN.md:172 (React 18.3, jsdom ^26, Node 22.11) is out of date too.
## Requests
- to P14-L5: raise the Node floor to 22.22.2 in package.json engines (and the lock root) and in both READMEs.
## Blockers
