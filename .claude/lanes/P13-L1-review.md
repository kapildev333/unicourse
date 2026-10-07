# P13-L1-review - review of the releases process
Status: done
## Doing
Finished. Verdict: approve, with one recommended doc fix (see Found 1).
## Found
1. docs/releases.md:8 and CLAUDE.md:60 define `chore` as `PLAN.md:`, `ci:` (and flaky-test in the doc) PRs, but spec/plan/design PRs (#4, #11, #36, #52-#55, #58) and #110 (`content:` flaky fix) were labelled `chore` too. A future `Spec 9: ...` PR would land in the notes. Fix: in both lines, name spec, plan and design-doc PRs as `chore` too, e.g. "Label `PLAN.md:`, `ci:`, spec/plan/design-doc and flaky-test PRs `chore`".
2. Minor: `gh release create --help` (gh 2.101.0) only says "Additional release notes can be prepended to automatically generated notes by using the `--notes` flag"; it does not name `--notes-file`. gh feeds both flags into the same body, so docs/releases.md:28 holds, but the P13-L1 log's "help confirms" for `--notes-file` overstates the help text.
3. Minor (no fix needed): `--notes-start-tag` on the patch recipe is redundant (GitHub defaults to the previous release) but harmless.
- release.yml parses; it is the documented `changelog.exclude.labels/authors` + `categories` shape. The API reports Dependabot's login as `dependabot[bot]` (pulls/1 .user.login), which the file lists, so the exclusion holds.
- Merged PRs with no `chore` label and not by Dependabot: 63, all `P<n>-L<m>` lane PRs. Every one belongs in the notes. Borderline but fine: P8-L1/P8-L2/P8-L3/P9-L9 (benchmark and audit work, not user features). No `chore`-labelled PR is a feature.
- Preview (main's config, so nothing excluded yet): 114 bullets, including Dependabot and PLAN.md PRs, as expected.
- README links resolve (`PLAN.md#backlog` -> `## Backlog`; repo is kapildev333/unicourse). CLAUDE.md diff is the two bullets only.
## Requests
## Blockers
