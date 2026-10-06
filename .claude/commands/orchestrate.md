---
description: Start or resume an orchestration session over PLAN.md
---

You are the orchestrator for this work. Source documents: $ARGUMENTS

1. Read PLAN.md at the repo root if it exists; otherwise read the source documents and propose PLAN.md: numbered decisions that need my call, method rules, and a first phase cut into lanes with disjoint directory scopes and a model per lane. Stop there and discuss it with me.
2. Once I agree, for each lane: `git worktree add -b lane/<LANE-ID>-<slug> .worktrees/<LANE-ID> main`, then launch the lane as a sub-agent with the worktree path and branch in its brief. Run lanes in parallel where scopes are disjoint: designer when a lane needs design judgement, then worker to build it, then reviewer on the landing. Every brief states its LANE-ID, worktree path, branch, scope paths, and the spec section it implements.
3. Read every `.claude/lanes/*.md` whenever a lane reports or you check in; relay urgent cross-lane requests to the target lane (SendMessage if it is still running, otherwise in its next brief).
4. Verify every landing yourself before opening a PR: in the lane's worktree, re-run the suites, then exercise the real thing (real database, real HTTP, real browser). Never trust a lane's own report or the reviewer's verdict as verification.
5. Land through GitHub, never by direct push to main: `git push -u origin lane/<...>`, `gh pr create --fill --body-file` using the PR template (lane, spec section, gates pasted, reviewer verdict, deferred), wait for `ci` to pass, `gh pr merge --squash --delete-branch`, then `git worktree remove .worktrees/<LANE-ID>`. Put the PR URL in the lane's status cell in PLAN.md.
6. Keep PLAN.md current on main: statuses with PR links and evidence, notes per lane, backlog, journal. PLAN.md changes go in their own small PRs (`chore/plan-<date>`), merged without waiting on other lanes.
7. Escalate to me only decisions that belong in the decisions table.

You do not write features yourself while a lane could.
