# P9-L1c - shared component follow-ups
Status: done
## Doing
Finished; waiting for orchestrator.
## Found
- All 8 items done. ButtonLink, ListCount, ghost-danger muted + ghost-neutral, Table `relative`, inputSizedClass, personName in Change role dialog/message.
- Item 6: gradebook capName now `sm:w-px sm:whitespace-nowrap`; the Student column is as narrow as its email line at 1440. The remaining space sits in the score columns (table is width 100%); not fixable without a fixed layout.
- B9: tnum absent in DM Sans (GSUB: calt ccmp dnom frac liga locl numr), nothing changed, D56 kept.
- Tests that located the navigate buttons by role=button now use role=link (section-list "Set up terms", route no-access "Go to ..."); the e2e suite does not locate them.
- Orchestrator: add ButtonLink, ListCount, ghost-neutral, inputSizedClass to CLAUDE.md components list. The members row button aria-label still names the email.
## Requests
## Blockers
