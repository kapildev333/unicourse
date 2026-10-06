# P2-L3a - Web: upload helper, CsvImportDialog, course list
Status: done
## Doing
Finished; awaiting orchestrator verification. P2-L3b is next.
## Found
- Spike: PASSED, no fallback needed. A jsdom `File` in `FormData` through `api.POST` (`bodySerializer: b => b as FormData`) reaches the stubbed `fetch` as `multipart/form-data`; `await req.formData()` (on the `req.clone()` mockApi hands to handlers) returns `checksum` and `file`. Tests assert the real content-type and checksum. `test-setup.ts` untouched.
- `src/api/upload.ts`: `previewImport(t, file)`, `commitImport(t, file, checksum)`, types `ImportPreview`, `ImportCounts`, `ImportTarget` (courses | sections | enrollments{sectionId}); six literal-path calls.
- `CsvImportDialog({ open, onClose, title, target, columns, hint?, onDone(result) })` and `importSummary(kind, counts)` in `features/courses/csv-import-dialog.tsx`. Body mounts only while `open`, so state resets on every open. P2-L3b: sections import = `columns ['courseCode','termName','number','instructorEmails','taEmails']` with `hint="Separate several emails with semicolons."`; roster = `['email','displayName']`, hint "Emails not yet in the university get an invitation." Parent's `onDone` should `await refresh()`, close, `setNotice(importSummary(kind, r.counts))` (see `course-list.tsx`).
- `RemoveDialog({ open, title, description, confirmLabel, pendingLabel, pending?, error?, copy?, onConfirm, onClose })`: controlled, no mutation inside; the page owns the `useMutation` (call `m.reset()` when opening). Props differ from the design sketch (`pending` added, `error` is the mutation error).
- `CourseDialog({ slug, mode, course?, defaultDepartmentId?, onClose, onSaved(code) })`: parent mounts it conditionally (it renders `<Dialog open>`); `defaultDepartmentId` is an added prop for the filter preselect.
- Shared: `Dialog` has `wide`; `Badge` has `className?` and a new `danger` tone (so `ActionBadge` error needs no conflicting class override; design said pass classes via `className`, which would have clashed with the neutral tone's `bg-surface`); `roleLabel`/`RoleBadge` know instructor/ta/student; `ActionBadge`.
- `features/courses/access.ts` was created here with only `canManageCourses` and `canManageDepartment` (needed by the layout and list; the design lists the file under L3b). P2-L3b: extend it with `SectionAccess`, `sectionAccess`, `personName` and write `access.test.ts`.
- `queries.ts` has `useAllCourses`, `useAllTerms`, `todayIso`, `defaultTermId` (all of §4.2, L3b uses them; `defaultTermId` has no test yet, L3b's `section-list.test.tsx` covers it).
- `courses-layout.tsx`: the "Sections" tab link is `to={'/$slug/courses/sections' as '/$slug/courses'}` because the route file does not exist yet. P2-L3b MUST create `routes/$slug/courses/sections.tsx`, regenerate the tree, and drop the cast.
- `routes/$slug/courses/index.tsx`: `validateSearch` is typed `(…): { departmentId?: string }`; the design's version returns a required `departmentId: string | undefined`, which makes every `<Link to="/$slug/courses">` (including AppShell's) demand a `search` prop. Use the optional type for `termId` in `sections.tsx` too.
- Dept_admin default filter: the list page itself calls `onDepartmentChange(firstManaged)` in an effect when no filter is set (design §3.1 says "the route passes it"; same behaviour, one place).
- Tests: in the course-list test, closed `<dialog>`s stay in the DOM, so find dialogs with `findByRole('dialog', { name, hidden: true })`. Page tests that need `useUniversity` render inside a memory router with `client.setQueryData(['me'], …)`.
## Requests
- to P2-L3b: see the access.ts, sections route and cast items above.
## Blockers
