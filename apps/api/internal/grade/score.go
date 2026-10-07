package grade

import (
	"context"
	"database/sql"
	"encoding/json"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/audit"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

type Score struct {
	UserID   string      `json:"userId"`
	Points   json.Number `json:"points"`
	GradedBy string      `json:"gradedBy"`
	GradedAt time.Time   `json:"gradedAt"`
}

type scoreList struct {
	Items []Score `json:"items"`
}

// scoreIn is a validated row: canonical points ("87.50"); Points "" = delete (PUT null).
type scoreIn struct{ UserID, Points string }

type scoreAudit struct {
	UserID string      `json:"userId"`
	Points json.Number `json:"points,omitempty"` // omitted in after = deleted
}

// ponytail: json.Number also accepts a JSON string holding a number ("87.5"); a strict number type would reject it.
// rat reads a numeric::text value; ok is always true for those (panics otherwise).
func rat(n json.Number) *big.Rat {
	r, ok := new(big.Rat).SetString(string(n))
	if !ok {
		panic("grade: not a number: " + string(n))
	}
	return r
}

// maxScore is 2 x maxPoints (bonus allowed), 2 decimals.
func maxScore(a Assessment) string {
	return new(big.Rat).Mul(rat(a.MaxPoints), big.NewRat(2, 1)).FloatString(2)
}

func putScores(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	ctx := r.Context()
	secID := r.PathValue("id")
	if err := sectionExists(r, tx, secID); err != nil {
		return 0, nil, err
	}
	a, err := loadAssessment(r, tx, secID, r.PathValue("assessmentId"), false)
	if err != nil {
		return 0, nil, err
	}
	if err := grade(r, tx, s, secID); err != nil {
		return 0, nil, err
	}
	var in struct {
		Scores []struct {
			UserID string                  `json:"userId"`
			Points tenant.Opt[json.Number] `json:"points"`
		} `json:"scores"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	if len(in.Scores) < 1 || len(in.Scores) > 500 {
		return 0, nil, httpx.Validation(httpx.FieldError{Field: "scores", Message: "must have 1 to 500 entries"})
	}
	var d []httpx.FieldError
	bad := func(i int, f, m string) {
		d = append(d, httpx.FieldError{Field: "scores[" + strconv.Itoa(i) + "]." + f, Message: m})
	}
	// prepass: ids that are valid and first occurrences are looked up; the rest get their error below
	seen := map[string]bool{}
	var ids []string
	for _, x := range in.Scores {
		if tenant.IsUUID(x.UserID) && !seen[x.UserID] {
			ids = append(ids, x.UserID)
		}
		seen[x.UserID] = true
	}
	enrolled := map[string]bool{}
	rows, err := tx.QueryContext(ctx, `select user_id::text from enrollments where section_id = $1 and user_id = any($2::uuid[])`, secID, ids)
	if err != nil {
		return 0, nil, err
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, nil, err
		}
		enrolled[id] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, nil, err
	}
	out := make([]scoreIn, len(in.Scores))
	again := map[string]bool{}
	for i, x := range in.Scores {
		out[i].UserID = x.UserID
		switch {
		case !tenant.IsUUID(x.UserID):
			bad(i, "userId", "must be a uuid")
		case again[x.UserID]:
			bad(i, "userId", "duplicate userId")
		case !enrolled[x.UserID]:
			bad(i, "userId", "is not enrolled in this section")
		}
		again[x.UserID] = true
		switch {
		case !x.Points.Set:
			bad(i, "points", "is required")
		case x.Points.Null: // deletes the score
		default:
			p, fe := tenant.Decimal("points", x.Points.V, 2, "0", maxScore(a))
			if fe != nil {
				bad(i, "points", fe.Message)
			}
			out[i].Points = p
		}
	}
	if len(d) > 0 {
		return 0, nil, httpx.Validation(d...)
	}
	if err := setScores(ctx, tx, s, a, out); err != nil {
		return 0, nil, err
	}
	items, err := listScores(ctx, tx, a)
	return http.StatusOK, scoreList{items}, err
}

// setScores upserts and deletes in one transaction step and writes one score.set audit row;
// used by putScores (and the import applier).
func setScores(ctx context.Context, tx *sql.Tx, s authz.Scope, a Assessment, in []scoreIn) error {
	all := make([]string, len(in))
	var upIDs, upPts, delIDs []string
	for i, x := range in {
		all[i] = x.UserID
		if x.Points == "" {
			delIDs = append(delIDs, x.UserID)
		} else {
			upIDs, upPts = append(upIDs, x.UserID), append(upPts, x.Points)
		}
	}
	old := map[string]json.Number{}
	rows, err := tx.QueryContext(ctx, `select user_id::text, points::text from scores where assessment_id = $1 and user_id = any($2::uuid[])`, a.ID, all)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id string
		var p json.Number
		if err := rows.Scan(&id, &p); err != nil {
			rows.Close()
			return err
		}
		old[id] = p
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(upIDs) > 0 {
		if _, err := tx.ExecContext(ctx, `insert into scores (university_id, assessment_id, user_id, points, graded_by)
			select $1, $2, u, p::numeric, $3 from unnest($4::uuid[], $5::text[]) as x(u, p)
			on conflict (university_id, assessment_id, user_id) do update
			  set points = excluded.points, graded_by = excluded.graded_by, graded_at = now()
			  where scores.points is distinct from excluded.points`, s.UniversityID, a.ID, s.UserID, upIDs, upPts); err != nil {
			return err
		}
	}
	if len(delIDs) > 0 {
		if _, err := tx.ExecContext(ctx, `delete from scores where assessment_id = $1 and user_id = any($2::uuid[])`, a.ID, delIDs); err != nil {
			return err
		}
	}
	after := make([]scoreAudit, len(in))
	var before []scoreAudit
	for i, x := range in {
		after[i] = scoreAudit{UserID: x.UserID, Points: json.Number(x.Points)}
		if p, ok := old[x.UserID]; ok {
			before = append(before, scoreAudit{UserID: x.UserID, Points: p})
		}
	}
	e := audit.Entry{ActorUserID: s.UserID, Action: "score.set", EntityType: "assessment", EntityID: a.ID,
		After: map[string]any{"assessmentId": a.ID, "scores": after}}
	if len(before) > 0 {
		e.Before = map[string]any{"scores": before}
	}
	return audit.Write(ctx, tx, s.UniversityID, e)
}

// listScores returns the assessment's scores of enrolled students, ordered by user_id.
func listScores(ctx context.Context, tx *sql.Tx, a Assessment) ([]Score, error) {
	rows, err := tx.QueryContext(ctx, `select sc.user_id, sc.points::text, sc.graded_by, sc.graded_at from scores sc
		join enrollments e on e.section_id = $2 and e.user_id = sc.user_id where sc.assessment_id = $1 order by sc.user_id`, a.ID, a.SectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Score{}
	for rows.Next() {
		var x Score
		if err := rows.Scan(&x.UserID, &x.Points, &x.GradedBy, &x.GradedAt); err != nil {
			return nil, err
		}
		x.GradedAt = x.GradedAt.UTC()
		out = append(out, x)
	}
	return out, rows.Err()
}
