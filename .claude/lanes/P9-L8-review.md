# P9-L8-review - review of P9-L8 (display names API)
Status: done
## Doing
Review complete: approve (two low test-coverage findings, no blocker).
## Found
Verdict: approve. Build matches docs/design/P9-L8.md; scope is exactly design §7 + design/lane log + orchestrator e2e line; routes.go untouched; types.ts byte-identical to a fresh openapi-typescript run.
Checked and holding: fill-only-if-empty on import add/invite (users and invitations UPDATEs guarded `= ''`), sign-in (`update ... and display_name = ''` after the upsert has row-locked the user, so a concurrent OIDC name is re-read under READ COMMITTED and not clobbered); exists rows untouched; 101-rune name is an error row, other rows still apply; migration default/check correct, no grant needed (0004 table-level grant + default privileges, no column grants, no unique open-invitation constraint); PATCH /me trims, 1-100 runes, NUL, envelope field displayName, Authenticate + SameSite=Lax like every other mutation (Sec-Fetch-Site is only on magic-link consume), body has no id, `for update` + no-op on same name, one membership.update per active membership in its own university, returns writeMe body. Import audit is written with s.UniversityID and the CTE's memberships join is confined by tenant_isolation, so no B row and no cross-university read.
1. (low, tests) apps/api/internal/course/import_test.go: the applier branch that fills an existing open invitation whose name is '' (import.go `case ref.InvitationID != ""` appends to invFillIDs) is never exercised with an empty name; deleting those 3 lines passes every test. Fix: in TestRosterImportNamesFillOnlyIfEmpty, import `new4@a.edu,` (no name) into SectionA, then `new4@a.edu,Late` into sec2, assert invName == "Late".
2. (low, tests) apps/api/internal/auth/invite_test.go TestSignInInvitationNameOldestWithName: A('') then B('Second') pins "skip unnamed" but not "oldest": an implementation taking the newest named invitation passes. The worker log's "still pins oldest-with-name" is inaccurate. Fix: set A's name to 'First' in a second case (or add a third invitation; nothing in the schema forbids it) and assert 'First'.
Gates (run by me): gofmt -l empty; go vet clean; go test ./... -count=1 -race: 14/14 ok, 0 SKIP, new tests PASS; web lint/typecheck ok, vitest 246/246, build ok; e2e 2 passed, 0 bind collisions.
## Requests
## Blockers
