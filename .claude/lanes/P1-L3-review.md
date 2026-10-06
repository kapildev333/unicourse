# P1-L3-review - review of unicourse-admin CLI (HEAD 5c5448f)
Status: done
## Doing
Review finished; verdict changes needed.
## Found
- Atomicity correct by code (one WithPlatformTx + SetUniversity), but no test pins it: mutation moving the invitation into a second tx passes all tests.
- Accepted without error (probes, exit 0): --timezone Local; --locale "" / "not a locale!!"; --slug invite / sign-in / api; --domains "a.edu,A.edu, a.edu" stored as a.edu,a.edu,a.edu; stray positional arg ignored; 100k-char name.
- --name with NUL -> exit 1 (DB error), should be 2.
- Output test uses Contains; mutation printing the URL first passes.
- Out of scope: internal/auth TestSignInExpiredInvitation failed once in the full run, passed -count=3 alone (expiry now()-1s set by Go clock vs Postgres now(): host/container clock skew flake). Not caused by this commit.
## Requests
None.
## Blockers
None.
