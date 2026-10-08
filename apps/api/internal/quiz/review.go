package quiz

import (
	"database/sql"
	"encoding/json"
	"math/big"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kapildev333/unicourse/apps/api/internal/audit"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/grade"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

// AttemptRow is one line of the staff attempts list.
type AttemptRow struct {
	UserID        string      `json:"userId"`
	Email         string      `json:"email"`
	DisplayName   string      `json:"displayName"`
	StartedAt     time.Time   `json:"startedAt"`
	SubmittedAt   *time.Time  `json:"submittedAt,omitempty"`
	AutoSubmitted bool        `json:"autoSubmitted"`
	Score         json.Number `json:"score,omitempty"`
	ReviewedAt    *time.Time  `json:"reviewedAt,omitempty"`
}

func listAttempts(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	secID := r.PathValue("id")
	if err := sectionExists(r, tx, secID); err != nil {
		return 0, nil, err
	}
	w, err := loadWork(r, tx, secID, r.PathValue("assessmentId"), "quiz", "")
	if err != nil {
		return 0, nil, err
	}
	if err := allow(editor(r, tx, s, secID)); err != nil {
		return 0, nil, err
	}
	rows, err := tx.QueryContext(r.Context(), `select a.user_id, u.email, u.display_name, a.started_at, a.submitted_at, a.auto_submitted, a.score::text, a.reviewed_at
		from attempts a join enrollments e on e.section_id = $2 and e.user_id = a.user_id join users u on u.id = a.user_id
		where a.assessment_id = $1 order by coalesce(nullif(u.display_name, ''), u.email), u.email`, w.ID, secID)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	items := []AttemptRow{}
	for rows.Next() {
		var x AttemptRow
		var sub, rev sql.NullTime
		var score sql.NullString
		if err := rows.Scan(&x.UserID, &x.Email, &x.DisplayName, &x.StartedAt, &sub, &x.AutoSubmitted, &score, &rev); err != nil {
			return 0, nil, err
		}
		x.StartedAt = x.StartedAt.UTC()
		if sub.Valid {
			t := sub.Time.UTC()
			x.SubmittedAt = &t
			x.Score = json.Number(score.String)
		}
		if rev.Valid {
			t := rev.Time.UTC()
			x.ReviewedAt = &t
		}
		items = append(items, x)
	}
	return http.StatusOK, struct {
		Items []AttemptRow `json:"items"`
	}{items}, rows.Err()
}

// optText validates an optional free text (trimmed, line breaks kept, no NUL, at most max runes); empty is allowed.
func optText(field, v string, max int) (string, *httpx.FieldError) {
	v = strings.TrimSpace(v)
	if strings.ContainsRune(v, 0) {
		return v, &httpx.FieldError{Field: field, Message: "must not contain NUL"}
	}
	if utf8.RuneCountInString(v) > max {
		return v, &httpx.FieldError{Field: field, Message: "must be at most " + strconv.Itoa(max) + " characters"}
	}
	return v, nil
}

type markIn struct {
	Points   json.Number `json:"points"`
	Feedback *string     `json:"feedback"`
}

func reviewAttempt(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	secID := r.PathValue("id")
	if err := sectionExists(r, tx, secID); err != nil {
		return 0, nil, err
	}
	w, err := loadWork(r, tx, secID, r.PathValue("assessmentId"), "quiz", "")
	if err != nil {
		return 0, nil, err
	}
	userID, staff, err := target(r, tx, s, secID)
	if err != nil {
		return 0, nil, err
	}
	if !staff { // target lets a student name themselves
		return 0, nil, httpx.Forbidden()
	}
	a, ok, err := loadAttempt(r, tx, w.ID, userID, "for update")
	if err != nil {
		return 0, nil, err
	}
	if !ok {
		return 0, nil, httpx.NotFound("attempt")
	}
	if a.SubmittedAt == nil {
		return 0, nil, httpx.Conflict("the attempt is not submitted")
	}
	var in struct {
		Grading  tenant.Opt[map[string]markIn] `json:"grading"`
		Feedback tenant.Opt[string]            `json:"feedback"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	if !in.Grading.Set && !in.Feedback.Set {
		return 0, nil, tenant.NoFields()
	}
	qs, err := keyedQuestions(r.Context(), tx, w.ID)
	if err != nil {
		return 0, nil, err
	}
	byID := make(map[string]Question, len(qs))
	for _, q := range qs {
		byID[q.ID] = q
	}
	marks := *a.Grading
	var d []httpx.FieldError
	if in.Grading.Null {
		d = append(d, httpx.FieldError{Field: "grading", Message: "must not be null"})
	}
	ids := make([]string, 0, len(in.Grading.V))
	for id := range in.Grading.V {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		m, p := in.Grading.V[id], "grading."+id
		q, ok := byID[id]
		if !ok {
			d = append(d, httpx.FieldError{Field: p, Message: "is not a question of this quiz"})
			continue
		}
		mark := marks[id]
		pts, fe := tenant.Decimal(p+".points", m.Points, 2, "0", string(q.Points))
		if fe != nil {
			d = append(d, *fe)
		} else {
			got, _ := new(big.Rat).SetString(pts)
			full, _ := new(big.Rat).SetString(string(q.Points))
			mark.Points, mark.Correct = json.Number(pts), got.Cmp(full) == 0
		}
		if m.Feedback != nil {
			f, fe := optText(p+".feedback", *m.Feedback, 2000)
			if fe != nil {
				d = append(d, *fe)
			}
			mark.Feedback = f
		}
		marks[id] = mark
	}
	feedback := *a.Feedback
	if in.Feedback.Null {
		d = append(d, httpx.FieldError{Field: "feedback", Message: "must not be null"})
	} else if in.Feedback.Set {
		f, fe := optText("feedback", in.Feedback.V, 5000)
		if fe != nil {
			d = append(d, *fe)
		}
		feedback = f
	}
	if len(d) > 0 {
		return 0, nil, httpx.Validation(d...)
	}
	sum := new(big.Rat)
	for _, q := range qs {
		if p, ok := new(big.Rat).SetString(string(marks[q.ID].Points)); ok {
			sum.Add(sum, p)
		}
	}
	score := sum.FloatString(2)
	raw, err := json.Marshal(marks)
	if err != nil {
		return 0, nil, err
	}
	if _, err := tx.ExecContext(r.Context(), `update attempts set grading = $3::jsonb, score = $4::numeric, feedback = $5, reviewed_by = $6, reviewed_at = now()
		where assessment_id = $1 and user_id = $2`, w.ID, userID, string(raw), score, feedback, s.UserID); err != nil {
		return 0, nil, err
	}
	if err := grade.UpsertScore(r.Context(), tx, s.UniversityID, w.ID, userID, score, s.UserID); err != nil {
		return 0, nil, err
	}
	if err := audit.Write(r.Context(), tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "attempt.review", EntityType: "assessment", EntityID: w.ID,
		Before: map[string]any{"userId": userID, "score": a.Score}, After: map[string]any{"userId": userID, "score": json.Number(score)}}); err != nil {
		return 0, nil, err
	}
	out, _, err := loadAttempt(r, tx, w.ID, userID, "")
	return http.StatusOK, out, err
}
