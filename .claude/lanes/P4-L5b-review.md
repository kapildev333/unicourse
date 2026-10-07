# P4-L5b-review - review of P4-L5b web landing (c03ef00 vs f51507a)
Status: done
## Doing
Verdict: changes requested (1 defect, nits below).
## Found
Gates (apps/web, run by reviewer): lint clean; typecheck clean; test 24 files / 167 passed; build ok (369.46 kB js).

Defects:
1. apps/web/src/features/catalog/catalog.tsx:32 - the new sync resets `text` to `q` whenever `q` changes, including when the page itself just wrote it. The debounce writes `text.trim()`, so typing "intro " and pausing 300 ms sets q="intro" and then setText("intro"): the trailing space is deleted under the user, and the next keystroke gives "introt". Probe (scratch): change to "intro ", wait 450 ms, expect value "intro " -> received "intro". Passes on base f51507a, so this is a regression. Smallest fix: `if ((q ?? '') !== text.trim()) setText(q ?? '')` inside the `seenQ !== q` block; with it, all 10 catalog tests and probes pass. (Leftover limit: if the router commits q only after the user has typed more, those extra characters are still lost. A ref holding the last value written would close that gap; not needed now.)

Nits:
- staff-editor.tsx:152 - the `cancel.error` banner stays up after a later add/remove succeeds, so a stale "Already cancelled" alert sits beside a new success notice. Call `cancel.reset()` in the add/remove onSuccess handlers, or at the start of add/remove.
- staff-editor.tsx:152 - "Already cancelled" is shown inside ErrorBanner with the title "Something went wrong", so a harmless outcome reads as a failure. Use a different title or put it in the status line.
- staff-editor.tsx:81 - the success notice has no full stop, unlike the other notices ("... removed from the staff.").
- staff-editor.tsx:152 - a 403 shows the default "You don't have permission to do this." and not the server's message. This is how errorMessage works everywhere in the app (D23 client copy), so it is fine, but the brief's "envelope message" is not literally met.
- Keying on bare `not_found` (no field) is fine: the API sends no field, form-errors allows `<code>` keys, and a missing section shows the section page's own 404 on refetch.
- staff-editor.test.tsx:240 - the "TA" case renders u-ta, who is not in `staff`, so it actually tests a student. The real TA case passes (probe), but the test is mislabelled.
- After a successful cancel the button unmounts and focus falls back to body. The status line is announced (role=status). The existing Remove flow behaves the same way.
- queries.ts:6 - `enabled = true` default for one caller; harmless.

Verified ok: the canRemove passed into group() is per group (canManageInstructors / canManageTAs from sectionAccess), so it matches the API rule for every role on every row type. Request: DELETE /api/v1/universities/a/sections/s1/pending-staff/inv-1. Button disabled while pending; double-click sends 1 DELETE (probe). 404 shows "Already cancelled" in role=alert, refetches, and the row goes. Accessible name "Cancel invitation for <email>", focusable. Catalog: an outside clear sends no request with the stale q (probe, 700 ms). Debounce with 100 ms gaps sends 1 request. The setState-during-render is React's documented "store info from previous renders" pattern: one extra render, no loop, no lint warning. Section dialog: edit mode makes 0 /courses requests; create mode still fetches.
Mutations: canRemove forced true -> 2 tests fail; sync removed (stale q written back) -> worker test + probe fail; enabled=false -> 4 create tests fail; enabled dropped -> edit test fails.
Scope: only the 7 web files listed + own lane log; no new dependency, no toast.
## Requests
- to orchestrator: defect 1 is a one-line fix for P4-L5b; add a catalog test "trailing space survives the write-back".
## Blockers
None.
