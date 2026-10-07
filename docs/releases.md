# Releases

## Rule

- One GitHub release per completed phase, tagged `v0.<phase>.<patch>` while nothing deploys (D112). Minor = the phase that completed; patch = a fix-only batch after it. `v1.0.0` is the first real university.
- The orchestrator cuts it at phase close, after the PLAN.md status PR is merged. A lane never tags or releases (D116).
- No artifacts yet (D115); a bundle workflow comes with the first deployment target.
- Notes = a Highlights section written by the orchestrator + the generated PR list. `.github/release.yml` drops PRs labelled `chore` and Dependabot. Label `PLAN.md:`, `ci:`, spec/plan/design-doc and flaky-test PRs `chore` when opening them.

## Highlights template

Three to six bullets in user terms, each naming what someone can now do and linking the spec:

```markdown
## Highlights

- Students see their sections, weekly schedule and clashes ([Spec 2](https://github.com/kapildev333/unicourse/blob/main/docs/specs/<file>.md)).
- Staff publish announcements and weekly materials on a section ([Spec 6](https://github.com/kapildev333/unicourse/blob/main/docs/specs/<file>.md)).
```

Links in release notes must be absolute GitHub URLs; relative paths do not resolve on the Releases page.

Source: the spec's section 1 and the phase outcome in PLAN.md.

## Phase close

`--generate-notes` prepends the notes you pass (`--notes` or `--notes-file`) to the generated list; verified on the first release.

```
git checkout main && git pull --ff-only
gh release create v0.<phase>.0 --target main --title "v0.<phase>.0 - <phase name>" --generate-notes --notes-file <highlights.md>
```

Patch release: the same command with `v0.<phase>.<n>` (n >= 1), highlights listing the fixes; add `--notes-start-tag v0.<phase>.<n-1>` so the list covers only the new PRs.

Preview the generated list without creating anything:

```
gh api repos/{owner}/{repo}/releases/generate-notes -f tag_name=v0.<phase>.0 -f target_commitish=main --jq .body
```
