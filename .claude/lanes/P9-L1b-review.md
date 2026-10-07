# P9-L1b-review - review of P9-L1b (shell, no-access page, capture script)
Status: done
## Doing
Finished: verdict changes needed.
## Found
Verdict: CHANGES NEEDED (header layout regressions at 390 and 768-1279).
1. HIGH app-shell.tsx:86 + :122 - 390 row one overflows its grid-cols-2 cell: trailing group (theme 106 + Sign out 114, 268 px on touch) is justify-self-end in a 159 px column and paints over the wordmark/university name. Measured live (vite preview, mocked /me, admin on /admin): 1 membership trail.left 134 vs wordmark.right 141 (fine pointer), 98 (coarse); 2 memberships 96 / 60 - the select covers "UniCourse". Visible in every 390 capture (20, 30, 45, 66: "N..." under the switch). Not caught by WIDE (no scrollWidth growth). Fix: base `grid-cols-[minmax(0,1fr)_auto]`, and row one has to lose width (e.g. university name and select off row one below md, Sign out icon-only below md with aria-label/title) - a design call, escalate.
2. HIGH app-shell.tsx:86 - one-row grid starts at md (768), main started it at xl (1280). 768: search squeezed to a 60 px pill over the theme switch; 2 memberships: wordmark, nav, search and select overlap. 1024: search 85-120 px, placeholder clipped "Find a c". Fix: one-row grid at `xl:` (keep md..xl as two/three rows like main); spec S5-04 only rules below 768.
3. MEDIUM app-shell.tsx:86/:48 - S5-03 outcome not met: search left x at 1440 is 540 (student) vs 671 (admin), spec cites 544 vs 673 as the problem. `auto_1fr_auto` + mx-auto still follows nav length. Spec wording is self-contradictory; orchestrator to decide (e.g. fixed-width left column, or drop the "does not move" claim).
4. MEDIUM routes/$slug/route.tsx:46-64 - "Go to Harbor College" and "Sign out" both carry mt-6 as inline-flex siblings: no gap between them and baseline offset (Go to top 330, Sign out 327 on 04 light 390 and dark 1440). Fix: wrap both in `<div className="mt-6 flex flex-wrap gap-3">`, drop per-button mt-6 (single-membership branch only, or always).
5. LOW route.tsx:46 - navigation as <button>+navigate: no href, so no middle-click/open-in-new-tab/copy link. No primary-styled Link exists on main (bg-accent only in button.tsx), and spec S5-13 says primary `Button`, so accepted; follow-up: export Button's class recipe for Link in components/ (Task 1a scope).
6. LOW app-shell.tsx:107-121 - regexes match the whole pathname, not the segment after the slug; slugs like `admin`, `courses`, `me` are not reserved (apps/api/cmd/unicourse-admin/main.go:48), so /admin/me marks My sections and Admin. Fix: prefix `^\/[^/]+\/` to each regex.
7. LOW app-shell.tsx:86-122 - focus order at 390 (nav, search, theme, Sign out) differs from visual order (theme and Sign out on row one). Inherent to the order-* approach; acceptable if noted, or move the trailing group before nav in DOM and reorder at md.
OK: `/` handler (INPUT/TEXTAREA/SELECT/contenteditable skip, meta/ctrl/alt skip, shift allowed for layouts, removed on unmount); S5-05 two-current impossible on real routes, other routes' current unchanged (no catalog/me child routes); TabNav snap; 72/73 flows mutate only grade scale (same values) and CS210 (no later capture uses it); no literal colours; e2e files untouched; scope = listed files + route.test.tsx.
Gates: lint ok, typecheck ok, test 37 files/251 passed, build ok, e2e 2 passed, screens red by design; WIDE = 21, 23, 43, 44, 45, 46, 47, 52, 53, 60, 61, 62, 63, 73 exactly (73 at 737).
Header height at 390 (capture, border included): 173 px on 20, 30 (light and dark), 45, 66; main was 205; 187 on touch (pointer coarse). Target ~150 not reached.
## Requests
- to orchestrator: decide row-one content at 390 (finding 1) and S5-03 intent (finding 3).
## Blockers
