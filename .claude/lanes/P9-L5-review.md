# P9-L5-review - review of P9-L5 (grades)
Status: done
## Doing
Finished; verdict: request changes (M1 required, M2 orchestrator call); everything else optional.
## Found
- M1 finalize-dialog.tsx:183: at 390 the Grade select (inputClass w-full in an auto-width cell) collapses to about 40 px and shows no text (capture 62-light-390). Since the Letter column is gone, "Keep {letter}" was the only place the computed letter shows per row, so at 390 it is now invisible. Fix: `${inputClass} min-w-28!` (or `w-auto!`).
- M2 scale-editor.tsx:239: "Unsaved changes" goes through StatusMessage, which renders text-success, bg-success-soft and a Check icon: a green tick for an unsaved draft, against CLAUDE.md "status colours mean success only". S5-63 literally names StatusMessage, so this is a spec/convention conflict for the orchestrator. Fix: `<p role="status" className="text-sm text-ink-muted">{dirty && 'Unsaved changes'}</p>`. Related: notice "Grade scale saved" is not cleared on the next edit, so both green messages show at once; clear notice in `edit`/`load`.
- L1 gradebook.tsx:348 vs grade.ts:60: gradebook "(n missing)" counts published assessments only; finalize counts every assessment without a score (unpublished included, all get 0 at finalize). With an unpublished, unscored assessment the gradebook says nothing while finalize says "1 missing". Consistent with the brief's "Missing marker only for published"; orchestrator to decide whether the Percent count should match finalize.
- L2 gradebook.tsx:459: S5-54 Student `w-auto` is a no-op; the Student column is still ~540 px at 1440 (60-dark-1440). Worker reported this. Possible fix: `sm:w-px sm:whitespace-nowrap` on the Student th, then check the capture.
- L3 scale-editor.tsx:263: Remove is `ghost`, which is accent purple: 12 purple Remove links at 1440 (66). No neutral ghost exists in components/; P9-L7 (B6) should add one.
- Nits: `title="Student"` on the header th does nothing useful (gradebook.tsx:459, finalize-dialog.tsx:151); the email line in the capped row header is clipped without an ellipsis (add `truncate` to the span); the two always-rendered StatusMessage `<p>`s in the scale form add empty gap-4 slots above the table; no test for an explicit 0 not showing "Missing".
- Worker workarounds: `className="relative"` on the three tables is harmless (positions the table as containing block for sr-only spans, no visual change at 1440 or 390); `w-32!` etc. after inputClass work and are tested.
- e2e locators: row(p, email) still resolves (email kept for named students, shown once for unnamed); "Final" columnheader exact, "Finalize N grades", "Published scores (2)" (Disclosure summary), "Cumulative GPA" parent, "Save scale" + status "Grade scale saved" all resolve once.
## Requests
## Blockers
