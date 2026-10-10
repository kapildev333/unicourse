# P21-L5-review - review of P21-L5 (web Upcoming panel, Calendar feed dialog, campus zone)
Status: done
## Doing
Review finished. Verdict: changes needed (small fixes).
## Found
- R1 (e2e, outside L5 scope): e2e/catalog.spec.ts:152 works out "today" in the runner's zone, but schedule.tsx now marks today in the campus zone (America/New_York for seed-demo). It fails on a UTC runner from 00:00 to 04:00/05:00 UTC and on an IST machine until about 09:30. Fix: add `timeZone: 'America/New_York'` to that toLocaleDateString call.
- R2: upcoming.tsx:16-21 uses staleTime 60s and nothing invalidates [slug,'upcoming'] after a quiz or assignment is submitted or created. A submitted quiz still shows if you go back within 60 s by in-app navigation. Fix: `refetchOnMount: 'always'` (in scope), or invalidate on submit.
- R3: format.test.ts:30-33 checks the date-only rule with Pacific/Auckland, which is east of UTC. A mutation that formats `new Date('YYYY-MM-DD')` in the campus zone still passes; I verified this. Use a zone west of UTC (America/Los_Angeles).
- R4: calendar-feed-dialog.test.tsx:42 fails under TZ=Pacific/Auckland (no campus set; 12:00Z is Oct 10 there). Set the campus to UTC in that test.
- R5 (minor): calendar-feed-dialog.tsx:49. A failed "Turn off" leaves revoke.error in place, so a later successful create shows a stale error banner over the URL. Also, the clipboard fallback copy goes through StatusMessage (success styling).
- e2e local() set to America/New_York is right: both files seed through seed-demo (demo.json timezone America/New_York). L6's upcoming.spec must pick the zone per university.
## Requests
- to orchestrator: R1 touches e2e/catalog.spec.ts (not in L5's scope). Give it to L5 as a fix or fold it into L6.
## Blockers
