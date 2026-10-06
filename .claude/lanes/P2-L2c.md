# P2-L2c - CSV imports (courses, sections, enrollments), tenant.CreateInvitations
Status: done
## Doing
Gates run; waiting for review.
## Found
- Routes (POST, under `/api/v1/universities/{slug}`): `/courses/import[/preview]`, `/sections/import[/preview]`, `/sections/{id}/enrollments/import[/preview]`. Multipart fields `file` (+ `checksum` on commit, the checksum returned by preview). Commit returns 200 in the preview shape.
- Columns (headers case-insensitive): courses `departmentCode, code, title`; sections `courseCode, termName, number`, optional `instructorEmails, taEmails` (`;` separated); enrollments `email`, optional `displayName` (ignored).
- Row messages are verbatim from design 8 (e.g. `invalid email`, `domain not allowed`, `membership is not active`, `duplicate of line 2`, `unknown department code X`, `unknown course code X`, `unknown term X`).
- Roster example: `email,displayName\nstudent1@a.edu,Student One\n` (a.edu must be in the university's allowed domains for non-members; `invite` rows get an invitation + pending_enrollments, applied by auth.SignIn).
- Unique constraints lead with `university_id`, so `on conflict (university_id, section_id, user_id)` is needed (design text omits it).
- Design nuance: a sections row for an existing section whose listed staff are all already on it is `exists`; a row that adds staff or invitations to an existing section is `add`/`invite` (design 8.3), so staff of an existing section can grow, never change role.
- `tenant.CreateInvitations` + `invitationMail` added; `pending_routes.txt` deleted, contract test passes with zero pending (diff helper kept).
- 5,000-row roster commit (5,000 invitations + mails) runs in under 1 s locally.
## Requests
None.
## Blockers
None.
