# Section Content — Implementation Plan

> **For agentic workers:** executed by `/orchestrate` (orchestrator + `designer` → `worker` → `reviewer` lanes, one branch and worktree per lane). Each task is one lane. Global constraints, gates and git rules are those of `docs/plans/2026-10-05-platform-core-and-staffing.md`. Tests need `UNICOURSE_TEST_DATABASE_URL` and `UNICOURSE_TEST_OWNER_DATABASE_URL` (D48); the API process additionally needs `UNICOURSE_FILES_DIR` from this phase on.

**Goal:** every section gets a home page with a description, announcements that reach the roster by mail, and weekly materials with file upload and download; TAs can do all of it (spec §1).

**Architecture:** metadata in Postgres under RLS (three new tables, one new column), bytes on a local directory per university (`internal/blob`, D70), one Go package `internal/content` with the handlers, mail through the existing outbox, non-nested TanStack routes with a shared tab strip on the web.

**Tech Stack:** as Spec 1 §3; no new Go module, no new npm dependency (uploads use `fetch` with `FormData`, downloads are plain links).

**Spec:** `docs/specs/2026-10-06-section-content.md` — every task cites its sections. **Visual rules:** Spec 3, tokens and components as on `main` at build time (no literal colours; `src/no-literal-colors.test.ts` fails the gate otherwise).

**Coexistence with Phases 6 and 7.** Shared files are `apps/api/openapi.json`, `apps/web/src/api/types.ts`, `internal/authz` and `internal/course` (Phase 7 adds `credits`; this phase adds `description` and blob cleanup on section delete), and `routes/$slug/sections.$id.tsx` plus `features/courses/section-detail.tsx` (Phase 6 P6-L4, PR #63). P8-L1a starts after P7-L1a has merged (or rebases onto it before its PR if run in parallel); P8-L2 starts after P6-L4 has merged. Migration number `0006` (`0005` is grading). The Grades tab is rendered only if `routes/$slug/sections_.$id.grades.tsx` exists on `main` when P8-L2a builds; otherwise P8-L2b adds it.

## Global Constraints

- Go 1.27 standard library `net/http`; allowed modules pgx, x/oauth2, go-oidc, x/crypto only (Spec 1 §3).
- Every new table: `university_id`, RLS policy, composite tenant FKs (D19), `created_at`/`updated_at` with the trigger (D20), `(university_id, id)` unique for composite references.
- Contract conventions D16: no `null`, arrays always present, optional = omitted; error envelope of Spec 1 §6; audit rows in the request transaction.
- Naming per `CLAUDE.md`: packages `content`, `blob`; routes `/api/v1/universities/{slug}/sections/{id}/...`; JSON camelCase matching Go tags and TS types; env `UNICOURSE_FILES_DIR`.
- Files: 100 MB cap, extension allowlist of spec §6, downloads always `attachment` with `nosniff` and `Content-Security-Policy: sandbox`, content type from the allowlist map only.
- Content authorization only through `authz.CanEditContent` and `authz.CanReadContent` (spec §4).
- Web: Spec 3 tokens, Lucide icons, no literal colours, no toasts (D23), dialogs trap focus, status lines in live regions.
- Gates: `gofmt`, `go vet ./...`, `go test ./...` (`-race` on `internal/content`), `npm run lint`, `npm run typecheck`, `npm test`, `npm run build`; Playwright before the phase is marked done.

## Review Focus

1. **Upload never buffers the file** — a 100 MB upload keeps the API's resident memory flat (`r.MultipartReader()` streamed to a temp file while hashing); a body of 100 MB + 1 byte is `validation_failed` on `file`, not a connection reset or a 500. Pinned in Task 1b (boundary test with `httptest` and a counting reader).
2. **Nothing served can run in the app's origin** — `html`, `svg`, `js` and extensionless names are rejected at upload; every download carries `Content-Disposition: attachment`, `X-Content-Type-Options: nosniff` and `Content-Security-Policy: sandbox`; `.md` and `.csv` are served as `text/plain`. Pinned in Task 1b.
3. **TA parity and the student wall** — a TA edits and deletes an instructor's announcement and uploads to any week; an enrolled student gets 403 on every write and on another section's content; a plain member gets 403 on every content read while `GET /sections/{id}` still returns the description. Pinned in Task 1a (role matrix) and Task 2a (tabs per role).
4. **Mail goes to the roster minus the author, once** — `email: true` queues exactly one outbox row per enrolled student and staff member other than the author, with the subject `[CODE-NNN] title` and the Announcements link; `email: false`, edits and deletes queue nothing; an invited-but-unsigned-in person receives nothing. Pinned in Task 1a.
5. **Blob and row stay in step** — deleting a material, a week or the whole section removes the blobs after commit and only after commit (a rolled-back delete keeps the file); a missing blob on download is a logged `internal` error, not a 404; `prune-files` lists orphans and deletes only with `--delete`. Pinned in Task 1b.

## File Structure

```
apps/api/migrations/0006_section_content.sql                 sections.description; announcements, weeks, materials; RLS; indexes
apps/api/internal/blob/{blob.go,blob_test.go}                Dir{Root}: Put(ctx, universityID, r) → (tmpPath, sha256, size, err); Commit(tmpPath, universityID, id); Open(universityID, id) (*os.File, error); Remove(universityID, id); Walk(fn(universityID, id))
apps/api/internal/authz/authz.go                             CanEditContent, CanReadContent (+ table test)
apps/api/internal/content/{types.go,description.go,announcement.go,mail.go,week.go,material.go,download.go,handler.go} + tests   handlers, SQL, allowlist map
apps/api/internal/course/{section.go,handler.go}             Section.description in the shared SELECT; blob cleanup after section delete
apps/api/internal/config/config.go                           FILES_DIR (required, writable check)
apps/api/internal/app/app.go                                 wire blob.Dir and content.Register
apps/api/cmd/unicourse-admin/main.go                         prune-files [--delete]
apps/api/openapi.json · apps/web/src/api/types.ts            contract
docker-compose.yml · apps/api/README.md · .gitignore         ./data/files
apps/web/e2e/env.ts                                          UNICOURSE_FILES_DIR: a temp dir per run
apps/web/src/features/section/{section-tabs,home,description-dialog,announcements,announcement-dialog,materials,week-dialog,upload,material-dialogs,queries,file-icon}.tsx + tests
apps/web/src/features/courses/section-detail.tsx            shrinks to the Participants content (staff editor, roster)
apps/web/src/routes/$slug/{sections.$id.tsx,sections_.$id.announcements.tsx,sections_.$id.materials.tsx,sections_.$id.participants.tsx}
apps/web/e2e/fixtures/{lecture.pdf,notes.zip,slides.pptx}   small real files (a few KB each, generated, no third-party content)
apps/web/e2e/staffing.spec.ts                                extended
```

---

## Phase 8 — Section content

### Task 1 (P8-L1, D): content API

Spec §3, §4, §5, §6, §7, §9, §10, §11. Designer writes `docs/design/P8-L1.md`: migration SQL, contract JSON for every route and schema, the `blob.Dir` surface, the upload sequence (temp file → row → rename → commit → cleanup on failure), the download headers, the mail text, the `prune-files` output format, the a/b cut. Designer rules on anything the spec leaves open and reports it for PLAN.md.

- **P8-L1a** — migration, `blob.Dir` with its tests on `t.TempDir()`, `FILES_DIR` config and wiring, `authz` predicates with the role table test, `Section.description` and `PUT .../description`, announcements CRUD with pagination and the outbox mail, contract and generated types, audit rows, RLS entries in the per-table tenancy test, README and compose. Lands first so P8-L2a can start.
- **P8-L1b** — weeks CRUD with dense renumbering, materials upload (streamed, hashed, allowlist, 100 MB boundary), rename and move, download with headers, `Range` and `If-None-Match`, blob removal after commit for material, week and section deletes, `prune-files`, `-race` on the package, the remaining audit rows.

### Task 2 (P8-L2, D): section web

Spec §8, §10. Designer writes `docs/design/P8-L2.md` (tab strip and title band, Home layout, announcement list and dialog, week cards, upload control and status line, per-role rendering, empty states, copy). Starts when P8-L1a's contract is on `main` and P6-L4 has merged.

- **P8-L2a** — `features/section/queries.ts`, `section-tabs.tsx`, Home with the description editor, Participants (move the staff editor and roster out of `section-detail.tsx`), Announcements list and dialogs, the four route files, unit tests, Grades tab if the route exists.
- **P8-L2b** — Materials: week cards, week dialog and move, multi-file upload with status line and per-file errors, rename, move-to-week, delete dialogs, file icons, unit tests.

### Task 3 (P8-L3, W): end to end

Spec §1: extend `staffing.spec.ts` with the criterion (instructor writes the description, posts an emailed announcement, creates Week 1, uploads a PDF and a ZIP; TA posts and uploads a PPTX; student sees everything newest first, downloads the PDF and the bytes equal the fixture; a non-enrolled member sees the description only). Fixtures generated into `apps/web/e2e/fixtures/`; the outbox mail is read through the stdout mailer log as the existing magic-link test does.

Order: P8-L1 designer → L1a worker → PR → (L1b worker ∥ P8-L2 designer → L2a → L2b) → P8-L3. Each sub-lane reviewed and verified by the orchestrator over real HTTP (upload a real deck, download it, compare `sha256sum`) before its PR.
