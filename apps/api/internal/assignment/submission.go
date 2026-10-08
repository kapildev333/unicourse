package assignment

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kapildev333/unicourse/apps/api/internal/audit"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
)

type File struct {
	ID          string    `json:"id"`
	FileName    string    `json:"fileName"`
	ContentType string    `json:"contentType"`
	SizeBytes   int64     `json:"sizeBytes"`
	Position    int       `json:"position"`
	UploadedAt  time.Time `json:"uploadedAt"`
	sha256      string
}

type Submission struct {
	UserID      string      `json:"userId"`
	Body        string      `json:"body"`
	SubmittedAt time.Time   `json:"submittedAt"`
	Files       []File      `json:"files"` // never nil
	Score       json.Number `json:"score,omitempty"`
	Feedback    *string     `json:"feedback,omitempty"`
	GradedAt    *time.Time  `json:"gradedAt,omitempty"`
	id          string
}

// hideGrade is the student projection of an unpublished assignment.
func (s *Submission) hideGrade() { s.Score, s.Feedback, s.GradedAt = "", nil, nil }

type SubmissionRow struct {
	UserID      string      `json:"userId"`
	Email       string      `json:"email"`
	DisplayName string      `json:"displayName"`
	SubmittedAt time.Time   `json:"submittedAt"`
	FileCount   int         `json:"fileCount"`
	Score       json.Number `json:"score,omitempty"`
	GradedAt    *time.Time  `json:"gradedAt,omitempty"`
}

const (
	maxBody  = 20000
	maxFiles = 10
)

// loadSubmission reads the submission of (assessment, user) with its files; none -> 404 "submission".
// lock is "" or "for update". The staff projection (graded fields, feedback always present) is returned.
func loadSubmission(ctx context.Context, tx *sql.Tx, assessmentID, userID, lock string) (Submission, error) {
	var s Submission
	var fb string
	var graded sql.NullTime
	err := tx.QueryRowContext(ctx, `select id, user_id, body, submitted_at, coalesce(score::text, ''), feedback, graded_at
		from submissions where assessment_id = $1 and user_id = $2 `+lock, assessmentID, userID).
		Scan(&s.id, &s.UserID, &s.Body, &s.SubmittedAt, &s.Score, &fb, &graded)
	if errors.Is(err, sql.ErrNoRows) {
		return s, httpx.NotFound("submission")
	}
	if err != nil {
		return s, err
	}
	s.SubmittedAt = s.SubmittedAt.UTC()
	s.Feedback = &fb
	if graded.Valid {
		t := graded.Time.UTC()
		s.GradedAt = &t
	}
	s.Files = []File{}
	rows, err := tx.QueryContext(ctx, `select id, file_name, content_type, size_bytes, position, uploaded_at, sha256
		from submission_files where submission_id = $1 order by position`, s.id)
	if err != nil {
		return s, err
	}
	defer rows.Close()
	for rows.Next() {
		var f File
		if err := rows.Scan(&f.ID, &f.FileName, &f.ContentType, &f.SizeBytes, &f.Position, &f.UploadedAt, &f.sha256); err != nil {
			return s, err
		}
		f.UploadedAt = f.UploadedAt.UTC()
		s.Files = append(s.Files, f)
	}
	return s, rows.Err()
}

// submissionAudit is the `after` of submission.update.
type submissionAudit struct {
	UserID     string   `json:"userId"`
	BodyLength int      `json:"bodyLength"`
	FileNames  []string `json:"fileNames"`
}

// auditUpdate writes submission.update for the caller's current submission state.
func auditUpdate(ctx context.Context, tx *sql.Tx, s authz.Scope, assessmentID string) error {
	sub, err := loadSubmission(ctx, tx, assessmentID, s.UserID, "")
	a := submissionAudit{UserID: s.UserID, FileNames: []string{}}
	if err == nil {
		a.BodyLength = utf8.RuneCountInString(sub.Body)
		for _, f := range sub.Files {
			a.FileNames = append(a.FileNames, f.FileName)
		}
	} else if he := new(httpx.Error); !errors.As(err, &he) || he.Status != http.StatusNotFound {
		return err
	}
	return audit.Write(ctx, tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "submission.update", EntityType: "assessment", EntityID: assessmentID, After: a})
}

// own is the student view: the grade only once the assignment is published.
func own(sub Submission, w work) Submission {
	if !w.Published {
		sub.hideGrade()
	}
	return sub
}

func listSubmissions(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	secID := r.PathValue("id")
	if err := sectionExists(r, tx, secID); err != nil {
		return 0, nil, err
	}
	w, err := loadWork(r, tx, secID, r.PathValue("assessmentId"), "")
	if err != nil {
		return 0, nil, err
	}
	if err := editor(r, tx, s, secID); err != nil {
		return 0, nil, err
	}
	rows, err := tx.QueryContext(r.Context(), `select s.user_id, u.email, u.display_name, s.submitted_at,
		(select count(*) from submission_files f where f.submission_id = s.id)::int, coalesce(s.score::text, ''), s.graded_at
		from submissions s join enrollments e on e.section_id = $2 and e.user_id = s.user_id join users u on u.id = s.user_id
		where s.assessment_id = $1 order by coalesce(nullif(u.display_name, ''), u.email), u.email`, w.ID, secID)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	items := []SubmissionRow{}
	for rows.Next() {
		var x SubmissionRow
		var graded sql.NullTime
		if err := rows.Scan(&x.UserID, &x.Email, &x.DisplayName, &x.SubmittedAt, &x.FileCount, &x.Score, &graded); err != nil {
			return 0, nil, err
		}
		x.SubmittedAt = x.SubmittedAt.UTC()
		if graded.Valid {
			t := graded.Time.UTC()
			x.GradedAt = &t
		}
		items = append(items, x)
	}
	return http.StatusOK, map[string]any{"items": items}, rows.Err()
}

func getSubmission(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	secID := r.PathValue("id")
	if err := sectionExists(r, tx, secID); err != nil {
		return 0, nil, err
	}
	w, err := loadWork(r, tx, secID, r.PathValue("assessmentId"), "")
	if err != nil {
		return 0, nil, err
	}
	userID, staff, err := target(r, tx, s, secID)
	if err != nil {
		return 0, nil, err
	}
	sub, err := loadSubmission(r.Context(), tx, w.ID, userID, "")
	if err != nil {
		return 0, nil, err
	}
	if !staff {
		sub = own(sub, w)
	}
	return http.StatusOK, sub, nil
}

func putSubmission(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	ctx := r.Context()
	secID := r.PathValue("id")
	if err := sectionExists(r, tx, secID); err != nil {
		return 0, nil, err
	}
	w, err := loadWork(r, tx, secID, r.PathValue("assessmentId"), "for share")
	if err != nil {
		return 0, nil, err
	}
	if err := mustEnroll(r, tx, s, secID); err != nil {
		return 0, nil, err
	}
	if err := w.window(); err != nil {
		return 0, nil, err
	}
	var in struct {
		Body *string `json:"body"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	bad := func(m string) error { return httpx.Validation(httpx.FieldError{Field: "body", Message: m}) }
	if in.Body == nil {
		return 0, nil, bad("is required")
	}
	body := strings.TrimSpace(*in.Body)
	if strings.ContainsRune(body, 0) {
		return 0, nil, bad("must not contain NUL")
	}
	if utf8.RuneCountInString(body) > maxBody {
		return 0, nil, bad("must be at most 20000 characters")
	}
	// upsert first: it locks the row, so the file count below cannot race a file delete; the 400 rolls it back
	if _, err := tx.ExecContext(ctx, `insert into submissions (university_id, assessment_id, user_id, body) values ($1, $2, $3, $4)
		on conflict (university_id, assessment_id, user_id) do update set body = excluded.body, submitted_at = now()`,
		s.UniversityID, w.ID, s.UserID, body); err != nil {
		return 0, nil, err
	}
	files, err := fileCount(r, tx, w.ID, s.UserID)
	if err != nil {
		return 0, nil, err
	}
	if body == "" && files == 0 {
		return 0, nil, bad("a submission needs text or a file")
	}
	if err := auditUpdate(ctx, tx, s, w.ID); err != nil {
		return 0, nil, err
	}
	sub, err := loadSubmission(ctx, tx, w.ID, s.UserID, "")
	if err != nil {
		return 0, nil, err
	}
	return http.StatusOK, own(sub, w), nil
}
