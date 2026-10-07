# Spec 6: Section content - home page, announcements, weekly materials

Status: decided by the orchestrator under D8 delegation, owner to confirm; the owner asked for it to be built on 2026-10-07 (PR #65 superseded, renumbered from Spec 5 / Phase 8 / D70-D73) · Date: 2026-10-06 · Builds on Specs 1, 2 and 4; visual rules from Spec 3

## 1. Intent

A section gets a home of its own that staff and students use every week: a page describing the section, announcements from staff, and course material organised by week with downloadable files. Instructors and TAs have the same powers over content; only staffing and grade finalization stay instructor-only (Specs 1 and 4).

**Owner's brief (2026-10-06).** "In each course section expand its features for announcements, course personal page, weekly course material; a TA can upload data like pdf, png, zip or ppt same as professor." The owner shared a screenshot of a Moodle course page (tabs Course, Settings, Participants, Grades, Reports, More; a participants table with role, group, last access and status) as a reference for what is possible. Only the page structure is taken from it. No name, id number or any other datum from the image appears anywhere in this product, its tests or its fixtures.

**Success criterion.** An instructor writes the section description, posts an announcement that is emailed to the roster, creates "Week 1" and uploads a PDF and a ZIP to it. A TA signs in, posts a second announcement and uploads a PPTX to Week 1. A student signs in, opens the section, sees the description, both announcements newest first, Week 1 with three files, downloads the PDF and receives the same bytes that were uploaded. A member who is neither staff nor enrolled sees the description and nothing else. A second university sees none of it; the per-table tenancy test covers every new table. An automated end-to-end test asserts it.

## 2. Out of scope

Assignments and submissions · quizzes and question banks · forums and comments · groups · per-student last access, reports, completion tracking · Moodle's Settings, Reports and More menu items · rich text (description, notes and announcement bodies are plain text, line breaks kept) · attachments on announcements · drafts, scheduled or hidden-until-date content · file versioning · virus scanning · video streaming or transcoding · per-university storage quotas · copying content from a previous term · links as materials (a URL goes in a week's notes).

## 3. Domain model

New tables are tenant-scoped with RLS, composite tenant FKs (D19), `created_at`/`updated_at` (D20). Migration `0007_section_content.sql` (`0005` is Spec 4's).

| Entity | Fields | Rules |
|---|---|---|
| `sections` (existing) | + `description text not null default ''` | ≤ 10,000 characters; the section's home page text |
| `announcements` | `university_id`, `section_id`, `author_user_id`, `title text`, `body text`, `emailed boolean` | `title` 1..200, `body` 1..20,000; `emailed` records whether mail was queued on creation; FK `(university_id, section_id)` on delete cascade; index `(section_id, created_at desc)` |
| `weeks` | `university_id`, `section_id`, `title text`, `notes text default ''`, `position integer` | `title` 1..100, `notes` ≤ 5,000; FK `(university_id, section_id)` on delete cascade; index `(section_id, position)`; positions are renumbered 1..n by the server on every insert, move and delete, so they are dense and unique in practice but not constrained (reordering under a unique constraint needs deferred checks for nothing) |
| `materials` | `university_id`, `section_id`, `week_id`, `title text`, `file_name text`, `content_type text`, `size_bytes bigint`, `sha256 text`, `position integer`, `uploaded_by_user_id` | `title` 1..200 (defaults to the file name); `file_name` is the uploaded name, cleaned to its base name, ≤ 255; `content_type` from the extension allowlist in §6, never from the client; `size_bytes` 1..104,857,600; `sha256` lowercase hex; FK `(university_id, week_id)` on delete cascade and `(university_id, section_id)` on delete cascade; index `(week_id, position)`; renumbered per week like weeks |

**Blobs.** File bytes live outside the database, under `UNICOURSE_FILES_DIR/<university_id>/<material_id>` (D88). The row is the source of truth: a blob without a row is garbage, a row without a blob is an `internal` error on download and is logged. Per-university directories keep "move a university" a directory copy plus a `university_id` filter, matching Spec 1 §10. No deduplication: two uploads of the same deck are two blobs (upgrade: content-address by `sha256` when disk matters).

**Deletes.** Deleting a material, a week or a section cascades the rows inside the transaction; the API removes the blobs after the commit succeeds, best effort, and logs a failure. The section delete handler in `course` collects the section's material ids before the delete and calls `blob.Remove` for each after commit, the one place `course` imports `blob`. A blob left behind by a crash between commit and removal is harmless and is cleaned by `unicourse-admin prune-files` (lists or removes blobs without a row), added in this spec because the orphan case is designed in.

**Audit.** `section.update` already covers the description (its `before`/`after` carry the field). New actions: `announcement.create/update/delete`, `week.create/update/delete` (`update` includes moves), `material.create/update/delete` (`after` holds metadata, never bytes). Written in the request transaction as every other action.

## 4. Authorization

| Who | Can |
|---|---|
| instructor or TA of the section (same powers) | write the description; create, edit and delete any announcement of the section (not only their own); create, edit, move and delete weeks; upload, rename, move and delete materials; read everything |
| admin; dept_admin of the course's department | the same as staff |
| enrolled student | read the description, announcements, weeks and materials; download |
| any other active member | read the description (it is part of `Section`, which the catalog shows); every other content route is 403 |

Two new predicates in `internal/authz`, next to `CanReadRoster`: `CanEditContent(ctx, tx, scope, sectionID)` (admin, dept_admin of the department, any `section_staff` row) and `CanReadContent(ctx, tx, scope, sectionID)` (`CanEditContent` or an enrollment). Handlers call one of them before every content read or write; RLS underneath bounds any mistake to the university.

## 5. API

New routes under `/universities/{slug}`; conventions D16 (no `null`, arrays always present); the error envelope and pagination of Spec 1 §6.

| Route | Who | Notes |
|---|---|---|
| `Section.description` on `Section` | existing readers | always present; empty string when unset |
| `PUT /sections/{id}/description` | content editors | body `{ description }`; returns the `Section`; audited as `section.update` |
| `GET /sections/{id}/announcements` | content readers | paginated (`limit`, `cursor`), newest first. `Announcement = { id, sectionId, title, body, author: { userId, displayName }, emailed, createdAt, updatedAt }` |
| `POST /sections/{id}/announcements` | content editors | body `{ title, body, email }` (`email` boolean, default `true`). With `email: true`, one outbox mail per enrolled student and staff member of the section other than the author is queued in the same transaction (§7). 201 with the `Announcement` |
| `PATCH,DELETE /sections/{id}/announcements/{announcementId}` | content editors | `PATCH { title?, body? }`; editing never re-sends mail |
| `GET /sections/{id}/weeks` | content readers | `{ items: Week[] }`, not paginated (a section holds at most 100 weeks and a week at most 200 materials; creating the 101st week or the 201st material is `validation_failed` on `weeks` / `materials`). `Week = { id, sectionId, title, notes, position, materials: Material[] }` ordered by position; `Material = { id, weekId, title, fileName, contentType, sizeBytes, position, uploadedBy: { userId, displayName }, createdAt }` |
| `POST /sections/{id}/weeks` | content editors | `{ title, notes? }`; appended last; 201 with the `Week` |
| `PATCH,DELETE /sections/{id}/weeks/{weekId}` | content editors | `PATCH { title?, notes?, position? }`; `position` moves the week and the server renumbers the rest; delete cascades materials and removes their blobs |
| `POST /sections/{id}/weeks/{weekId}/materials` | content editors | `multipart/form-data`, field `file` (required, one file) and `title` (optional). Body limit 100 MB plus 1 MB of overhead through `http.MaxBytesReader`; a larger body is `validation_failed` on `file` ("larger than 100 MB"). The file's extension must be in the allowlist (§6), else `validation_failed` on `file`. 201 with the `Material` |
| `PATCH,DELETE /sections/{id}/materials/{materialId}` | content editors | `PATCH { title?, weekId?, position? }` (`weekId` moves the file to another week of the same section; a week of another section is `validation_failed` on `weekId`); delete removes the blob after commit |
| `GET /sections/{id}/materials/{materialId}/download` | content readers | the bytes. Headers: `Content-Type` from the row, `Content-Disposition: attachment; filename*=UTF-8''<encoded name>`, `Content-Length`, `ETag: "<sha256>"`, `Cache-Control: private, no-store`, `X-Content-Type-Options: nosniff`, `Content-Security-Policy: sandbox`. Served with `http.ServeContent` from the blob, so `Range` and `If-None-Match` work. The contract describes the response as `application/octet-stream` (the contract test only matches routes) |

**Upload flow.** The handler checks `CanEditContent`, parses the multipart stream (`r.MultipartReader()`, not `ParseMultipartForm`, so a 100 MB deck never lands in memory), writes the part to `<files dir>/<university_id>/.tmp/<random>` while hashing it, then in one transaction inserts the row with the size and hash, writes the audit row, renames the temp file to `<university_id>/<material_id>` and commits. A failure after the rename removes the blob. Uploads are one file per request; the web sends several files as sequential requests.

## 6. File types

Allowed extensions and the content type served for each; the list is one Go map in `internal/content` and the only place that knows it:

| Extension | Served as |
|---|---|
| `pdf` | `application/pdf` |
| `png`, `jpg`/`jpeg`, `gif`, `webp` | `image/png`, `image/jpeg`, `image/gif`, `image/webp` |
| `zip` | `application/zip` |
| `ppt`, `pptx` | `application/vnd.ms-powerpoint`, `application/vnd.openxmlformats-officedocument.presentationml.presentation` |
| `doc`, `docx` | `application/msword`, `application/vnd.openxmlformats-officedocument.wordprocessingml.document` |
| `xls`, `xlsx` | `application/vnd.ms-excel`, `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet` |
| `txt`, `md`, `csv` | `text/plain; charset=utf-8` (all three; never `text/markdown` or `text/csv` so nothing renders inline) |

Everything else (notably `html`, `svg`, `js`, executables) is rejected at upload. Downloads are always attachments with `nosniff` and a sandbox CSP, so even an allowed type never executes in the app's origin. A `.PDF` is a `pdf` (case-insensitive); `deck.tar.gz` is rejected (`gz` is not listed); the extension is the text after the last dot of the cleaned base name.

## 7. Announcement mail

`POST .../announcements` with `email: true` enqueues through `outbox.Enqueue` in the request transaction: recipients are the emails of the section's enrolled students and staff, minus the author; subject `[<course code>-<section number>] <title>`; text body = the announcement body, a blank line, "Posted by <display name> in <course code> <course title>, <term name>", and the link `<BASE_URL>/<slug>/sections/<id>/announcements`. Mail is not urgent (sign-in links stay ahead of it). The sender delivers one mail at a time per process (PLAN.md backlog); a 300-person section drains in a few minutes, which is acceptable for announcements. No mail is sent for edits or deletes, and nothing is sent to invited-but-not-yet-signed-in people (they have no user row).

## 8. Web app

New feature directory `apps/web/src/features/section/` for the section's own pages; the existing staff editor and roster move from `features/courses/section-detail.tsx` into the Participants tab here. Routes under `routes/$slug/`:

| Tab | Route | Content | Who sees the tab |
|---|---|---|---|
| Home | `sections.$id.tsx` | Title band (class code, course title, term, your role badge) as P6-L4 built it; the description as pre-wrapped text with an Edit button for content editors (dialog with a textarea, 10,000-character counter); staff with role badges; meeting times | every member who can read the section |
| Announcements | `sections_.$id.announcements.tsx` | list newest first: title, body (pre-wrapped), "Posted by <name> · <date>", "Emailed" text badge when `emailed`; "Load more" for the next page; content editors get "New announcement" (dialog: title, body, checkbox "Also email everyone in this section", checked by default), Edit and Delete (confirm dialog) | content readers |
| Materials | `sections_.$id.materials.tsx` | weeks in order as cards: title, notes (pre-wrapped), materials as rows with a Lucide icon by type (`file-text`, `image`, `file-archive`, `presentation`, `file`), title, file name, size (`1.2 MB`), uploader, date, and a Download link (`<a href=".../download" download>`; the browser carries the session cookie). Content editors get "Add week" (title prefilled "Week <n+1>"), Edit week, Move up / Move down, Delete week (confirm, names the number of files), "Upload files" (native `<input type="file" multiple accept=".pdf,.png,…">`, one request per file, an inline status line "Uploading 2 of 3: lecture.pdf", errors listed per file and the rest continue), Rename, Move to week (select), Delete file (confirm) | content readers |
| Participants | `sections_.$id.participants.tsx` | the existing staff editor and roster (table with name, email, role badge, remove, CSV import) exactly as on `main`, moved; students see staff only | staff, admin, dept_admin; students see it with staff only; other members do not see the tab |
| Grades | existing `/{slug}/sections/{id}/grades` (Spec 4, `sections_.$id.grades.tsx` is on `main`) | link only | whoever Spec 4 lets open the gradebook |

The tab strip is one component (`section-tabs.tsx`), rendered by each page under the shared title band, with the current tab marked (`aria-current="page"`); tabs the viewer cannot open are not rendered rather than disabled. Non-nested route files (`sections_.$id.*`) follow Spec 4 §6 so every page owns its own loader. Query keys per resource (`[slug, 'sections', id, 'announcements']`, `[slug, 'sections', id, 'weeks']`), invalidated on every write. Deep links from announcement mail land on the Announcements tab.

Accessibility as Spec 1 §7 and Spec 3: every control labelled, dialogs trap focus, upload status in a live region, file type icons `aria-hidden` next to the visible name, no colour-only state, no literal colours (Spec 3 gate).

## 9. Configuration and operations

- `UNICOURSE_FILES_DIR` (required): an existing writable directory. The API creates `<university_id>/` and `.tmp/` under it on demand. Startup fails with the usual missing-variable list when unset, and with a clear error when the directory is not writable.
- `apps/api/docker-compose.yml` and `apps/api/README.md` document a local `apps/api/data/files` (gitignored). CI's e2e step and the Playwright config export a temporary directory; Go tests use `t.TempDir()`.
- Backups: the database and the files directory are backed up together; a restore of one without the other yields the row-without-blob or blob-without-row cases above, both of which are logged and non-fatal.
- `unicourse-admin prune-files [--delete]`: lists (default) or removes blobs under the files directory that no `materials` row references, and reports rows whose blob is missing.
- Ceiling (D88): one files directory means every API process must mount the same volume; an S3-compatible store is the upgrade when a deployment cannot share one.

## 10. Testing

| Layer | What |
|---|---|
| `authz` | `CanEditContent` and `CanReadContent` for every role: admin, dept_admin of the department and of another department, instructor, TA, enrolled student, plain member, member of another university (table-driven, like the existing predicates) |
| `content` (Go, real Postgres, `httptest`) | description round-trip and length limit; announcements CRUD, ordering, pagination, TA edits an instructor's post, mail rows queued for roster minus author with the subject and link, `email: false` queues nothing, edits queue nothing; weeks CRUD, append position, move renumbers densely, delete renumbers and removes blobs; materials upload (good PDF, `.PDF`, 100 MB boundary accepted, 100 MB + 1 byte rejected, `html` and `svg` rejected, empty file rejected, missing `file` part, title default), rename, move between weeks, move to another section's week rejected, delete removes the blob, download headers and bytes equal the upload, `Range` request, `If-None-Match` 304, missing blob → 500 and a log line; cascade from section delete removes blobs; audit rows per action; role matrix per route; RLS on the three new tables (extend the per-table tenancy test); `prune-files` lists an orphan blob and a missing blob, deletes only with `--delete` |
| contract | routes in Go == routes in `openapi.json` (existing test) |
| web (Vitest) | tabs render per role; announcements list, dialog validation and error envelope display, email checkbox default; materials: week cards in order, upload status line and per-file error, rename and move dialogs; home description edit; no literal colours |
| end to end | the §1 criterion, added to `staffing.spec.ts`: upload real small files from `e2e/fixtures/`, download and compare bytes |

Gates as every lane: `gofmt`, `go vet ./...`, `go test ./...` (with `-race` on the content package), `npm run lint`, `npm run typecheck`, `npm test`, `npm run build`; Playwright before the phase is marked done.

## 11. Internal layout

```
apps/api/
├── internal/content/        description, announcements, weeks, materials, download, mail text, type allowlist + handlers and tests
├── internal/blob/           Dir: Put(stream) → (path, sha256, size), Open, Remove, Walk; the only package that touches UNICOURSE_FILES_DIR
├── internal/authz/          CanEditContent, CanReadContent
├── cmd/unicourse-admin/     prune-files
├── migrations/0007_section_content.sql
└── openapi.json

apps/web/src/
├── features/section/        section-tabs, home, description-dialog, announcements, announcement-dialog, materials, week-dialog, upload, material-dialogs, queries + tests
├── features/courses/        section-detail.tsx shrinks to what Participants needs (staff editor, roster)
└── routes/$slug/            sections.$id.tsx, sections_.$id.{announcements,materials,participants}.tsx
```

`internal/blob` is a package rather than a function in `content` only because `prune-files` and the tests need the same directory logic; it has no interface and one implementation (D88).

## 12. Scale and fit

A large university (Spec 1 §10: 5k sections per term) with 50 files of 5 MB per section stores ~1.25 TB per term; that is a volume-sizing question, not a design one, and the per-university directory layout moves a tenant to its own volume or an object store without a schema change. Downloads stream from disk with `sendfile`-class efficiency through `http.ServeContent`; uploads stream to disk; no request holds a file in memory. Announcement mail rides the existing outbox and its documented ceiling. Nothing assumes a week count, a language or a file naming scheme.

## 13. Deferred, with triggers

| Deferred | Add when |
|---|---|
| S3-compatible blob store (stdlib SigV4 or a PLAN.md-approved module) | first deployment whose API processes cannot share a volume |
| Content-addressed blobs (dedupe by `sha256`) | files directory passes the volume budget and duplicate decks are a measurable share |
| Storage quota per university | first customer contract with a storage clause |
| Markdown in descriptions, notes and announcements | first instructor asks; render client-side with a small, vetted dependency (PLAN.md decision) |
| Attachments on announcements | first instructor asks; `materials` already carries file metadata, add a nullable `announcement_id` |
| Hidden-until dates on weeks | first instructor asks; nullable `visible_from timestamptz` on `weeks`, filtered for students |
| Copy weeks and materials from a previous term's section | second term of a course on the platform |
| Upload progress bars | first complaint about large uploads; `XMLHttpRequest` progress events, no library |
| Virus scanning | first security review that requires it; ClamAV behind an outbox-style queue |
| Links and embedded video as materials | first instructor asks; `materials.url` nullable with `file_name` empty |
| Groups, last access, completion | first university that runs sections in groups (Moodle parity) |

## 14. Decisions needing the owner's call

Recorded in PLAN.md as "owner to confirm": blob storage on a local directory (D88); TA parity on content including editing another author's announcement (D89); announcement mail on by default through the outbox (D90); plain text everywhere, no rich text (D91).
