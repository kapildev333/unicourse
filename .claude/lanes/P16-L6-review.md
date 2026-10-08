# P16-L6-review - review of grades.spec.ts (P16-L6)
Status: done
## Doing
Review finished: changes needed (3 assertion findings, 1 selector finding, nits).
## Found
- Gates at port 18116: lint clean, typecheck clean, `npm run e2e -- grades.spec.ts` 12/12 (14.8 s). `npm test -- --run` skipped (Node 22.11 below floor 22.22.2, known).
- E-GRADES-12 (grades.spec.ts:274): scrollIntoViewIfNeeded also scrolls the Table's overflow-x-auto wrapper sideways, so the case passes even if S5-52's narrow sticky Student column regresses. Check the cell's boundingBox right edge <= 390 instead.
- E-GRADES-08 (grades.spec.ts:241-243): toHaveCount(0) on the My sections card with no positive wait; passes against a page still loading. Assert the CS101-001 card visible first. Line 242 ternary is dead code.
- E-GRADES-10 (grades.spec.ts:307): /\bB\b/ on the whole row also matches "B+"/"B-" and the running letter; passes only by luck of the data. Use row.getByText('B', { exact: true }).
- grades.spec.ts:107,116: file input has label "CSV file" (FormField); use dlg.getByLabel('CSV file') as content.spec.ts:127 does.
- Nits: expects without §0.7 messages (52, 80, 129, 217, 218, 227, 302); unnamed dialogs at 56, 63.
## Requests
- to P16-L6: fix the four findings above (orchestrator relays).
## Blockers
none
