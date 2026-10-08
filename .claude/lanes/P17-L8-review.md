# P17-L8-review - review of P17-L8 (seed --fixture, fixture schema, Makefile, screens via seed-demo)
Status: done
## Doing
Finished: verdict changes needed (Makefile env leaks into `make e2e`; fresh-clone `make dev` lacks npm install; schema test misses B8's "demo.json has the required keys" direction).
## Found
- `make e2e` fails: the Makefile's global `export UNICOURSE_WEB_ORIGIN` passes through e2e/env.ts (`...process.env`, not overridden), so the e2e API redirects sign-ins to :5173. Reproduced: demo.spec E-DEMO-01 fails with UNICOURSE_WEB_ORIGIN set, passes without.
- `make dev` on a fresh clone runs `npm run dev` without `npm install`; README sends the reader to step 6, past step 5's install.
- Compose project: Makefile uses `-p unicourse`, README manual step uses `docker compose up -d` in apps/api (project `api`): the two paths start two Postgres containers on 5432.
- `seed-demo --fixture`: `../x` and absolute paths rejected ("must be a bare file name"); material `"."` now passes validation and fails mid-run (partial seed); missing-file message still says "no embedded file".
- Schema compiles under ajv 8 draft 2020-12 strict; demo.json validates.
- Gates: gofmt clean, vet ok, seed and cmd tests pass (no SKIP), lint and typecheck ok, screens 1 passed (161 captures), `make seed` against unicourse_e2e_17 ok.
## Requests
- to orchestrator: compose top-level `name: unicourse` (apps/api/docker-compose.yml, outside L8 scope) would make the README manual path and the Makefile agree.
## Blockers
