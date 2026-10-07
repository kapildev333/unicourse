# web

Part of [UniCourse](../../README.md).

UniCourse web app (React + Vite + TypeScript). Stack and internal layout are defined in the first spec.

## Run

Node 22.12 or newer, npm. From `apps/web`:

- `npm run dev`: Vite dev server; `/api` is proxied to `http://localhost:8080`.
- `npm run build`: typecheck, then production build.
- `npm run lint`, `npm run typecheck`, `npm test -- --run` (plain `npm test` is watch mode).
- `npm run screens`: capture light and dark screenshots of the main pages into `e2e/.tmp/screens/` (needs the e2e setup below).
- `npm run generate`: regenerate `src/api/types.ts` from `../api/openapi.json`.

## End-to-end test

`npm run e2e` runs the Playwright staffing flow (`e2e/staffing.spec.ts`) in Chromium against the real API binary and a real Postgres. One-time: `npx playwright install chromium`, and the `unicourse_e2e` database must exist (`apps/api/docker/initdb.sql` creates it on a fresh setup).

Global setup builds the web app and both Go binaries into `e2e/.tmp/`, migrates, and starts the API on `127.0.0.1:18180` serving `dist/` (`UNICOURSE_WEB_DIR`). Dev mail goes to `e2e/.tmp/api.log`, which the tests read for magic links. Slugs, domains and emails are unique per run, so runs never collide and the database is never reset. Override the database with `UNICOURSE_E2E_DATABASE_URL` and `UNICOURSE_E2E_OWNER_DATABASE_URL` together (the CLI migrates through the owner URL, the API reads the other).
