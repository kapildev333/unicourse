# P17-L7-review - review of P17-L7 (unicourse_platform role, migration 0009)
Status: done
## Doing
Review finished: changes needed (one security/doc finding, see Found 1).
## Found
1. The tenant boundary is the client-set GUC `app.university_id`, so a holder of the platform credential can set it to ANY existing university and (a) read that tenant's memberships joined to users (the roster) and pending invitations, (b) insert an `admin` invitation with a token hash it chose = become admin of any university. Shown on unicourse_e2e_17 in a rolled-back tx. README row "cannot ... read tenant data outside the university being created" is false. Fix: correct the README (minimum) or restrictive policies `to unicourse_platform` scoping to universities created in the current transaction (design call for the orchestrator/owner).
2. users readable in full (all columns); column grant `select (id, email)` narrows it. Optional.
3. Gates green: gofmt, vet, cmd/unicourse-admin + internal/pg tests (no SKIP). Real check on unicourse_e2e_17: migrate x2 ok, create-university as platform ok, migrate as platform 42501, merges with L2 and L8 branches are clean.
## Requests
- to orchestrator: decide Found 1 (doc fix vs restrictive policies); D133 owner note should state the platform credential can still administer any tenant unless hardened.
## Blockers
