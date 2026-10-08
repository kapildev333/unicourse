# P17-L9b-review - review of XHR upload progress, router pending and preload
Status: done
## Doing
Review complete: approve, two low findings.
## Found
- Gates: lint clean, typecheck clean, vitest 445/445, build index chunk 294.67 kB (gzip 93.49 kB).
- F1 (low): a non-JSON 413 from a proxy becomes code 'internal', so uploadReason says "Something went wrong on our side. Try again in a moment." for a file that will always be refused. Fix: in uploadReason, `if (e.status === 413) return 'It is larger than the server accepts.'` plus a test.
- F2 (nit): lib/upload.ts onload builds `new Response(null, { status })`, which throws RangeError for a status outside 200-599 (non-standard proxy codes); the throw escapes onload and the promise never settles, leaving the week busy. Fix: try/catch around the body of onload, reject on throw.
- accent-accent compiles to accent-color: var(--color-accent), which Chromium and Firefox apply to <progress>; color-scheme: dark is set for the dark theme so the track follows it.
- Router: autoCodeSplitting is on and no route has a loader/beforeLoad, so 'intent' only prefetches chunks; bare Loading (default variant) shows after the default 1 s pendingMs.
## Requests
none
## Blockers
none
