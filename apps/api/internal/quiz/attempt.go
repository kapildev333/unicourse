package quiz

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kapildev333/unicourse/apps/api/internal/audit"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
)

// Attempt is one student's attempt. Grading, Score and Feedback are nil/empty while hidden or not submitted (never null).
type Attempt struct {
	UserID        string            `json:"userId"`
	StartedAt     time.Time         `json:"startedAt"`
	DeadlineAt    time.Time         `json:"deadlineAt"`
	SubmittedAt   *time.Time        `json:"submittedAt,omitempty"`
	AutoSubmitted bool              `json:"autoSubmitted"`
	Answers       map[string]Answer `json:"answers"`
	Grading       *map[string]Mark  `json:"grading,omitempty"`
	Score         json.Number       `json:"score,omitempty"`
	Feedback      *string           `json:"feedback,omitempty"`
	ReviewedAt    *time.Time        `json:"reviewedAt,omitempty"`
}

// hideResults is the student projection before release.
func (a *Attempt) hideResults() {
	a.Grading, a.Feedback, a.Score = nil, nil, ""
}

// loadAttempt reads the attempt of user; lock is "" or "for update" (a constant). ok is false when there is none.
// Results are filled only for a submitted attempt.
func loadAttempt(r *http.Request, tx *sql.Tx, assessmentID, userID, lock string) (a Attempt, ok bool, err error) {
	var answers, grading []byte
	var submitted, reviewed sql.NullTime
	var score sql.NullString
	var feedback string
	if lock != "" {
		lock += " of a"
	}
	err = tx.QueryRowContext(r.Context(), `select a.started_at, `+deadlineSQL+`, a.submitted_at, a.auto_submitted, a.answers, a.grading, a.score::text, a.feedback, a.reviewed_at
		from attempts a join assessments q on q.id = a.assessment_id where a.assessment_id = $1 and a.user_id = $2 `+lock, assessmentID, userID).
		Scan(&a.StartedAt, &a.DeadlineAt, &submitted, &a.AutoSubmitted, &answers, &grading, &score, &feedback, &reviewed)
	if errors.Is(err, sql.ErrNoRows) {
		return a, false, nil
	}
	if err != nil {
		return a, false, err
	}
	a.UserID = userID
	a.StartedAt, a.DeadlineAt = a.StartedAt.UTC(), a.DeadlineAt.UTC()
	a.Answers = map[string]Answer{}
	if err := json.Unmarshal(answers, &a.Answers); err != nil {
		return a, false, err
	}
	if submitted.Valid {
		t := submitted.Time.UTC()
		a.SubmittedAt = &t
		g := map[string]Mark{}
		if err := json.Unmarshal(grading, &g); err != nil {
			return a, false, err
		}
		a.Grading, a.Score, a.Feedback = &g, json.Number(score.String), &feedback
	}
	if reviewed.Valid {
		t := reviewed.Time.UTC()
		a.ReviewedAt = &t
	}
	return a, true, nil
}

func startAttempt(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	secID := r.PathValue("id")
	if err := sectionExists(r, tx, secID); err != nil {
		return 0, nil, err
	}
	w, err := loadWork(r, tx, secID, r.PathValue("assessmentId"), "quiz", "for share")
	if err != nil {
		return 0, nil, err
	}
	ok, err := enrolled(r, tx, secID, s.UserID)
	if err := allow(ok, err); err != nil {
		return 0, nil, err
	}
	if w.Upcoming {
		return 0, nil, httpx.Conflict("not open yet")
	}
	if w.Closed {
		return 0, nil, httpx.Conflict("closed")
	}
	var n int
	if err := tx.QueryRowContext(r.Context(), `select count(*) from questions where assessment_id = $1`, w.ID).Scan(&n); err != nil {
		return 0, nil, err
	}
	if n == 0 {
		return 0, nil, httpx.Conflict("the quiz has no questions")
	}
	res, err := tx.ExecContext(r.Context(), `insert into attempts (university_id, assessment_id, user_id) values ($1, $2, $3)
		on conflict (university_id, assessment_id, user_id) do nothing`, s.UniversityID, w.ID, s.UserID)
	if err != nil {
		return 0, nil, err
	}
	created, err := res.RowsAffected()
	if err != nil {
		return 0, nil, err
	}
	status := http.StatusOK
	if created == 1 {
		status = http.StatusCreated
		if err := audit.Write(r.Context(), tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "attempt.start", EntityType: "assessment", EntityID: w.ID,
			After: map[string]string{"userId": s.UserID}}); err != nil {
			return 0, nil, err
		}
	}
	a, _, err := loadAttempt(r, tx, w.ID, s.UserID, "")
	if err != nil {
		return 0, nil, err
	}
	if !w.Published {
		a.hideResults()
	}
	return status, a, nil
}

func getAttempt(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
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
	a, ok, err := loadAttempt(r, tx, w.ID, userID, "")
	if err != nil {
		return 0, nil, err
	}
	if !ok {
		return 0, nil, httpx.NotFound("attempt")
	}
	if !staff && !w.Published {
		a.hideResults()
	}
	return http.StatusOK, a, nil
}

// checkAnswers validates the answers against the quiz's questions (keys sorted, so the errors are stable) and drops empty entries.
func checkAnswers(qs []Question, in map[string]Answer) (map[string]Answer, []httpx.FieldError) {
	byID := make(map[string]Question, len(qs))
	for _, q := range qs {
		byID[q.ID] = q
	}
	keys := make([]string, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := map[string]Answer{}
	var d []httpx.FieldError
	for _, k := range keys {
		a, p := in[k], "answers."+k
		q, ok := byID[k]
		if !ok {
			d = append(d, httpx.FieldError{Field: p, Message: "is not a question of this quiz"})
			continue
		}
		bad := false
		if q.Kind == "choice" {
			seen := map[int]bool{}
			for _, i := range a.Choices {
				if i < 0 || i >= len(q.Choices) || seen[i] {
					d = append(d, httpx.FieldError{Field: p + ".choices", Message: "must be distinct indexes from 0 to " + strconv.Itoa(len(q.Choices)-1)})
					bad = true
					break
				}
				seen[i] = true
			}
			if a.Text != "" {
				d = append(d, httpx.FieldError{Field: p + ".text", Message: "only for fill in the blank"})
				bad = true
			}
		} else {
			if len(a.Choices) > 0 {
				d = append(d, httpx.FieldError{Field: p + ".choices", Message: "only for multiple choice"})
				bad = true
			}
			if strings.ContainsRune(a.Text, 0) {
				d = append(d, httpx.FieldError{Field: p + ".text", Message: "must not contain NUL"})
				bad = true
			} else if utf8.RuneCountInString(a.Text) > 1000 {
				d = append(d, httpx.FieldError{Field: p + ".text", Message: "must be at most 1000 characters"})
				bad = true
			}
		}
		if !bad && (len(a.Choices) > 0 || a.Text != "") {
			out[k] = a
		}
	}
	return out, d
}

// pastDeadline reports whether clock (the constant "now()" or "clock_timestamp()") is at or past the attempt's effective deadline.
func pastDeadline(r *http.Request, tx *sql.Tx, assessmentID, userID, clock string) (past bool, err error) {
	err = tx.QueryRowContext(r.Context(), `select `+clock+` >= `+deadlineSQL+` from attempts a join assessments q on q.id = a.assessment_id
		where a.assessment_id = $1 and a.user_id = $2`, assessmentID, userID).Scan(&past)
	return past, err
}

func putAttempt(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	secID := r.PathValue("id")
	if err := sectionExists(r, tx, secID); err != nil {
		return 0, nil, err
	}
	w, err := loadWork(r, tx, secID, r.PathValue("assessmentId"), "quiz", "")
	if err != nil {
		return 0, nil, err
	}
	ok, err := enrolled(r, tx, secID, s.UserID)
	if err := allow(ok, err); err != nil {
		return 0, nil, err
	}
	a, ok, err := loadAttempt(r, tx, w.ID, s.UserID, "for update")
	if err != nil {
		return 0, nil, err
	}
	if !ok {
		return 0, nil, httpx.NotFound("attempt")
	}
	if a.SubmittedAt != nil {
		return 0, nil, httpx.Conflict("already submitted")
	}
	if past, err := pastDeadline(r, tx, w.ID, s.UserID, "now()"); err != nil {
		return 0, nil, err
	} else if past {
		return 0, nil, httpx.Conflict("closed; your answers as of the closing time were submitted")
	}
	var in struct {
		Answers *map[string]Answer `json:"answers"`
		Submit  bool               `json:"submit"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	// The body may have taken a while: the deadline is authoritative against the live clock, not the transaction's now().
	if late, err := pastDeadline(r, tx, w.ID, s.UserID, "clock_timestamp()"); err != nil {
		return 0, nil, err
	} else if late {
		return 0, nil, httpx.Conflict("closed; your answers as of the closing time were submitted")
	}
	if in.Answers == nil {
		return 0, nil, httpx.Validation(httpx.FieldError{Field: "answers", Message: "is required"})
	}
	qs, err := keyedQuestions(r.Context(), tx, w.ID)
	if err != nil {
		return 0, nil, err
	}
	answers, d := checkAnswers(qs, *in.Answers)
	if len(d) > 0 {
		return 0, nil, httpx.Validation(d...)
	}
	raw, err := json.Marshal(answers)
	if err != nil {
		return 0, nil, err
	}
	if _, err := tx.ExecContext(r.Context(), `update attempts set answers = $3::jsonb where assessment_id = $1 and user_id = $2`, w.ID, s.UserID, string(raw)); err != nil {
		return 0, nil, err
	}
	if in.Submit {
		if err := submit(r.Context(), tx, s.UniversityID, w.ID, s.UserID, false); err != nil {
			if errors.Is(err, errAlreadySubmitted) {
				return 0, nil, httpx.Conflict("already submitted")
			}
			return 0, nil, err
		}
	}
	a, _, err = loadAttempt(r, tx, w.ID, s.UserID, "")
	if err != nil {
		return 0, nil, err
	}
	if !w.Published {
		a.hideResults()
	}
	return http.StatusOK, a, nil
}
