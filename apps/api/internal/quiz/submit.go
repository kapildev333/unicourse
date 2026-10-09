package quiz

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/kapildev333/unicourse/apps/api/internal/audit"
	"github.com/kapildev333/unicourse/apps/api/internal/grade"
)

var errAlreadySubmitted = errors.New("already submitted")

// keyedQuestions reads the questions with their keys for autoGrade.
func keyedQuestions(ctx context.Context, tx *sql.Tx, assessmentID string) ([]Question, error) {
	rows, err := tx.QueryContext(ctx, `select id, kind, points::text, choices, answers from questions where assessment_id = $1 order by position`, assessmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var qs []Question
	for rows.Next() {
		var q Question
		var choices, answers []byte
		if err := rows.Scan(&q.ID, &q.Kind, &q.Points, &choices, &answers); err != nil {
			return nil, err
		}
		as := []string{}
		if err := json.Unmarshal(choices, &q.Choices); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(answers, &as); err != nil {
			return nil, err
		}
		q.Answers = &as
		qs = append(qs, q)
	}
	return qs, rows.Err()
}

// submit grades the stored answers of (assessmentID, userID), sets grading, score, submitted_at (now() for a
// student, the effective deadline when auto) and auto_submitted, upserts the scores row (graded_by = the student)
// and writes audit attempt.submit (actor: the student; after {userId, score, auto}). The caller holds the
// attempt row lock. It returns errAlreadySubmitted when the attempt is missing or already submitted.
func submit(ctx context.Context, tx *sql.Tx, universityID, assessmentID, userID string, auto bool) error {
	var raw []byte
	err := tx.QueryRowContext(ctx, `select answers from attempts where assessment_id = $1 and user_id = $2 and submitted_at is null`, assessmentID, userID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return errAlreadySubmitted
	}
	if err != nil {
		return err
	}
	answers := map[string]Answer{}
	if err := json.Unmarshal(raw, &answers); err != nil {
		return err
	}
	qs, err := keyedQuestions(ctx, tx, assessmentID)
	if err != nil {
		return err
	}
	marks, total := autoGrade(qs, answers)
	grading, err := json.Marshal(marks)
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `update attempts a set grading = $3::jsonb, score = $4::numeric, auto_submitted = $5::boolean,
		submitted_at = case when $5::boolean then (select `+deadlineSQL+` from assessments q where q.id = a.assessment_id) else now() end
		where a.assessment_id = $1 and a.user_id = $2 and a.submitted_at is null`, assessmentID, userID, string(grading), total, auto)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return errAlreadySubmitted
	}
	if err := grade.UpsertScore(ctx, tx, universityID, assessmentID, userID, total, userID); err != nil {
		return err
	}
	return audit.Write(ctx, tx, universityID, audit.Entry{ActorUserID: userID, Action: "attempt.submit", EntityType: "assessment", EntityID: assessmentID,
		After: map[string]any{"userId": userID, "score": json.Number(total), "auto": auto}})
}
