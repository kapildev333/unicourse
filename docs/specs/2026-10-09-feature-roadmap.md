# Feature roadmap: the gap matrix and the ranked phases after Phase 20

Status: written by the synthesis lane P20-L3 under D155-D156, owner to confirm · Date: 2026-10-09 · Builds on Specs 1-12

Sources: `docs/design/P20-L1.md` (the 2026 landscape; citations `[Sn]` are its §7 numbering, repeated in §7 here) and `docs/design/P20-L2.md` (UniCourse's inventory; ids X1-X62 deferred items, K1-K10 backlog, G1-G19 code gaps, its §5 persona dead ends and §6 cost signals). "L1 §3" means P20-L1 section 3. Nothing here is product code; the owner picks phases from §4 and answers §6, as with Spec 5 (D75).

## 0. Brainstorm

**Intent as understood.** UniCourse is a multi-tenant platform where a university admin, department admins, instructors, TAs and students run courses, sections and their teaching. Twelve specs in, it covers catalog, schedule, content, quizzes, assignments, grading and GPA (L2 §1). The owner asks which important features are still missing and in which order to build them.

**Who it is for.** The owner, who approves one phase at a time; the orchestrator, who turns an approved phase into a plan; and, behind them, the first pilot university, whose people would hit the dead ends in L2 §5 during their first term.

**What success looks like.**
- The owner can say "go" or "no" on each of five phases without reading the research.
- Each phase is buildable by the existing pipeline (designer, workers, reviewer, local gates under D148) in about one day, as Phases 18 and 19 were (Spec 12: 11 PRs in a day, PLAN.md journal).
- Every ranked item traces to landscape evidence (L1) or to a persona dead end with a cost signal (L2). Nothing is invented.

**Constraints that shape the ranking.**
- Stack (D1): Go standard library plus pgx, x/oauth2, go-oidc and x/crypto only, and no new npm runtime dependency without a decision. This rules out SAML, PDF and Markdown libraries and AI SDKs without an owner decision first.
- No AI provider decision and no data-processing terms. AI work also carries EU AI Act high-risk exposure if it grades [S59].
- No deployment target (K3, D115). Push notifications, object storage, metrics and the real OIDC smoke test (K4) all wait on it.
- One pipeline, one integration suite at a time. A phase is 3-8 lanes, and migrations and `openapi.json` are serial hot spots.
- The e2e suite is the regression arbiter (CLAUDE.md). A feature that rewrites existing copy or landing routes costs e2e churn, so additive placement is preferred.
- No customer yet (`v1.0.0` waits for the first university, D112). Triggers in L2 §2 that say "first university asks" cannot fire. The ranking therefore uses the landscape as a stand-in for the buyer.

**Alternative framings.**
1. *Table stakes first.* Close every "cheap but expected" row of L1 §5 in source-count order: notifications, calendar, discussions, rubrics, attendance, groups, peer review. Pro: matches procurement scoring (L1 §3). Con: it ignores the operational dead ends (term rollover, single enrollment) that hit a pilot in week one, and it orders by vendor parity, not by who is stuck.
2. *Student experience first.* Build what the largest persona lacks: a what-is-due view, notifications, a calendar feed, campus time, an installable app, discussions. Pro: L1 trend 6 (5 sources) and the student's dead ends (L2 §5). Con: it leaves the admin's second-term problem (G9, X33) and grading depth (rubrics are in 5 of 5 products) until late.
3. *Institution readiness first.* SAML, SIS sync, LTI, exports, privacy tooling, deployment. Pro: these are procurement gates [S38][S56]. Con: most of it needs an owner decision or a partner (§3.2), so it cannot run unattended, and none of it helps a user inside a term.

**Recommendation: "the term loop", a blend of 2 then 1, with 3 kept to its owner-free parts.** Order phases by when a pilot university would hit each dead end across one term, then a second one:
- Week 1, daily use: what is due, notifications, the calendar, campus time.
- The second term: rollover, copying content, roster fixes, exports.
- The teaching table stakes, in source-count order: discussions, rubrics, extensions, attendance.

Institution-readiness items that need no owner decision (exports, the WCAG 2.2 self-audit) ride inside those phases. The rest go to §3.2 as owner questions. Why: each phase removes dead ends named in L2 §5 and closes rows that L1 marks table stakes, and the first two phases are pure CRUD on data UniCourse already holds (L2 §6), so they are cheap and low-risk.

## 1. Gap matrix

Landscape: **TS** = table stakes (Y in 4-5 of the five products in L1 §1, or listed in L1 §3 table stakes); **D** = differentiator (gated, add-on or in 1-2 products, L1 §3); **E** = emerging (L1 §2 trend without broad shipping). UniCourse: **Yes / Partial / No**, with L2 ids.

| # | Capability | Landscape | UniCourse | L2 ids | Sources |
|---|---|---|---|---|---|
| 1 | Modules / structured content | TS | Partial: weeks, notes, files; plain text; per section only; no links, scheduling, attachments on announcements | X30-X34, G15 | [S6][S13][S40] |
| 2 | Assignments, late handling | TS | Yes (late penalty, Spec 12); no per-student extension | X57 | [S8][S19][S40] |
| 3 | Quizzes, question banks | TS | Partial: choice and blank, one attempt, no bank or shuffle, no essay | X50-X52 | [S1][S8][S11] |
| 4 | Rubrics | TS (5 of 5) | No | X22 | [S3][S14][S20][S29][S42] |
| 5 | Gradebook | TS | Yes (weights, finalize, GPA); no drop-lowest; only gradebook CSV out | X23, G8 | [S20][S29][S40] |
| 6 | Discussions | TS (4 of 5) | No; no contact path to staff | X28, G10 | [S3][S6][S11][S14] |
| 7 | Announcements, notifications | TS | Partial: announcements plus optional mail; no in-app, preferences or read state | G1, X21, X27, X53, G18 | [S19][S20][S22][S40] |
| 8 | Calendar, to-do | TS | Partial: weekly grid; no due-date view, no iCal | G2, X54, X20 | [S6][S14][S44] |
| 9 | Mobile | TS | Partial: responsive to 390 px; no PWA, app or push | G13, X6, X40 | [S36] |
| 10 | Attendance | D (native in 2, plugin in 2) | No | G11 | [S31][S32][S33] |
| 11 | Groups, group submissions | TS (4 of 5), E (trend 14) | No | X29, X55 | [S13][S23][S29][S45] |
| 12 | Peer review | D (3 of 5) | No | X22 | [S34][S35] |
| 13 | Accessibility conformance | TS, regulatory (2027-04-26) | Partial: AA contrast measured, axe scans; no WCAG 2.2 audit, no ACR | X41 | [S53][S56][S68] |
| 14 | SIS / roster exchange | TS | Partial: CSV import only; no export, no sync, no single add | X2, G8, G4 | [S43][S47][S64] |
| 15 | LTI 1.3 | TS (median 28 tools) | No | X3 | [S38][S47] |
| 16 | Programmatic access, data export | D | Partial: OpenAPI contract, cookie sessions only | G14 | [S14][S47] |
| 17 | Analytics, early alert | D (paid tiers), E (trend 5) | No (gradebook stats only) | X29, G18 | [S16][S33][S48] |
| 18 | Video, live classroom | D (add-ons) | No | X34 | [S30][S32][S46] |
| 19 | Plagiarism, proctoring | Partial and retreating (trend 2) | No, by design | X22, X56 | [S49][S50][S51] |
| 20 | Generative AI, instructor | TS-bound (base tier at Canvas, core at Moodle) | No | X5 | [S1][S2][S8] |
| 21 | Generative AI, student | D | No | X5 | [S14][S24][S26] |
| 22 | Course copy, date shift, reuse | TS (Sakai Date Manager), E (trend 13) | No | G9, X33, G15 | [S8][S21][S23] |
| 23 | Distributed administration | E (trend 12) | Yes (admin and department admin); no custom roles | X12 | [S13][S14][S21] |
| 24 | Privacy and data governance | E (trend 7, regulatory) | Partial: RLS, audit log; no session list, no personal export | G6, G7 | [S15][S16][S58][S69] |
| 25 | Digital credentials | E (trend 15, adoption unknown) | No | none | [S14][S21][S65] |
| 26 | Sign-in beyond Google and Microsoft | Not covered by L1 | No SAML | X1 | persona dead end (L2 §5 operator) |
| 27 | University settings, roster upkeep | Not covered by L1 | No settings edit, no single enrollment, no invitation resend | G3, G4, G5 | persona dead end (L2 §5 admin) |
| 28 | Times in campus time zone | Not covered by L1 | No: browser timezone and locale | G12 | persona dead end (L2 §5 student) |

Self-service sign-up, the entry point of the Canvas incident [S60], does not exist: universities are created by CLI and sign-in needs an allowed domain or an invitation (D13, D128). This answers L1 §6 Q7.

## 2. Candidate features

Size: **S** = one lane; **M** = one phase of 3-5 lanes; **L** = two phases, or blocked on an owner decision.

**R1 Upcoming (what is due).** Afterwards, a student opens My sections and sees, across their sections, open quizzes and assignments due in the next 14 days that they have not yet submitted. Each has its section, kind, deadline and a link. Staff see the same list for the sections they teach, with submitted counts. Personas: Stu, Ins, TA. Extends: `GET /me/grades` data in `internal/grade` (a new `GET /me/upcoming`) and a panel in `features/my-sections/` (L2 §6 G2, X54). Size S. Evidence: trend 6, 5 sources [S6][S14][S22][S44]; L1 §5 "activity overview". Risk: per-student submitted state needs a join over `attempts` and `submissions`. Keep the window bounded so the query stays indexed.

**R2 In-app notifications and mail preference.** Afterwards, every member has a header bell with an unread count and a notifications page, and can mark items read one at a time or all at once. Items are created for:
- a new announcement;
- a new quiz or assignment in a section they are enrolled in;
- a released grade (published or finalized).

Each member can turn off announcement mail for themselves. Personas: all. Extends: a new `notifications` table and package, fanned out in the request transaction by the producers in `internal/content` and `internal/grade`; a bell in `components/app-shell.tsx` (L2 §6 G1, X21, X27, X53). Size M. Evidence: L1 §5 first cheap-but-expected item [S22][S40]; read tracking in Blackboard [S19][S20]; Open edX digests [S22]. Risks:
- Fan-out to a 500-student roster is one `insert ... select` per event.
- No pruning in the first cut. The ceiling is table growth; add a CLI prune later.
- Due-date reminders need a scheduler and are left out; R1 covers the need.

**R3 Calendar feed.** Afterwards, a member copies a private URL into Google, Apple or Outlook calendar. The feed holds their section meetings (weekly, bounded by the term's `starts_on`/`ends_on`, in the university's time zone) and their quiz and assignment deadlines. They can rotate the URL. Personas: Stu, Ins, TA. Extends: one handler over `/me/sections` and assessment data plus a per-member secret token (L2 §6 X20). Size S. Evidence: L1 §5 "calendar / to-do view ... with an iCal feed" [S6][S44]. Risk: a calendar client cannot send cookies, so the token is a bearer credential. It must be read-only, revocable and kept out of the request log (D27, D32 pattern).

**R4 Campus time and locale.** Afterwards, every date and time shows in the university's time zone and locale whatever the browser's setting, with the zone named where it differs from the browser's. Personas: all. Extends: `timezone` and `locale` on the `university` object of `GET /me` (`internal/auth/handler.go`) and the two formatters in `apps/web/src/lib/format.ts` plus `features/quiz/window.ts` (L2 §6 G12). Size S. Evidence: persona dead end (L2 §5 student, G12); it is the base for R3's TZID and R1's deadlines. Risk: e2e universities are created with `--timezone UTC` (`e2e/helpers.ts`), so assertions on displayed times can shift on a non-UTC runner. The lane runs the full e2e suite.

**R5 Admin upkeep: settings, single enrollment, invitation resend.** Afterwards:
- an admin edits the university's name, email domains, time zone and locale on an Admin Settings tab;
- staff who manage a section add one student by email without a CSV;
- an admin resends an invitation with a fresh token and expiry.

Personas: Adm, DA, Op. Extends: one `PATCH` in `internal/tenant` (G3); `POST .../enrollments` reusing the import row handler in `internal/course` (G4); re-enqueue in `internal/tenant/invitation.go` (G5). Size S to M (three small items). Evidence: persona dead ends (L2 §5 operator, admin, invited person). Risk: changing email domains must not lock out existing members (D128 checks active membership, not domain).

**R6 Term rollover and copy.** Afterwards, an admin or department admin picks a source term and a target term. They get the target's sections created from the source's: number, meetings and staff, never students. Separately, staff copy a past section's weeks, materials, quizzes and assignments into their new section, with every date shifted by the difference between the term starts. Copies arrive unpublished. Personas: Adm, DA, Ins, TA. Extends: copy routines in `internal/course`, `internal/content`, `internal/quiz` and `internal/assignment` keyed by section (L2 §6 X33, G9, G15). Size M. Evidence: Sakai 25 Date Manager and bulk publish [S23]; reusable libraries, trend 13 [S8][S21][S14]; the X33 trigger "second term of a course on the platform" fires for every pilot. Risks:
- Material files are copied byte for byte. Storage doubles per copy, which is X35's ceiling.
- Quiz keys are copied, so the copy must stay staff-only until published.

**R7 Registrar exports.** Afterwards, staff download a section roster CSV and finalized grades as CSV, per section or per term for admins, and admins download the filtered audit log as CSV. Personas: Adm, DA, Ins. Extends: export handlers in `internal/course`, `internal/grade` and `internal/audit` beside the existing gradebook CSV (L2 §6 G8). Size S. Evidence: L1 §5 "CSV/API export of grades and rosters (and later OneRoster)" [S43]. Risk: a term-wide export must stream rather than build in memory.

**R8 Discussions.** Afterwards, each section has a Discussions tab where staff and enrolled students start threads and reply; staff pin, lock and delete. A student can start a thread visible only to staff, which covers G10's contact path. Personas: Stu, Ins, TA. Extends: a new package and section tab (`features/section/section-tabs.tsx`), and R2 producers (L2 §6 X28, G10). Size M. Evidence: §1 row 6, Y in 4 of 5 products [S3][S6][S11][S14]. Risks: moderation and abuse, so every post is audited and staff can delete. Plain text unless §6 Q7 says otherwise.

**R9 Rubrics.** Afterwards, staff attach a rubric (criteria × levels with points) to an assignment and grade by clicking levels; the total fills the grade. Students see the rubric before they submit and the marked rubric once the grade is published. Rubrics can be reused across the instructor's sections. Personas: Ins, TA, Stu. Extends: `internal/assignment`, `internal/grade` and `features/assignment/` (L2 §6 X22). Size M. Evidence: §1 row 4, Y in 5 of 5 [S3][S14][S20][S29][S42]; trend 2, assessment redesign towards feedback (6 sources) [S15][S17][S51]. Risk: points are `numeric` strings (D68), and the rubric total must equal the score cell.

**R10 Per-student extensions and accommodations.** Afterwards, staff give one student a later due date, late window or quiz close, or extra time on timed quizzes (a multiplier, the usual disability accommodation). The deadline SQL respects it. Personas: Ins, TA, Stu. Extends: the one deadline expression in `quiz/deadline.go` (D153) and the assignment window check (L2 §6 X57). Size S. Evidence: Blackboard second due dates (2026) [S19]; accommodations are part of the accessibility gate, trend 4 [S53][S54]. Risk: the sweeper must read per-student deadlines, so the D150/D153 interleave tests (K8) need extending.

**R11 Quiz depth.** Afterwards, staff allow several attempts (best or latest counts), shuffle questions and choices, and add essay questions graded in the existing review screen. Banks and shared pools come later. Personas: Ins, TA, Stu. Extends: `internal/quiz` question jsonb, the attempts unique key and grading (L2 §6 X50-X52). Size M. Evidence: §1 row 3 [S8][S11]; trend 13 [S8]. Risk: the attempts unique key changes, which touches the sweeper and regrade (D149, D154).

**R12 Attendance.** Afterwards, staff open a register for a meeting date (derived from `section_meetings` and the term dates) and mark each student present, late, absent or excused. Students see their own record and staff see per-student totals. Grades are not affected in the first cut (§6 Q10). Personas: Ins, TA, Stu. Extends: a new table keyed by section and date, and a section tab (L2 §6 G11). Size M. Evidence: Blackboard and Brightspace native, the others plugins [S31][S32][S33]; L1 §5. Risk: meeting dates assume the D35 wall-clock model, so holidays need a "no class" mark.

**R13 Needs attention (rule-based early alert).** Afterwards, staff see a per-section list of students matching any of three rules:
- missing a past-due quiz or assignment;
- below a threshold percentage;
- no sign-in for N days (or N absences once R12 exists).

Each entry gives the reason and links to the student. Personas: Ins, TA, DA. Extends: R1's queries, `sessions.last_seen_at` and the gradebook (L2 §6 X29, G18). Size S. Evidence: trend 5, 5 sources [S16][S48][S13][S33]. The rule-based version answers L1 §6 Q6, since vendors gate the predictive version behind paid tiers [S48][S33]. Risk: `last_seen_at` is per session, not per university, so the design must say what "seen" means.

**R14 Groups and group assignments.** Afterwards, staff split a section into groups, and an assignment can take one submission per group with one grade copied to members. Personas: Ins, TA, Stu. Extends: new group tables, `internal/assignment` (L2 §6 X29, X55). Size M. Evidence: §1 row 11 [S13][S23][S29][S45]; trend 14.

**R15 Peer review.** Students review assigned peers' submissions against the rubric. Needs R9. Size M. Evidence: 3 of 5 products [S34][S35].

(R16, rich text, moved to §3.2 as O7: it needs a dependency decision.)

**R17 Content depth.** Afterwards, staff:
- add a link (including a video URL) as a material;
- hide a week until a date (`weeks.visible_from`);
- attach files to an announcement.

Personas: Ins, TA, Stu. Extends: the `internal/content` materials model and `weeks` (L2 §6 X31, X32, X34). Size S to M. Evidence: §1 row 1 [S13][S40]; L2 §5 instructor dead ends.

**R18 Sessions and personal data export.** Afterwards, a member sees their signed-in sessions (device, last seen) and revokes the others, and can download their own data (profile, memberships, submissions, grades) as JSON. Personas: all. Extends: `internal/auth/session.go` and `/me` (L2 §6 G6, G7). Size S. Evidence: trend 7, 5 sources [S15][S16][S58]; trend 8, security [S16][S60]. Account deletion is excluded because it conflicts with the append-only audit and grade retention, and needs the owner's retention policy.

**R19 WCAG 2.2 AA self-audit and ACR draft.** Afterwards, the repo holds a conformance report that walks every WCAG 2.2 AA criterion over every page, with fixes landed for the new 2.2 criteria: focus appearance, target size (24 px) and accessible authentication. Personas: all, and university procurement. Extends: `components/`, `index.css`, `e2e/a11y.spec.ts`. Size S. Evidence: trend 4, 6 sources; the Title II date of 2027-04-26 [S53]; procurement asks for 2.2 AA [S56][S68]. Risk: a self-audit is not a third-party audit, which is an owner spend (§6 Q11).

**R20 Installable web app (PWA).** Afterwards, a phone user adds UniCourse to the home screen with an icon and a standalone window. No offline mode and no push. Extends: `apps/web/index.html` and `public/` (L2 §6 G13). Size S. Evidence: mobile is scored in procurement [S36]. Needs an icon (X44 has no logo, §6 Q4).

**R21-R24 Small items.**
- R21: course description and prerequisites text in the catalog (G16; student dead end).
- R22: app-wide search (G17, X49; Spec 5 B4 kept it out).
- R23: bulk roster and member actions and list counts (X46-X48; Spec 5 "wait").
- R24: drop-lowest per assessment category (X23; gradebook row [S29]).

Each is S except R22 (M). None has landscape evidence beyond table-stakes parity, so all four rank low.

## 3. Ranking

### 3.1 Ranked by impact × effort

Impact:
- **H**: table stakes in at least 4 of 5 products (§1) or a trend with at least 5 sources, and a dead end for at least one persona in L2 §5.
- **M**: one of the two.
- **L**: parity only.

Order: H before M before L, smaller size first within a level, then enablers before their dependants.

| Rank | Id | Impact | Size | Why here |
|---|---|---|---|---|
| 1 | R1 Upcoming | H | S | Trend 6 (5 sources) plus the student's "what is due" dead end; every input already exists in `/me/grades` |
| 2 | R19 WCAG 2.2 audit | H | S | Trend 4 (6 sources) with a legal date; cheap. Scheduled after the UI phases so one audit covers their pages (§4) |
| 3 | R2 Notifications | H | M | L1 §5's first expected item; removes four L2 dead ends (G1, X21, X27, X53); the outbox and producers exist |
| 4 | R6 Rollover and copy | H | M | Its trigger fires on every pilot's second term; Sakai, Moodle and Open edX ship it [S23][S8][S21] |
| 5 | R8 Discussions | H | M | Table stakes in 4 of 5; the only student-to-staff channel (G10) |
| 6 | R9 Rubrics | H | M | Table stakes in 5 of 5; trend 2 moves assessment towards feedback |
| 7 | R4 Campus time | M | S | Correctness for every date; an enabler for R1 and R3 |
| 8 | R5 Admin upkeep | M | S | Three dead ends, each one endpoint (G3, G4, G5) |
| 9 | R3 Calendar feed | M | S | L1 §5 expected; one handler plus a token |
| 10 | R7 Exports | M | S | The registrar's only way out of UniCourse without an SIS (G8) [S43] |
| 11 | R10 Extensions | M | S | Accommodations; one deadline expression to extend (D153) |
| 12 | R13 Needs attention | M | S | Trend 5 claimed without ML; strongest after R1 and R12 |
| 13 | R18 Sessions and export | M | S | Trend 7; security hygiene after [S60] |
| 14 | R17 Content depth | M | S-M | Instructor dead ends; links cover video without hosting it |
| 15 | R20 PWA | M | S | Mobile is scored [S36]; waits on an icon (§6 Q4) |
| 16 | R12 Attendance | M | M | Native in only 2 of 5; real but narrower demand |
| 17 | R11 Quiz depth | M | M | Spec 11 covered the common quiz; this is depth |
| 18 | R14 Groups | M | M | Trend 14 has 3 sources; needs R9 for group rubrics to pay off |
| 19-22 | R21, R24, R23, R22 | L | S, S, S, M | Parity or Spec 5 "wait" items |
| 23 | R15 Peer review | L | M | 3 of 5 products; depends on R9 |

### 3.2 Needs an owner decision first

| Id | Decision | Blocks | Evidence |
|---|---|---|---|
| O1 | AI provider, data terms, and whether any AI ships at all | Instructor generation (trend 1, 7 sources, moving to table stakes), student tutor (trend 3), summaries | [S1][S2][S8][S14][S24][S26]; EU AI Act high risk for grading [S59]; the hype vs use gap [S39] |
| O2 | Deployment target (host, region, object storage) | Release artifacts (K3), real OIDC (K4), web push, metrics (X9), S3 (X37), any pilot | D115; L2 §5 operator |
| O3 | SIS and LTI partner, and API tokens | SIS sync (X2, OneRoster or Edu-API [S43][S64]), LTI 1.3 consumer (X3, [S38]), tokens and webhooks (G14) | L1 §6 Q5 |
| O4 | Mobile strategy | Native apps (X6), web push, the PWA icon (R20) | [S36] |
| O5 | Billing | X4 | none in L1; no customer yet |
| O6 | SAML or Shibboleth module | X1; the stdlib has no XML-DSig, so it needs a dependency decision (D1) | L2 §6 X1 |
| O7 | Rich text (a dependency or our own renderer) | X30 in R8, R17 and question text | L2 §6 X30 |
| O8 | Self-enrollment and waitlists | X16 (D38 keeps rosters import-driven) | L2 §5 student |

## 4. Proposed phase order (the next five)

Phase numbers continue from Phase 20, and spec numbers from Spec 12. Every phase follows the established pattern: one designer lane, then workers, with `migrations/`, `openapi.json`, `types.ts` and Postman as serial hot spots, then one e2e lane, then the close (full suite, PLAN.md status PR, release).

| Phase | Name | Bundles | Spec it becomes | Lanes it would cut | Needs from the owner |
|---|---|---|---|---|---|
| 21 | Stay on top | R1, R2, R3, R4 | Spec 13, this document's R1-R4 plus `docs/design/P21-L1.md` (plan: `docs/plans/2026-10-09-feature-roadmap.md`) | L1 design; L2 notifications API; L3 upcoming, calendar feed, campus time API; L4 web bell and notifications page; L5 web Upcoming panel, calendar dialog, formatters; L6 e2e | §6 Q1, Q8, Q9, Q12 (defaults given) |
| 22 | Run the next term | R5, R6, R7 | Spec 14 "term operations" | L1 design; L2 tenant settings, resend and single enrollment API; L3 section rollover (`course`); L4 content, quiz and assignment copy; L5 exports; L6 web admin (settings tab, resend, add student, rollover dialog, exports); L7 web copy-from-section dialog; L8 e2e | §6 Q13 (what a copy carries) |
| 23 | Section conversation and content | R8, R17 | Spec 15 | L1 design; L2 discussions API and package; L3 content depth API (links, `visible_from`, announcement files); L4 web Discussions tab; L5 web content forms; L6 e2e | §6 Q7 (plain text default) |
| 24 | Assessment depth | R9, R10, R11 (several attempts and essay; banks deferred) | Spec 16 | L1 design; L2 rubrics; L3 extensions (deadline SQL, sweeper); L4 quiz attempts and essay; L5 web assignment and rubric grading; L6 web quiz; L7 e2e | none |
| 25 | Attendance, early alert, accessibility | R12, R13, R19 | Spec 17 | L1 design; L2 attendance API; L3 needs-attention API; L4 web register and needs-attention list; L5 WCAG 2.2 audit, fixes in `components/` and ACR draft (`docs/accessibility/`); L6 e2e | §6 Q10, Q11 |

Phases 23-25 can swap without breaking dependencies. Phase 21 should come first, because its notifications are the channel later phases publish into: discussion replies and rubric releases. Phase 22 should come before the first pilot's second term. Next in line after Phase 25: R18, R20 (with an icon), R14, R21-R24, then the owner-gated O items as they are decided.

## 5. Non-goals for the next six months

| Not building | Reason |
|---|---|
| Proctoring, lockdown browsers, AI-writing detection (X56, X22 part) | Universities are switching detection off [S49][S50], and guidance favours process and feedback [S51]; R9 rubrics serve that direction instead |
| Plagiarism checking | A partner integration (Turnitin is LTI or a plugin everywhere [S29][S40]); waits on O3 |
| Video hosting, transcoding, captions, live classroom | Add-on territory in every product [S30][S32][S46]; R17 links cover embedding a hosted video |
| Predictive or ML analytics, data warehouse feeds | Gated, paid-tier features [S33][S47][S48]; R13's rules claim the trend at a fraction of the cost |
| Native mobile apps | O4; R20 plus the responsive web first |
| Any AI feature before O1 | Provider, data terms and EU AI Act duties [S59] are owner decisions; usage is thinner than the marketing [S39] |
| LTI Advantage certification, an app marketplace | [S21][S47]; consumer support only when a partner names its tools (O3) |
| Digital credentials (Open Badges 3.0) | Trend 15 has no adoption figures [S65] |
| Self-service instructor or university sign-up | The Canvas intrusion path [S60]; universities stay CLI-created (D28, D128) |
| Custom roles, cross-listing, section hierarchy, rooms (X10-X12, X17) | Triggers need a second university or a never-written scheduling spec (L2 §2 note) |
| Platform admin UI, regional deployments, audit partitioning (X8, X14, X15) | Fewer than 10 universities; no EU customer; no volume |
| Translations (X43) | Locale formatting comes in R4; copy translation waits for a non-English customer |
| Account deletion | Conflicts with the append-only audit and grade retention; needs the owner's retention policy (see R18) |
| OpenSpec, Postman chaining, visual snapshots (X59-X61) | Tooling with no user; the owner asks (D121) |

## 6. Questions for the owner

Each has a recommended answer, so an unattended run can proceed under D8. The orchestrator records the answers as decisions from D157 on.

1. **Framing and first phase.** Approve "the term loop" (§0) and start Phase 21 "Stay on top"? *Recommended: yes.*
2. **AI in the next six months.** *Recommended: no AI feature through Phase 25.* When it comes:
   - instructor-side drafting first (quiz items, rubric drafts), behind a per-university switch;
   - called over plain `net/http` (no SDK, D1);
   - never grading students (EU AI Act Annex III [S59]).
3. **Deployment target.** *Recommended: pick one before Phase 22 closes* (a single container host plus managed Postgres, one region). It unblocks K3, K4, push and the first pilot. Until then, feature phases continue unaffected.
4. **Mobile strategy and icon.** *Recommended: a PWA, not native apps, for six months.* Web push follows a deployment target. For the icon, an accent-purple "U" monogram generated as SVG until a logo exists.
5. **Interop order.** *Recommended: exports first (R7, Phase 22).* OneRoster or SIS sync and an LTI 1.3 consumer only when a signed university names its SIS and tools. No API tokens before then.
6. **Billing.** *Recommended: none until the first paying university; invoice by contract.*
7. **Rich text.** *Recommended: plain text with autolinked URLs and kept line breaks for Phases 23-24* (built from React elements, so no HTML injection). A Markdown subset written in-house only if instructors ask; no new dependency.
8. **Notification mail.** *Recommended:* in-app for all three kinds. Mail stays as today: announcement mail is still chosen per post, with a new per-member opt-out. No digest and no mail for grades or new work in Phase 21.
9. **Calendar feed security.** *Recommended:* one secret token per member per university, in the URL, read-only, rotatable and revoked when the membership ends. The token never appears in logs.
10. **Attendance and grades.** *Recommended:* a register outside the gradebook first (four statuses, per meeting date). A weighted gradebook column only if instructors ask (L1 §6 Q4).
11. **Accessibility commitment.** *Recommended:* target WCAG 2.2 AA and publish the self-audit ACR in Phase 25. Commission a third-party audit when the first university's procurement asks; that audit is an owner spend.
12. **Where Upcoming lives.** *Recommended:* a panel at the top of My sections. `/{slug}` keeps landing on `/me`, so the e2e landing assertions stay; no separate dashboard page.
13. **What a copy carries (Phase 22).** *Recommended:*
    - Rollover copies sections, meetings and staff. Copy-from-section copies weeks, notes, materials (bytes), quizzes (questions and keys) and assignments.
    - Dates shift by the term-start difference, and everything arrives unpublished.
    - Never copied: announcements, students, attempts, submissions, scores.
14. **Self-enrollment.** *Recommended: no in six months.* Rosters stay import-driven (D38), plus R5's single add.
15. **SAML.** *Recommended: not before a university needs it.* At that point, a dependency decision for an XML-DSig library under D1.

## 7. Sources

P20-L1's numbering (its §7 has URLs and dates). Cited here: S1, S2, S3, S6, S8, S11, S13, S14, S15, S16, S17, S19, S20, S21, S22, S23, S24, S26, S29, S30, S31, S32, S33, S34, S35, S36, S38, S39, S40, S42, S43, S44, S45, S46, S47, S48, S49, S50, S51, S53, S54, S56, S58, S59, S60, S64, S65, S68, S69. Persona dead ends and cost signals: `docs/design/P20-L2.md` §5 and §6, by id.
