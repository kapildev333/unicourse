# P12-L1-review - review of the Postman collection lane
Status: done
## Doing
Finished. Verdict: approve (nits only).
## Found
- Gates rerun in the worktree: lint 0, typecheck 0, `npm test -- --run scripts` 7/7 passed; `npm run postman` + `git diff --exit-code` on the collection clean.
- Collection: 0 `id`/`_postman_id`/`auth` keys (grep hits are path segments and query `key` values), 73 requests all with `response: []`, `variable: []`, path vars `{{slug}}`/`{{token}}`/`""`, 34 query params all disabled and empty, magic-link body `{{email}}`, consume item `followRedirects: false` + CAPTURE test, collection prerequest SEND. CAPTURE/SEND verbatim with design §4; environment equals design §5; README names exist; CI lines only in web job after the types check.
- Exit paths: no args 2; unreadable/invalid spec 1; missing consume route 1 with "not found in the spec".
- Lock: 81 added packages, all dev; no existing package version changed (ansi-regex lost `peer: true`, now a real transitive dep). `openapi-to-postmanv2` pinned `6.4.0`, only package.json addition.
- Nits: (1) postman.mjs:63 CLI guard compares unresolved argv[1] to the realpath'd module URL; run via a symlinked path it exits 0 without writing (reproduced). Fix: `realpathSync(process.argv[1])`. Not hit in CI. (2) postman.test.mjs: no assertion that other path vars become `""`, nor that a missing request-link route rejects; Spec 8 §5 does not list them.
- Newman flow not rerun (brief: no API, no Newman); orchestrator repeats it.
## Requests
- none
## Blockers
none
