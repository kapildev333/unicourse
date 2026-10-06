# P3-L1a - serve the built web app from the API binary
Status: done
## Doing
Finished; awaiting review.
## Found
- Env var `UNICOURSE_WEB_DIR` (-> `Config.WebDir`); empty = API only. Start the binary with `UNICOURSE_WEB_DIR=<repo>/apps/web/dist`; with `UNICOURSE_WEB_ORIGIN` empty, auth redirects land on `UNICOURSE_BASE_URL` (authDeps unchanged).
- Fallback rules (`httpx.Static`): GET/HEAD only (else JSON 404); existing regular file served as is; `/api` and `/api/*` -> JSON 404; other path with a file extension -> JSON 404; everything else -> `index.html` 200. `index.html` no-cache, `/assets/*` immutable 1y. `newHandler(logger, ready, api, web)` has a new 4th param (nil = old behaviour).
- Root opened with `os.OpenRoot`, so `..` and escaping symlinks are not served.
## Requests
## Blockers
