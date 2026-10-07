# P9-L1c-review - review of P9-L1c (shared component follow-ups)
Status: done
## Doing
Finished: approve, one minor test finding.
## Found
- Verdict: approve.
- Minor: apps/web/src/routes/$slug/route.test.tsx:70 weakened from "no `/^Go to /` button" to "no `/^Go to Bay/` link"; a stray "Go to Harbor College" primary link with several memberships would now pass. Fix: `queryByRole('link', { name: /^Go to / })`.
- ghost-danger rest (ink-muted): light 5.08 on surface, 4.81 on canvas (row hover); dark 5.63 / 7.07. Hover danger on danger-soft: light 5.95, dark 7.36. ghost-neutral hover ink on canvas 15.97. All from tokens.
- ButtonLink keeps TanStack route typing (probe: bad `to` and bad `params` key both TS errors); renders an anchor with href; Link's aria-current/active handling untouched.
- Built CSS: wrapper `.relative` and th `.sticky` are separate elements, no conflict; audit When (audit-log.tsx:163,176) and gradebook Student sticky unchanged. No `!` widths left in features/grades; no importers of features/admin/list-count.
- e2e locators: none for "Set up terms", "Showing", "Change role"; "Remove ... as Instructor" unaffected.
- e2e first run here failed on a port collision (api.log: bind: address already in use); re-run 2 passed. Consistent with the worker's "unrelated" claim; the worker's own api.log was overwritten (no bind line now), so not independently provable.
- Out of scope: gradebook 1440, last body row's bottom rule only under the sticky Student cell (Tomasz row); not caused by this diff as far as the code shows.
## Requests
## Blockers
