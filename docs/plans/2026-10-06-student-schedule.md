# Student Course View and Schedule — Implementation Plan

> **For agentic workers:** executed by `/orchestrate` (orchestrator + `designer` → `worker` → `reviewer` lanes, one branch and worktree per lane). Each task is one lane. Global constraints, gates and git rules are those of `docs/plans/2026-10-05-platform-core-and-staffing.md` and are not repeated.

**Goal:** a student sees, per term, what they take, who teaches it, when and where it meets, and what clashes; a catalog answers the same for any section (spec §1).

**Spec:** `docs/specs/2026-10-06-student-schedule.md` — every task cites its sections.

**Order:** Task 1a lands the contract and migration first so Task 2 (web) runs in parallel with Task 1b. Task 3 last.

## Review Focus

1. **Replace semantics** — `PATCH` with `meetings: []` clears, without `meetings` keeps; never a partial merge. Pinned in Task 1a.
2. **Boundary times** — `10:00-11:00` and `11:00-12:00` do not clash; `10:00-11:01` and `11:00-12:00` do. Pinned in Task 2 (`overlaps` test) and in the import parser (Task 1b).
3. **Privacy** — a student listing sections gets `pendingStaff: []` and 403 on every roster; a dept_admin still sees pending staff. Pinned in Task 1b.
4. **Current-term pick** — a university with no term containing today picks the next upcoming term, and with none at all shows an empty state, not a crash. Pinned in Task 2.
5. **CSV additive rule** — re-importing a timetable file never overwrites meetings an admin edited by hand. Pinned in Task 1b.

## File Structure

```
apps/api/migrations/0002_section_meetings.sql
apps/api/openapi.json                         Meeting, MeetingInput, Section.meetings, q/departmentId params
apps/api/internal/course/meeting.go           parse/validate input, replaceMeetings(tx, sectionID, []MeetingInput), scan into Section
apps/api/internal/course/{section,import}.go  wire meetings, filters, member read, pendingStaff masking, meetings column
apps/web/src/api/types.ts                     regenerated
apps/web/src/lib/schedule.ts                  overlaps(), clashesFor(), currentTerm(), weekday labels
apps/web/src/features/my-sections/{schedule.tsx,term-picker.tsx}
apps/web/src/features/catalog/{catalog.tsx,queries.ts}
apps/web/src/features/courses/meetings-editor.tsx
apps/web/src/routes/$slug/catalog.tsx
apps/web/e2e/staffing.spec.ts                 extended
```

---

## Phase 5 — Student schedule

### Task 1 (P5-L1, D): meetings in the API, catalog reads, import column

Spec §3, §4, §5, §7. Designer writes `docs/design/P5-L1.md`: migration, contract text, the `meeting.go` surface, the import slot grammar, and the a/b cut below (D7).

**P5-L1a — migration, contract, meetings read/write**
- [ ] `0002_section_meetings.sql`: table per spec §3 with composite FK, checks, RLS policy, `updated_at` trigger, index.
- [ ] Extend the per-table tenancy test to `section_meetings`.
- [ ] `openapi.json`: `Meeting`, `MeetingInput`, `Section.meetings` (required), `SectionCreate.meetings`, `SectionUpdate.meetings`; regenerate `types.ts`. Contract test green.
- [ ] `meeting.go`: validate (weekday 1-7, `HH:MM`, start < end, ≤ 20 rows, location ≤ 100, no NUL), `replaceMeetings` (delete + insert in the section's transaction), load meetings into the shared section SELECT (one extra query per page, or `json_agg` in the existing SELECT — designer's call).
- [ ] Tests: create with meetings, patch replace, patch `[]`, patch without the field, get, list, validation detail paths `meetings[1].endsAt`, cascade on delete, audit `after.meetings`.
- [ ] Commit; orchestrator opens the PR so Task 2 can start against the contract.

**P5-L1b — catalog filters, member read, import column**
- [ ] `GET /sections`, `GET /courses`: `q` (ILIKE on `code`, `title`, escaped `%` and `_`) and `departmentId`; contract params; tests.
- [ ] Remove the plain-member 403 on both lists; mask `pendingStaff` to `[]` when the caller is a plain member (section manage rights for members are per section and would cost a query per row; a member never sees pending staff). Tests for member, dept_admin, admin.
- [ ] Sections import: `meetings` column parsed by the `meeting.go` slot grammar; error rows for bad slots; additive rule per spec §5; preview message `section exists; keeps meetings`. Tests with a fixture.
- [ ] Gates; commit.

### Task 2 (P5-L2, D): schedule, catalog, meetings editor

Spec §6, §7. Designer writes `docs/design/P5-L2.md`: schedule grid layout (column per weekday, rows per 30 minutes, block spans), empty states, catalog row, meetings editor, term picker behaviour. Starts when P5-L1a's contract is on `main`.

- [ ] `lib/schedule.ts`: `overlaps(a, b)`, `clashesFor(section, others)`, `currentTerm(terms, today)`; unit tests incl. boundary times and no-term.
- [ ] `my-sections`: term picker (stored in the URL search param), list/schedule toggle, schedule grid with blocks, clash line and distinct border, "No meeting times" list; tests against the contract-shaped mock.
- [ ] `catalog`: route, term picker, department filter (reuse the admin department picker), debounced search → `q`, paginated rows, clash note computed against `GET /me/sections` for that term; tests.
- [ ] `courses`: meetings editor in the section dialog (add/remove rows, native time inputs, field errors from `details`), meetings on section detail, CSV help text; tests.
- [ ] Nav: plain member sees My sections and Catalog.
- [ ] Gates; commit.

### Task 3 (P5-L3, W): end to end

Spec §1, §7.
- [ ] Extend `staffing.spec.ts`: admin sets overlapping meetings on two sections via the dialog, imports a roster with the student, creates a third section with a clashing slot; student signs in, picks the term, sees both blocks and the clash line, searches the catalog by code, sees the instructor and the clash note; student gets no roster.
- [ ] CI `e2e` job green; orchestrator re-runs locally.
