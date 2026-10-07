# P11-L2-review - review of the root README and screenshots (lane/P11-L2-root-readme)
Status: done
## Doing
Review complete. Verdict: changes needed (two one-line README edits, no blocker in commands or links).
## Found
- Quick start re-run by the reviewer on a scratch database `unicourse_p11l2review` in the running unicourse-postgres-1 (no volume touched, database dropped afterwards): migrate ok; seed-demo exit 0, output byte-matches the README persona table and design P11-L1 §7.3; second run exits 2 with the --reset hint; API /readyz ok; magic link for amara and ingrid -> 202, `"msg":"mail"` log line with an 8080 link; GET = confirmation page ("Continue signing in"), POST = 302 to http://localhost:5173/northfield/me with session cookie. Web dev server not re-run (npm install skipped).
- `seed-demo -h` flags (--slug, --reset) and env (OWNER URL, BASE_URL, FILES_DIR) match the README; WEB_ORIGIN in the inline block is needed (redirect target and printed sign-in URL).
- Links: 29 relative links in README.md and apps/web/README.md all resolve; `#backlog` matches `## Backlog` in PLAN.md. Every docs/specs and docs/plans file listed (docs/specs/README.md is the folder note, rightly omitted); topics match each §1.
- Screens: 8 files, all 1440 wide, 768K total. my-sections-dark = Amara's schedule, dark; section-home-light = CS101 Materials as a student; gradebook-light = instructor gradebook of CS101.
- Change 1 (README.md:7): catalog, my-sections and gradebook come from the capture seed, not seed-demo: gradebook-light shows emails `@northfield-muyj3ukc.edu` and CS101 not finalised (Finalize grades button, Lucas missing), while seed-demo finalises CS101 with northfield.edu emails. "the demo dataset" is not what is pictured.
- Change 2 (README.md:20-25): no warning for a port 5432 clash; the lane hit it, and a Homebrew/Postgres.app install hits it on a fresh clone.
- Nits: step 4 "it prints the sign-in links" reads as at startup; step 5 "cd apps/web" assumes the repo root; step 6 could name the button "Continue signing in"; Status omits P4-L4 (Google/Microsoft credentials) which is a Phase 4 lane, not in the Backlog; `npm test` is vitest watch mode in a terminal (same in apps/web/README.md, pre-existing).
## Requests
- to orchestrator: apply the two README edits (or relay to P11-L2); optional backlog item: add a Materials capture to e2e/screens.ts so section-home-*.png are reproducible.
## Blockers
none
