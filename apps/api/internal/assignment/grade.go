package assignment

import (
	"database/sql"
	"encoding/json"
	"math/big"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/kapildev333/unicourse/apps/api/internal/audit"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/grade"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

type gradeAudit struct {
	UserID string      `json:"userId"`
	Score  json.Number `json:"score,omitempty"`
}

// gradeSubmission sets score and feedback (D141) and upserts the scores row; allowed any time after a
// submission exists, also after closes_at.
func gradeSubmission(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	ctx := r.Context()
	secID := r.PathValue("id")
	if err := sectionExists(r, tx, secID); err != nil {
		return 0, nil, err
	}
	w, err := loadWork(r, tx, secID, r.PathValue("assessmentId"), "for share")
	if err != nil {
		return 0, nil, err
	}
	if err := editor(r, tx, s, secID); err != nil {
		return 0, nil, err
	}
	userID := r.PathValue("userId")
	if !tenant.IsUUID(userID) {
		return 0, nil, httpx.NotFound("submission")
	}
	// like the gradebook, only enrolled students are graded
	if ok, err := enrolled(ctx, tx, secID, userID); err != nil {
		return 0, nil, err
	} else if !ok {
		return 0, nil, httpx.NotFound("submission")
	}
	before, err := loadSubmission(ctx, tx, w.ID, userID, "for update")
	if err != nil {
		return 0, nil, err
	}
	var in struct {
		Score    json.Number        `json:"score"`
		Feedback tenant.Opt[string] `json:"feedback"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	var d []httpx.FieldError
	max := new(big.Rat).Mul(mustRat(w.MaxPoints), big.NewRat(2, 1)).FloatString(2)
	score, fe := tenant.Decimal("score", in.Score, 2, "0", max)
	if fe != nil {
		d = append(d, *fe)
	}
	feedback := *before.Feedback
	if in.Feedback.Null {
		d = append(d, httpx.FieldError{Field: "feedback", Message: "must not be null"})
	} else if in.Feedback.Set {
		feedback = strings.TrimSpace(in.Feedback.V)
		if strings.ContainsRune(feedback, 0) {
			d = append(d, httpx.FieldError{Field: "feedback", Message: "must not contain NUL"})
		} else if utf8.RuneCountInString(feedback) > 5000 {
			d = append(d, httpx.FieldError{Field: "feedback", Message: "must be at most 5000 characters"})
		}
	}
	if len(d) > 0 {
		return 0, nil, httpx.Validation(d...)
	}
	if _, err := tx.ExecContext(ctx, `update submissions set score = $2::numeric, feedback = $3, graded_by = $4, graded_at = now() where id = $1`,
		before.id, score, feedback, s.UserID); err != nil {
		return 0, nil, err
	}
	if err := grade.UpsertScore(ctx, tx, s.UniversityID, w.ID, userID, score, s.UserID); err != nil {
		return 0, nil, err
	}
	after, err := loadSubmission(ctx, tx, w.ID, userID, "")
	if err != nil {
		return 0, nil, err
	}
	if err := audit.Write(ctx, tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "submission.grade", EntityType: "assessment", EntityID: w.ID,
		Before: gradeAudit{userID, before.Score}, After: gradeAudit{userID, after.Score}}); err != nil {
		return 0, nil, err
	}
	return http.StatusOK, after, nil
}

// mustRat reads a numeric::text value.
func mustRat(v string) *big.Rat {
	r, ok := new(big.Rat).SetString(v)
	if !ok {
		panic("assignment: not a number: " + v)
	}
	return r
}
