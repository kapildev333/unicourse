package grade

import (
	"context"
	"database/sql"
	"encoding/json"
	"math/big"
	"net/http"
	"sort"

	"github.com/kapildev333/unicourse/apps/api/internal/authz"
)

type GradebookStudent struct {
	UserID      string                 `json:"userId"`
	Email       string                 `json:"email"`
	DisplayName string                 `json:"displayName"`
	Scores      map[string]json.Number `json:"scores"` // never nil
	Percent     json.Number            `json:"percent,omitempty"`
	Letter      string                 `json:"letter,omitempty"`
	Final       *FinalGrade            `json:"final,omitempty"`
	running     *big.Rat               // unrounded; nil when none
}

type Stats struct {
	Mean    json.Number    `json:"mean,omitempty"`
	Median  json.Number    `json:"median,omitempty"`
	Letters map[string]int `json:"letters"`
}

type Gradebook struct {
	Assessments []Assessment       `json:"assessments"`
	Students    []GradebookStudent `json:"students"`
	Stats       Stats              `json:"stats"`
}

// matrix is everything one section's grade computations read: 4 statements, none per student.
type matrix struct {
	assessments []Assessment
	students    []GradebookStudent // Scores and Final filled, Percent/Letter not yet
	scale       []ScaleRow
}

func loadMatrix(ctx context.Context, tx *sql.Tx, sectionID string) (matrix, error) {
	var m matrix
	var err error
	if m.assessments, err = listSectionAssessments(ctx, tx, sectionID, false); err != nil {
		return m, err
	}
	rows, err := tx.QueryContext(ctx, `select u.id, u.email, u.display_name,
		fg.id, fg.status, fg.percent::text, fg.letter, fg.points::text, fg.credits::text, fg.override_reason, fg.finalized_by, fg.finalized_at
		from enrollments e join users u on u.id = e.user_id
		left join final_grades fg on fg.section_id = e.section_id and fg.user_id = e.user_id
		where e.section_id = $1 order by u.email`, sectionID)
	if err != nil {
		return m, err
	}
	m.students = []GradebookStudent{}
	idx := map[string]int{}
	for rows.Next() {
		st := GradebookStudent{Scores: map[string]json.Number{}}
		var id, status, pct, letter, pts, credits, reason, by sql.NullString
		var at sql.NullTime
		if err := rows.Scan(&st.UserID, &st.Email, &st.DisplayName, &id, &status, &pct, &letter, &pts, &credits, &reason, &by, &at); err != nil {
			rows.Close()
			return m, err
		}
		if id.Valid {
			st.Final = &FinalGrade{id: id.String, UserID: st.UserID, Email: st.Email, DisplayName: st.DisplayName, Status: status.String,
				Percent: json.Number(pct.String), Letter: letter.String, Points: json.Number(pts.String), Credits: json.Number(credits.String),
				OverrideReason: reason.String, FinalizedBy: by.String, FinalizedAt: at.Time.UTC()}
		}
		idx[st.UserID] = len(m.students)
		m.students = append(m.students, st)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return m, err
	}
	// scores of users who are not enrolled are ignored
	rows, err = tx.QueryContext(ctx, `select sc.user_id::text, sc.assessment_id::text, sc.points::text
		from scores sc join assessments a on a.id = sc.assessment_id where a.section_id = $1`, sectionID)
	if err != nil {
		return m, err
	}
	for rows.Next() {
		var u, a string
		var p json.Number
		if err := rows.Scan(&u, &a, &p); err != nil {
			rows.Close()
			return m, err
		}
		if i, ok := idx[u]; ok {
			m.students[i].Scores[a] = p
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return m, err
	}
	m.scale, err = loadScale(ctx, tx)
	return m, err
}

// runningParts: the assessments with a score for st (published or not).
func (m matrix) runningParts(st GradebookStudent) []part {
	var out []part
	for _, a := range m.assessments {
		if p, ok := st.Scores[a.ID]; ok {
			out = append(out, part{rat(a.Weight), rat(a.MaxPoints), rat(p)})
		}
	}
	return out
}

// finalParts: every assessment, a missing score counts as 0.
func (m matrix) finalParts(st GradebookStudent) []part {
	out := make([]part, 0, len(m.assessments))
	for _, a := range m.assessments {
		p := new(big.Rat)
		if v, ok := st.Scores[a.ID]; ok {
			p = rat(v)
		}
		out = append(out, part{rat(a.Weight), rat(a.MaxPoints), p})
	}
	return out
}

// stats works on each student's effective grade: a graded final, else the running grade;
// incomplete and withdrawn finals are skipped.
func stats(students []GradebookStudent, scale []ScaleRow) Stats {
	out := Stats{Letters: map[string]int{}}
	for _, r := range scale {
		out.Letters[r.Letter] = 0
	}
	var vals []*big.Rat
	for _, st := range students {
		var v *big.Rat
		var l string
		switch {
		case st.Final != nil && st.Final.Status == "graded":
			v, l = rat(st.Final.Percent), st.Final.Letter
		case st.Final != nil:
			continue
		case st.running != nil:
			v, l = st.running, st.Letter
		default:
			continue
		}
		vals = append(vals, v)
		if l != "" {
			out.Letters[l]++
		}
	}
	if len(vals) == 0 {
		return out
	}
	sort.Slice(vals, func(i, j int) bool { return vals[i].Cmp(vals[j]) < 0 })
	sum := new(big.Rat)
	for _, v := range vals {
		sum.Add(sum, v)
	}
	n := int64(len(vals))
	out.Mean = round2(sum.Quo(sum, big.NewRat(n, 1)))
	med := new(big.Rat).Set(vals[n/2])
	if n%2 == 0 {
		med.Add(med, vals[n/2-1]).Quo(med, big.NewRat(2, 1))
	}
	out.Median = round2(med)
	return out
}

func getGradebook(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	secID := r.PathValue("id")
	if err := sectionExists(r, tx, secID); err != nil {
		return 0, nil, err
	}
	if err := grade(r, tx, s, secID); err != nil {
		return 0, nil, err
	}
	m, err := loadMatrix(r.Context(), tx, secID)
	if err != nil {
		return 0, nil, err
	}
	for i := range m.students {
		st := &m.students[i]
		if p, ok := percentOf(m.runningParts(*st)); ok {
			if row, ok := letterFor(m.scale, p); ok {
				st.running, st.Percent, st.Letter = p, round2(p), row.Letter
			}
		}
	}
	return http.StatusOK, Gradebook{Assessments: m.assessments, Students: m.students, Stats: stats(m.students, m.scale)}, nil
}

type part struct{ weight, max, points *big.Rat }

// percentOf = 100 x sum(weight x points/max) / sum(weight), exact; false when the weights sum to 0.
// A 0-weight part contributes nothing.
func percentOf(parts []part) (*big.Rat, bool) {
	sum, w := new(big.Rat), new(big.Rat)
	for _, p := range parts {
		w.Add(w, p.weight)
		t := new(big.Rat).Quo(p.points, p.max)
		sum.Add(sum, t.Mul(t, p.weight))
	}
	if w.Sign() == 0 {
		return nil, false
	}
	sum.Quo(sum, w)
	return sum.Mul(sum, big.NewRat(100, 1)), true
}

// round2 rounds half away from zero, which is half up for the non-negative values used here.
func round2(r *big.Rat) json.Number { return json.Number(r.FloatString(2)) }
