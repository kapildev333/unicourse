# Spec 12: Quiz and assignment follow-ups

Status: decided by the orchestrator under D8 delegation, owner to confirm (D148-D152) · Date: 2026-10-09 · Builds on Spec 11 (Phase 18); visual rules from Spec 3

## 1. Intent

Four deferred items of Spec 11 §8 and the Phase 18 backlog, asked for by the owner on 2026-10-09: staff can fix an answer key after students have started and every submitted attempt is regraded; an assignment may accept late submissions until a second date with a percentage penalty; a quiz may limit each attempt to a number of minutes inside its window; and the API server bounds how long a client may take to send a request.

**Success criterion.** An instructor edits a quiz whose students have already submitted, flips the correct choice of question 2 and saves: every submitted attempt is regraded, a reviewed question keeps its reviewed points, the gradebook column changes accordingly, and the audit log records a regrade per attempt. An assignment with "late until" a week after the due date and a 20% penalty accepts a submission after the due date, marks it late, and a grade of 15 is recorded in the gradebook as 12.00 while the student sees both numbers and the penalty. A quiz with a 10-minute limit started at 09:50 of a window ending at 10:30 is submitted automatically at 10:00; the student's countdown shows the attempt deadline, not the window. A client that opens a connection and sends its request body slowly is cut off after the configured read timeout, while a 100 MB upload on a slow link still completes inside the upload timeout. An automated end-to-end test asserts the first three; a Go test asserts the fourth.

## 2. Out of scope

Regrading manual assessments or finalized sections (the snapshot rule of Spec 4 stands: a regrade changes `scores`, never `final_grades`) · changing a quiz's structure (number of questions, kinds, number of choices) after attempts exist (still 409) · per-student extensions · penalties by hour or day (one flat percentage) · pausing an attempt · a write timeout on responses (downloads stream; `WriteTimeout` stays unset).

## 3. Domain model

Migration `0011_quiz_and_assignment_followups.sql`; conventions of Spec 11 §3.

| Entity | Change | Rules |
|---|---|---|
| `assessments` | + `time_limit_minutes integer` nullable (quiz only, 1..1440); + `late_until timestamptz` nullable and `late_penalty_percent numeric(5,2) not null default 0` (assignment only, 0..100) | check constraints: `time_limit_minutes` null unless `kind = 'quiz'`; `late_until` null or `> closes_at` and only when `kind = 'assignment'`; `late_penalty_percent` between 0 and 100 (D150, D151) |
| `attempts` | no column; **effective deadline** = `least(started_at + time_limit, closes_at)` when a limit exists, else `closes_at`, computed in SQL wherever the close is checked (`PUT attempts/me`, the sweeper's claim, `GET questions` for students) | `submitted_at` of an automatic submission = the effective deadline; `Attempt` JSON gains `deadlineAt` (D150) |
| `attempts.grading[*]` | entries gain `"manual": true` when a review set them | a regrade recomputes every entry without `manual`; `manual` entries keep their points and feedback (D149) |
| `submissions` | + `late boolean not null default false` | set when `submitted_at > closes_at` (any student write after the due date); cleared never (a late submission stays late even if replaced again) |
| `scores` row of a late submission | `points` = `score × (100 − late_penalty_percent) / 100`, rounded half up to 2 decimals; `submissions.score` keeps the raw grade | the penalty applies at grading time from the assessment's current percent; `Submission` JSON gains `late`, `penaltyPercent` (the assessment's), `finalScore` (the penalised number, present when `score` is) (D151) |

**Regrade** (D149): `PUT /sections/{id}/assessments/{assessmentId}/questions` is accepted after attempts exist when every item carries the `id` of an existing question, the count, order and kinds are unchanged, and each `choice` question keeps its number of choices; `prompt`, `points`, `choices[].text`, `choices[].correct` and `answers` may change. In the same transaction every submitted attempt is regraded: non-manual grading entries are recomputed from `answers` and the new keys, `score` and the `scores` row are rewritten, `max_points` is the new sum, and one audit row `attempt.regrade` per changed attempt (`before`/`after` score) plus `question.update` is written. Any structural change is still 409 "questions are locked once a student has started". Unsubmitted attempts are untouched (they are graded at submission against the current keys).

**Server timeouts** (D152): `http.Server{ReadHeaderTimeout: 10 s, ReadTimeout: UNICOURSE_READ_TIMEOUT (default 30 s), IdleTimeout: 120 s}`; the two upload handlers (materials, submission files) extend their own read deadline with `http.NewResponseController(w).SetReadDeadline(now + UNICOURSE_UPLOAD_TIMEOUT)` (default 10 min) before reading the multipart body. Both variables are whole seconds, validated at startup like `UNICOURSE_OUTBOX_SENDERS`. A client that exceeds a deadline gets the connection closed; the request log records status 499 as for an abandoned client (D32).

## 4. Authorization

Unchanged from Spec 11 §4: content editors set the limit, the late window and the penalty (same dialogs), regrade by saving questions, grade late submissions; students submit late when allowed and see `late`, `penaltyPercent` and `finalScore` once published.

## 5. API

- `Assessment` gains `timeLimitMinutes?` (quiz), `lateUntil?` and `latePenaltyPercent` (assignment; `latePenaltyPercent` always present for assignments, default 0). `POST`/`PATCH /assessments` accept them with the rules of §3 (`validation_failed` on the field: "only for quizzes", "only for assignments", "must be after closesAt", "must be between 1 and 1440", "must be between 0 and 100").
- `QuestionsPut` items accept an optional `id`; the response is unchanged. After attempts exist: ids required and the structural rules above; else 409 with the existing text. The `maxPoints` rule and all Spec 11 validation stay.
- `Attempt` gains `deadlineAt` (RFC 3339; equals `closesAt` without a limit). `PUT attempts/me` and `GET questions` use the effective deadline for the 409 "closed; your answers as of the closing time were submitted" / "closed" texts.
- `Submission` gains `late`, `penaltyPercent`, `finalScore?`. `PUT submissions/me` and the file routes accept writes while `now() < coalesce(late_until, closes_at)`; after `closes_at` the write marks `late`. The staff list rows gain `late`. `PATCH submissions/{userId}` (grade) stores the raw `score`, writes the penalised value to `scores`, returns `finalScore`.
- `GET /sections/{id}/gradebook` and `GET /me/grades` are unchanged in shape (they read `scores`).

## 6. Web

- Quiz dialog: "Time limit (minutes)" (optional, hint "Each attempt is submitted this many minutes after it starts, or at Closes, whichever is first."). Quiz list rows show "10 min each" after the window when set. Take page: the countdown and the "Closes in" line use `deadlineAt` ("Time left: 9 min"); the intro before "Start quiz" states the limit.
- Question editor: when attempts exist the editor is no longer locked; structural controls (add, remove, move, kind, add/remove choice) are disabled with a note "Students have started: you can fix prompts, points and answer keys; every submitted attempt is regraded when you save."; the Save button reads "Save and regrade" with a confirm `Dialog` naming the number of submitted attempts; the status line reports "Saved. N attempts regraded."; the 409 text for a structural change maps to the same note.
- Assignment dialog: "Late submissions until" (optional `datetime-local`, after Due) and "Late penalty (%)" (shown when the first is set). Assignment list rows: "Late until …" after the window. Student page: between Due and Late-until the form stays open with a caution `StatusMessage` "Late submissions are accepted until … with a 20% penalty."; after submitting late, "Submitted late" in the submitted line; once published, "Score 15 / 20, 12.00 after the 20% late penalty". Grading page: a "Late" `Badge` in the table and the penalised number next to the Score input ("Recorded as 12.00 after the 20% late penalty").
- Review page: regraded attempts show nothing new (the review inputs reflect the current points); a manual entry keeps its values.

## 7. Testing

| Layer | What |
|---|---|
| `grade` | the three new assessment fields per kind with their messages; `late_until > closes_at`; PATCH moving `closesAt` past `lateUntil` is 400 on `closesAt` |
| `quiz` | regrade: key flip regrades every submitted attempt, manual entries kept, unsubmitted untouched, `scores` rewritten, audit rows, structural change still 409, missing ids 400; time limit: `deadlineAt` in the Attempt, PUT after the deadline but before the close is 409, the sweeper submits at the deadline with `submitted_at` = deadline, a quiz without a limit behaves as before, limit longer than the remaining window clamps to the close |
| `assignment` | late window: write between due and late-until marks `late`, write after late-until is 409, no late window → 409 after due as before; grading a late submission writes the penalised `scores` row with half-up rounding, `finalScore` in the response, penalty 0 leaves it equal; student sees `late` always and `finalScore` only when published |
| `config` + `cmd/api` | the two timeout variables parsed and validated; a test server with `ReadTimeout` 1 s refuses a body that arrives over 3 s (status 499 in the log) while an upload handler with the extended deadline accepts the same slow body |
| web (Vitest) | dialog fields per kind, editor unlocked-with-note state and the confirm, countdown from `deadlineAt`, late states and copy, grading page penalty line; no literal colours |
| end to end | `quizzes.spec.ts` gains E-QUIZ-11 (time limit: a 1-minute limit on a long window; automatic submission observed before the close) and E-QUIZ-12 (regrade: flip a key after submissions, gradebook changes, reviewed points kept); `assignments.spec.ts` gains E-ASG-09 (late submission with penalty: due in the past, late-until ahead, student submits late, graded 15 → 12.00 shown) |

Gates as Phase 18 (D146 local gates).

## 8. Decisions (owner to confirm)

| # | Decision |
|---|---|
| D148 | Owner 2026-10-09: GitHub billing will not be fixed; the `ci` workflow stays disabled and every PR merges on the orchestrator's local gates (D146 made permanent); CLAUDE.md's rule says so |
| D149 | Regrade = the questions `PUT` with ids after attempts exist, structure locked (count, order, kinds, choice counts), keys/points/prompts editable; every submitted attempt is regraded except grading entries marked `manual` by a review; one `attempt.regrade` audit row per changed attempt |
| D150 | Per-attempt time limit as `assessments.time_limit_minutes`; the effective deadline is `least(started_at + limit, closes_at)` computed in SQL everywhere the close is checked; the sweeper submits at the deadline; `Attempt.deadlineAt` drives the countdown |
| D151 | Late submissions as `late_until` plus a flat `late_penalty_percent`; `submissions.late` set on any write after the due date; the raw grade stays on the submission and the penalised value goes to `scores`; students see both once published |
| D152 | Server `ReadHeaderTimeout` 10 s, `ReadTimeout` from `UNICOURSE_READ_TIMEOUT` (default 30 s), `IdleTimeout` 120 s; upload handlers extend their read deadline to `UNICOURSE_UPLOAD_TIMEOUT` (default 10 min) through `http.ResponseController`; no `WriteTimeout` |
