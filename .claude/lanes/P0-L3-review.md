# P0-L3-review - review of web skeleton and API client
Status: done
## Doing
Review finished; verdict: changes needed (two small fixes in client.ts / dialog.tsx).
## Found
- Gates green from clean `npm ci` on node 22.11: lint, typecheck, 6 tests, build; git status clean afterwards.
- openapi-fetch returns non-JSON error bodies as a string (no throw); unwrap then yields code 'internal' and message '' (statusText is empty on Response/HTTP2).
- Dialog uses a fixed id "dialog-title"; two mounted dialogs collide.
- For later lanes: `next` from /sign-in must be validated as a same-origin path before navigating (starts with '/' and not '//' or '/\').
## Requests
- to orchestrator: consider a CI drift check on src/routeTree.gen.ts (vitest regenerates it, so CI never sees stale commits).
## Blockers
