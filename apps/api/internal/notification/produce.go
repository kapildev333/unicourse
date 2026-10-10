// Package notification holds the in-app notifications (Spec 13 R2): producers called inside the
// transaction of the event, and the member's own list, counts and settings.
package notification

import (
	"context"
	"database/sql"
)

// Each producer inserts one row per recipient with a single insert-select inside the caller's transaction.
// Recipients are active members other than the actor, so the author never notifies themself.

const insertHead = `insert into notifications (university_id, user_id, kind, section_id, `

// Announcement notifies the section's enrolled students and staff.
func Announcement(ctx context.Context, tx *sql.Tx, universityID, actorID, sectionID, announcementID string) error {
	_, err := tx.ExecContext(ctx, insertHead+`announcement_id)
		select $1, m.user_id, 'announcement', $3, $4 from memberships m
		where m.status = 'active' and m.user_id <> $2
		  and m.user_id in (select user_id from enrollments where section_id = $3 union select user_id from section_staff where section_id = $3)`,
		universityID, actorID, sectionID, announcementID)
	return err
}

// AssessmentCreated notifies the section's enrolled students of a new quiz or assignment.
func AssessmentCreated(ctx context.Context, tx *sql.Tx, universityID, actorID, sectionID, assessmentID string) error {
	_, err := tx.ExecContext(ctx, insertHead+`assessment_id)
		select $1, m.user_id, 'assessment_created', $3, $4 from memberships m
		where m.status = 'active' and m.user_id <> $2
		  and m.user_id in (select user_id from enrollments where section_id = $3)`,
		universityID, actorID, sectionID, assessmentID)
	return err
}

// ScoresPublished notifies the enrolled students once per assessment, however often it is toggled.
func ScoresPublished(ctx context.Context, tx *sql.Tx, universityID, actorID, sectionID, assessmentID string) error {
	_, err := tx.ExecContext(ctx, insertHead+`assessment_id)
		select $1, m.user_id, 'scores_published', $3, $4 from memberships m
		where m.status = 'active' and m.user_id <> $2
		  and m.user_id in (select user_id from enrollments where section_id = $3)
		  and not exists (select 1 from notifications n where n.assessment_id = $4 and n.kind = 'scores_published' and n.user_id = m.user_id)`,
		universityID, actorID, sectionID, assessmentID)
	return err
}

// GradesFinalized notifies every holder of a final grade of the section (sourceID is unused: pass "").
func GradesFinalized(ctx context.Context, tx *sql.Tx, universityID, actorID, sectionID, _ string) error {
	_, err := tx.ExecContext(ctx, insertHead+`assessment_id)
		select $1, m.user_id, 'grades_finalized', $3, null from memberships m
		where m.status = 'active' and m.user_id <> $2
		  and m.user_id in (select user_id from final_grades where section_id = $3)`,
		universityID, actorID, sectionID)
	return err
}
