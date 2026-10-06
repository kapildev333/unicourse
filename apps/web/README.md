# web

UniCourse web app (React + Vite + TypeScript). Stack and internal layout are defined in the first spec.

## Run

Node 22, npm. From `apps/web`:

- `npm run dev`: Vite dev server; `/api` is proxied to `http://localhost:8080`.
- `npm run build`: typecheck, then production build.
- `npm run lint`, `npm run typecheck`, `npm test`.
- `npm run generate`: regenerate `src/api/types.ts` from `../api/openapi.json`.
