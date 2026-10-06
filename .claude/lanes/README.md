# Lane logs

One file per lane, named by its LANE-ID from the brief: `lanes/<LANE-ID>.md`. The lane that owns the file is the only one that writes it.

```markdown
# <LANE-ID> - <deliverable>
Status: started | blocked | done
## Doing
<current step, one line, keep it current>
## Found
<things other lanes or the orchestrator need to know: changed interfaces, gotchas>
## Requests
- to <LANE-ID>: <what you need from them>
## Blockers
```

- Read every other lane's file when you start and before you finish; answer requests addressed to you in your own `Found` section.
- The orchestrator reads all lane files, relays urgent requests, and folds anything durable into PLAN.md. PLAN.md is still orchestrator-only.
