# P2-L3b - Web: section list, section detail, staff editor, roster
Status: done
## Doing
Finished; awaiting orchestrator verification. P2-L3c (my sections + polish) is next.
## Found
- `access.ts` now has `SectionAccess`, `sectionAccess(membership, userId, section)`, `personName({displayName?, email})` (falls back to email); `access.test.ts` is a role x relation table.
- `SectionListPage({ slug, termId?, onTermChange })`, `SectionDialog({ slug, term, onClose, onSaved(code, number) })` (course picker is filtered to departments the user manages), `SectionDetailPage({ slug, sectionId })`, `StaffEditor({ slug, sectionId, sectionLabel?, staff, pendingStaff, canManageInstructors, canManageTAs })`, `Roster({ slug, sectionId, canEdit })`.
- Route `routes/$slug/courses/sections.tsx` (optional `termId` search), cast removed from `courses-layout.tsx`, `routeTree.gen.ts` regenerated; `routes/$slug/sections.$id.tsx` renders `SectionDetailPage`. Section detail back link: managers go to `/$slug/courses/sections?termId=`, others to `/$slug/me` (L3c must make that route exist as My sections, it does today).
- For L3c: reuse `Chips`-style person names via `personName`; the `my-sections` key prefix `[slug,'my-sections']` is already invalidated by every section/staff/roster mutation here. Section-list test pattern (memory router + `client.setQueryData(['me'], ...)`) and `vi.setSystemTime` to pin the default term.
- Tests: jsdom `<dialog>` has no autofocus, so "Cancel has focus" is asserted as "Cancel is the first button in the dialog". `StatusMessage` renders "✓ {message}" in one span, so match notices with a regex, not a string. `getByRole('status')` is ambiguous on pages with `LoadMore`; find the notice by text then `.closest('[role=status]')`.
- Notices from Staff add use the display name from the 201 body (`personName`), role word "TA" / "Instructor".
## Requests
None.
## Blockers
None.
