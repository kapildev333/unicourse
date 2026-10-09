package quiz

import (
	"context"
	"database/sql"
	"encoding/json"
	"reflect"

	"github.com/kapildev333/unicourse/apps/api/internal/audit"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/grade"
)

// sameStructure reports whether items may replace existing in place: the same count, and at every position the
// same question id, kind and (for choice) number of choices.
func sameStructure(existing []Question, items []questionIn) bool {
	if len(existing) != len(items) {
		return false
	}
	for i, q := range existing {
		it := items[i]
		if it.ID != q.ID || it.Kind != q.Kind || (q.Kind == "choice" && len(it.Choices) != len(q.Choices)) {
			return false
		}
	}
	return true
}

// regrade applies rows (questions with their ids, in position order) in place and regrades every submitted attempt (D149).
// The caller holds the assessment row lock and has checked sameStructure.
func regrade(ctx context.Context, tx *sql.Tx, s authz.Scope, assessmentID string, rows []byte) error {
	// every attempt first: an in-flight submit or sweeper claim finishes before the keys change
	if _, err := tx.ExecContext(ctx, `select 1 from attempts where assessment_id = $1 order by id for update`, assessmentID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `update questions q set prompt = x.v->>'prompt', points = (x.v->>'points')::numeric, choices = x.v->'choices', answers = x.v->'answers'
		from jsonb_array_elements($2::jsonb) as x(v) where q.assessment_id = $1 and q.id = (x.v->>'id')::uuid`, assessmentID, string(rows)); err != nil {
		return err
	}
	qs, err := keyedQuestions(ctx, tx, assessmentID)
	if err != nil {
		return err
	}
	type att struct {
		userID, score string
		answers, old  []byte
	}
	var atts []att
	res, err := tx.QueryContext(ctx, `select user_id::text, answers, grading, score::text from attempts where assessment_id = $1 and submitted_at is not null order by user_id`, assessmentID)
	if err != nil {
		return err
	}
	for res.Next() {
		var a att
		if err := res.Scan(&a.userID, &a.answers, &a.old, &a.score); err != nil {
			res.Close()
			return err
		}
		atts = append(atts, a)
	}
	res.Close()
	if err := res.Err(); err != nil {
		return err
	}
	for _, a := range atts {
		answers, old := map[string]Answer{}, map[string]Mark{}
		if err := json.Unmarshal(a.answers, &answers); err != nil {
			return err
		}
		if err := json.Unmarshal(a.old, &old); err != nil {
			return err
		}
		marks, total := regradeMarks(qs, answers, old)
		if reflect.DeepEqual(marks, old) {
			continue
		}
		raw, err := json.Marshal(marks)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `update attempts set grading = $3::jsonb, score = $4::numeric where assessment_id = $1 and user_id = $2`, assessmentID, a.userID, string(raw), total); err != nil {
			return err
		}
		if err := grade.UpsertScore(ctx, tx, s.UniversityID, assessmentID, a.userID, total, s.UserID); err != nil {
			return err
		}
		if err := audit.Write(ctx, tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "attempt.regrade", EntityType: "assessment", EntityID: assessmentID,
			Before: map[string]any{"userId": a.userID, "score": json.Number(a.score)}, After: map[string]any{"userId": a.userID, "score": json.Number(total)}}); err != nil {
			return err
		}
	}
	return nil
}
