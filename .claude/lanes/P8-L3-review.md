# P8-L3-review - read-only review of Spec 5 and its Phase 9 plan
Status: done
## Doing
Done: verdict request changes (spec §5.7 selector list incomplete and partly wrong; three A items break e2e assertions not in §5.7).
## Found
- e2e breaks not in §5.7: staffing.spec.ts:200 region "Sunday" (S5-25), :205-206 instructor in schedule block (S5-26), :247 "New section" with no terms (S5-15).
- §5.7 lists staffing.spec.ts:81 for S5-07, but that line is the admin invitation dialog's Email field (invitations.tsx:77), not sign-in.
- Unit selectors missing from §5.7: catalog.test.tsx:106,116,127 (S5-01), sign-in-page.test.tsx:9,17 (S5-07), my-grades.test.tsx:134-135, gradebook.test.tsx:317, section-list.test.tsx:137,190, course-list.test.tsx:80 (S5-70).
- schedule.tsx:124 em dash (S5-70 "schedule blocks") is in neither L2 nor L6 scope.
- S5-48/S5-49 sort/filter invitations client-side, but the list is keyset-paged by created_at (limit 50): needs an API param or "loaded rows only".
- C1 import rule misses invitees (D21: roster rows without an account become invitations, no users row).
- Verified true: section dialog uses size="lg" (max-w-2xl), Location cut at 1440; no API writes display_name except auth.SignIn; screens.ts snap can carry expect.soft; all cited captures and sampled references exist.
## Requests
## Blockers
