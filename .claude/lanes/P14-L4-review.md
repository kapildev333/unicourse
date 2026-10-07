# P14-L4-review - review of web sections, courses, e2e backlog
Status: done
## Doing
Finished: approve, no blocking findings.
## Found
- Participants route, tab order (Spec 6 §8), single SectionBand, additive action/bandAction all as briefed; P14-L3's SectionFrame use (slug, sectionId, children) is compatible.
- Tab visibility = canRead (staff, managers, students), matches §8. Plain member: no strip, typed URL shows staff read-only (server returns Section to any member, §4).
- Observation for orchestrator: Spec 6 §8 Home row still lists "staff with role badges"; after the split a plain member sees only instructor names in the band (TAs no longer visible to them). Brief scoped Home to band, description, meetings, so not a finding.
- e2e selector changes are navigation only (Participants link clicks / `/participants` gotos) plus the plain-member strip-absent assertion; no copy changes.
- Nits: capture numbering 48 / 48b run before 46/47; route file uses double quotes and semicolons unlike its siblings.
## Gates (apps/web, worktree)
lint clean; typecheck clean; test 56 files / 410 passed; build ok; routeTree.gen.ts unchanged after build.
## Requests
none
## Blockers
none
