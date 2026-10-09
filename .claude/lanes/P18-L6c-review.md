# P18-L6c-review - review of sub-lane 6c (web review-attempts page)
Status: done
## Doing
Done: verdict approve, low findings only.
## Found
- Gates green in worktree: lint, typecheck, vitest src 495/495 (incl. no-literal-colors), build (routeTree.gen.ts unchanged after build).
- API use matches design §1/§6: changed-only grading entries, numeric points, feedback only when changed, setQueryData attempt + attemptsKey invalidated (Release note count refreshes).
- Low: copy deviation "Graded: x of y points" (design says "Automatic: …") needs orchestrator acceptance; "Correct answer:" singular for multi-correct; 400 with unmapped detail field is silent; dead bannerRef / h3 tabIndex; tests do not pin "feedback omitted when unchanged".
- P18-L6a's request to 6c (route + typed link) is satisfied.
## Requests
none
## Blockers
none
