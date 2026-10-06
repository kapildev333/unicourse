# Spec 2: Student course view and schedule

Status: decided by the orchestrator under D8 delegation, owner to confirm · Date: 2026-10-06 · Builds on Spec 1

## 1. Intent

A student signs in and can answer, without asking anyone: what am I taking this term, who teaches it, when and where does it meet, and does anything clash. A student can also browse the term's catalog and see the same for any section before asking to be added.

**Success criterion.** An admin gives two sections meeting times that overlap on Monday. A student enrolled in both signs in, opens the schedule for the current term, sees both sections with instructor names, class codes and times, and a clash warning naming the other section. The catalog, searched by course code, shows the professor and times of a third section and flags that it clashes with the student's schedule. An automated end-to-end test asserts it.

Interpretation of the brief ("who is professor, collisions, winter fall term, class code"): professor = instructors on `section.staff`; collisions = overlapping meeting times; term = the existing `terms` rows (`Fall 2026`, `Winter 2027`); class code = `course.code` plus `section.number` (`CS101-001`).

## 2. Out of scope

Student self-enrollment, drops, enrollment caps and waitlists (rosters stay import-driven; the SIS is the source of truth) · registrar class numbers (CRN) · rooms as entities (location is free text) · meetings with their own date ranges (half-term labs) · calendar export (iCal) · course content, grading, announcements · notifications when a timetable changes.

## 3. Domain model changes

One new table, nothing else changes shape.

| Entity | Fields | Rules |
|---|---|---|
| `section_meetings` | `id`, `university_id`, `section_id`, `weekday smallint` (1 = Monday … 7 = Sunday, ISO), `starts_at time`, `ends_at time`, `location text` (default `''`, max 100) | `starts_at < ends_at`; composite FK `(university_id, section_id)` on delete cascade (D19); RLS as every tenant table; index on `(section_id)`. Times are wall-clock in the university's timezone, no conversion anywhere. A section has zero or more meetings; a meeting is one weekly slot (Mon 10:00-11:15 is one row, Mon/Wed is two). |

Deliberately not modelled: a `kind` on terms (winter, fall) - terms are free-form rows and the UI picks "current term" by date; a clash as a stored fact - it is derived from meetings at read time.

## 4. Authorization changes

| Who | Change |
|---|---|
| any active `member` | reads `GET /courses` and `GET /sections` (the catalog is public inside a university; D22 made them 403 for plain members). `pendingStaff` is returned empty unless the caller can manage the section (invitee emails stay private). Rosters stay staff-only. |
| admin, dept_admin (own departments) | write meetings, as part of writing a section. |

No new `authz` predicate: list reads need membership only, which tenant middleware already guarantees.

## 5. API

Contract widenings only (D16); no new routes.

| Where | Change |
|---|---|
| `Section` | `meetings: Meeting[]` always present. `Meeting = { id, weekday (1-7), startsAt "HH:MM", endsAt "HH:MM", location }`. Ordered by weekday, startsAt. |
| `SectionCreate`, `SectionUpdate` | optional `meetings: MeetingInput[]`; when present it replaces the whole list (a timetable is edited as a unit). Max 20 rows; `startsAt < endsAt` per row (`validation_failed` on `meetings[i].endsAt`). Audit `before`/`after` on the section include meetings. |
| `GET /sections`, `GET /courses` | new query `q` (case-insensitive substring of course code or title, max 100 chars) and `departmentId` (uuid). Existing `termId`, `courseId` unchanged. Who: any member. |
| `GET /me/sections` | unchanged; `Section` now carries meetings, so the schedule needs nothing else. |
| Sections CSV import | optional column `meetings`: semicolon-separated slots `Mon 10:00-11:15 @ Room 101`; weekday as `Mon`..`Sun` (case-insensitive), location after `@` optional. Bad slot = `error` row. Additive rule D31: meetings are written only for a new section, or an existing section with no meetings; preview says `section exists; keeps meetings` otherwise. |

Clash detection is not an endpoint. Two meetings clash when they share a weekday and `a.startsAt < b.endsAt && b.startsAt < a.endsAt`. The web computes it from data it already has; one pure function with a unit test. Server-side detection is added when something other than the browser needs it (reports, SIS).

## 6. Web app

| Feature | Screen | Who |
|---|---|---|
| `my-sections` | `/{slug}/me` gains a term picker (default: the term containing today, else the next one starting) and two views of the same data: the existing list and a **schedule**: a Mon-Sun week grid (CSS grid, no library) with one block per meeting showing class code, title, instructor names, time and location. Clashing blocks get a "Clashes with CS102-001" line and a distinct border, text first, colour second. Sections with no meetings are listed under the grid as "No meeting times". Works for every role (an instructor sees their teaching timetable). | every member |
| `catalog` (new) | `/{slug}/catalog`: term picker, department filter, search box (`q`), paginated section rows: class code, title, instructors, meetings, location. A row whose meetings clash with the viewer's own schedule for that term carries a "clashes with …" note. Row links to the section page. | every member |
| `courses` | Section dialog gains a meetings editor: rows of weekday select, start and end `<input type="time">`, location. CSV import help text lists the `meetings` column. Section detail shows meetings. | admin, dept_admin |

Routing adds `/{slug}/catalog`. Nav for a plain member: My sections, Catalog. Section detail stays read-only for students and never shows the roster.

Accessibility: the week grid is also readable linearly (each block is a list item under a weekday heading for screen readers); time inputs are native; clash text is explicit, never only a colour.

## 7. Testing

| Layer | What |
|---|---|
| `course` | meetings round-trip on create, patch (replace, clear with `[]`), get, list; validation (`endsAt` before `startsAt`, weekday 8, 21 rows); cascade on section delete; `q` and `departmentId` filters; plain member reads lists and gets empty `pendingStaff`; RLS test for the new table (extend the per-table tenancy test). |
| import | `meetings` column parse: good slot, bad weekday, bad time, missing `@` location; additive rule on an existing section with and without meetings. |
| web | `overlaps()` unit test; schedule renders blocks and the clash line; catalog search calls the API with `q` and flags a clash; meetings editor validation display. |
| end to end | extend `staffing.spec.ts` with the §1 criterion. |

## 8. Deferred, with triggers

| Deferred | Add when |
|---|---|
| Self-enrollment, caps, waitlists | first university that runs open enrollment without an SIS |
| Registrar class number (CRN) on sections | first SIS import that carries one; additive nullable column |
| Meeting date ranges | first half-term lab or exam slot; add nullable `starts_on`/`ends_on` to `section_meetings` |
| Server-side clash report (students double-booked across a term, instructor double-booking) | first admin asks; same predicate in SQL over `section_meetings` |
| iCal feed `GET /me/schedule.ics` | first student asks; stdlib text formatting with `RRULE` bounded by the term |
| Rooms as entities, room clashes | Spec 3 scheduling |
