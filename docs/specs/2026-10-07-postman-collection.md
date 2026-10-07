# Spec 8: Postman collection and environment from the OpenAPI spec

Written 2026-10-07 by the orchestrator from the owner's ask ("write OpenAPI specs, Postman collection and env for testing", test-driven and spec-driven) and the P12-L1 design, which the owner steered directly. Bounded change on an existing contract; decisions in §8.

## 1. Intent

Anyone with Postman can exercise the whole API against the local demo in minutes: import two files, sign in as a demo persona, call any route. The collection never drifts from the API, because it is generated from the contract and CI fails when it is stale.

## 2. What exists

- `apps/api/openapi.json` is the OpenAPI 3.1 description of all 49 routes, and `internal/app/contract_test.go` fails CI when a route is missing from it or vice versa. It is the single source of the API description; this spec adds no second one and changes nothing under `apps/api`.
- The web CI job already regenerates `src/api/types.ts` from the spec and fails on a diff. The collection follows the same pattern.

## 3. Deliverables

- `docs/postman/unicourse.postman_collection.json`: Postman Collection v2.1 generated from the spec by `openapi-to-postmanv2` (pinned dev dependency of `apps/web`), normalised to be byte-identical on every run (no random ids, no example responses, no spec-level auth, no collection variables), with a collection-level pre-request script that sends the session cookie and a test script on the magic-link consume request that captures it. Exact rules: design §4.
- `docs/postman/unicourse.postman_environment.json`: hand-written, "UniCourse local": `baseUrl`, `slug`, `email`, `token`, `session` (secret). Defaults match `seed-demo`. Exact content: design §5.
- `apps/web/scripts/postman.mjs` and `npm run postman`; CI regenerates and diffs the collection in the `web` job right after the types check.
- Root `README.md` gains the "Explore the API with Postman" section of design §7, after "Demo personas".

## 4. Sign-in

The server's cross-site check on magic-link consume looks at `Sec-Fetch-Site`, which Postman does not send, so the consume request works from Postman. The session cookie is `Secure` and `HttpOnly`; whether Postman's cookie jar replays it over plain `http://localhost` is not proven, so the collection captures the cookie value into the `session` environment variable and sends it as a `Cookie` header on every request (D110). The consume request does not follow its redirect.

## 5. Testing, test-driven

- `apps/web/scripts/postman.test.mjs` (Vitest, included in `npm test`): written first and failing before the script exists. Cases: with a small inline OpenAPI document containing the two magic-link routes, one `{slug}` route with a query parameter and a cookie security scheme, the output has no `id`, `_postman_id` or `auth` key at any depth, every `response` is `[]`, `variable` is `[]`, the `{slug}` path variable is `{{slug}}`, the query parameter is disabled with an empty value, the magic-link request body is the `{{email}}` JSON, the consume item has `followRedirects: false` and the capture test script, the collection has the pre-request send script, and running the normaliser twice gives identical JSON; with a document missing the consume route the run rejects with the "not found in the spec" error. The script exposes its conversion as an exported function so the test calls it without spawning a process.
- `npm run postman` twice on the real spec, `git diff --exit-code` clean after the second.
- Real run with Newman (`npx --yes newman@6`, verification only, not a dependency) against the local API with `seed-demo` loaded: request a magic link for Elena, take the token from the API log, run the consume request (its test passes, 302 with the cookie) and then "My sections" with the exported environment: 200 with three items. Transcript in the lane log; the orchestrator repeats it.
- The owner imports both files into the Postman app once and repeats §7 of the design by hand; recorded in PLAN.md when done.

## 6. Out of scope

- OpenSpec (the P12-L2 design exists on the design branch; not asked for in this phase, owner's call later).
- Request chaining (storing created ids), staging environments, file bodies for multipart requests: design §10 ceilings.
- Any change to `apps/api` or the spec.

## 7. Internal layout

```
apps/web/scripts/postman.mjs        convert + normalise; export `buildCollection(specText) -> Promise<object>`; CLI main when run directly
apps/web/scripts/postman.test.mjs   Vitest
apps/web/package.json               script `postman`, dev dependency `openapi-to-postmanv2` pinned
docs/postman/                       collection (generated) and environment (hand-written)
.github/workflows/ci.yml            two lines in the web job
README.md                           one section
docs/design/P12-L1.md               the design, as written by the designer
```

## 8. Decisions needing the owner's call

- D109: the Postman collection is generated from `apps/api/openapi.json` by `openapi-to-postmanv2` as a pinned dev dependency of `apps/web`, committed under `docs/postman/` with a hand-written environment, and CI regenerates it and fails on a difference. The spec stays the single source; `apps/api` is untouched.
- D110: the collection authenticates by capturing `unicourse_session` from the consume response into an environment variable and sending it as a `Cookie` header from a collection pre-request script, not through Postman's cookie jar.
- D111: a Vitest unit test covers the generator's normaliser (the owner asked for test-driven work; the design had proposed the CI diff as the only test).
