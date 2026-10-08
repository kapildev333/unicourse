# P17-L9a-review - review of route code splitting
Status: done
## Doing
Verdict: approve. No blocking findings.
## Found
- Plugin 1.168.42 default split groups: component, errorComponent, notFoundComponent (loader not split). No route file has loader/beforeLoad/pending/error components, and every route file exports only `Route`; nothing to move or duplicate.
- route.test.tsx under src/routes: generator warning is pre-existing and harmless; the test imports the real Route whose component is now lazy under vitest too, but it awaits with findByRole, so it should still pass (not run: Node 22.11).
- Build deterministic (two builds, identical asset hashes), 69 JS assets, index 292.71 kB / 92.97 kB gzip, routeTree.gen.ts unchanged.
- Follow-up (outside lane): no defaultPendingComponent/defaultPreload. On a slow network, a click keeps the old page up to 1 s (defaultPendingMs) then the outlet renders nothing until the chunk lands; a cold load shows a blank page until the index chunk and the route chunk load. Suggest `defaultPendingComponent` = Spec 3 `Loading` and `defaultPreload: 'intent'` in src/main.tsx.
## Requests
- to orchestrator: consider the pending/preload follow-up above as a backlog item.
## Blockers
