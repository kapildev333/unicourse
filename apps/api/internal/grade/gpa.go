package grade

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

type sectionRef struct {
	ID     string `json:"id"`
	Number string `json:"number"`
}

type finalSummary struct {
	Status  string      `json:"status"`
	Percent json.Number `json:"percent,omitempty"`
	Letter  string      `json:"letter,omitempty"`
	Points  json.Number `json:"points,omitempty"`
}

type runningGrade struct {
	Percent json.Number `json:"percent"`
	Letter  string      `json:"letter"`
}

type myAssessment struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Weight    json.Number `json:"weight"`
	MaxPoints json.Number `json:"maxPoints"`
	DueOn     string      `json:"dueOn,omitempty"`
	Points    json.Number `json:"points,omitempty"`
}

// courseRef has exactly the JSON fields of course.Course (the grade package does not import course).
type courseRef struct {
	ID           string      `json:"id"`
	DepartmentID string      `json:"departmentId"`
	Code         string      `json:"code"`
	Title        string      `json:"title"`
	Credits      json.Number `json:"credits"`
}

type myGradesSection struct {
	Section     sectionRef     `json:"section"`
	Course      courseRef      `json:"course"`
	Credits     json.Number    `json:"credits"`
	Final       *finalSummary  `json:"final,omitempty"`
	Running     *runningGrade  `json:"running,omitempty"`
	Assessments []myAssessment `json:"assessments"`
}

type myGradesTerm struct {
	Term     tenant.Term       `json:"term"`
	GPA      json.Number       `json:"gpa,omitempty"`
	Credits  json.Number       `json:"credits"`
	Sections []myGradesSection `json:"sections"`
}

type myGrades struct {
	Terms             []myGradesTerm `json:"terms"`
	CumulativeGPA     json.Number    `json:"cumulativeGpa,omitempty"`
	CumulativeCredits json.Number    `json:"cumulativeCredits"`
}

// getMyGrades serves the caller's own grades: published assessments only, a running grade only while
// nothing unpublished is scored, finals for sections the caller may have left, and GPA from the snapshot credits.
func getMyGrades(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	ctx := r.Context()
	rows, err := tx.QueryContext(ctx, `with mine as (select section_id from enrollments where user_id = $1
			union select section_id from final_grades where user_id = $1)
		select s.id, s.number, c.id, c.department_id, c.code, c.title, c.credits::text,
		  t.id, t.name, to_char(t.starts_on, 'YYYY-MM-DD'), to_char(t.ends_on, 'YYYY-MM-DD'),
		  fg.status, fg.percent::text, fg.letter, fg.points::text
		from mine join sections s on s.id = mine.section_id join courses c on c.id = s.course_id join terms t on t.id = s.term_id
		left join final_grades fg on fg.section_id = s.id and fg.user_id = $1
		order by t.starts_on desc, t.id, c.code, s.number`, s.UserID)
	if err != nil {
		return 0, nil, err
	}
	out := myGrades{Terms: []myGradesTerm{}, CumulativeCredits: "0"}
	idx := map[string]int{} // term id -> index in out.Terms
	var secIDs []string
	for rows.Next() {
		var x myGradesSection
		var t tenant.Term
		var status, pct, letter, pts sql.NullString
		if err := rows.Scan(&x.Section.ID, &x.Section.Number, &x.Course.ID, &x.Course.DepartmentID, &x.Course.Code, &x.Course.Title, &x.Course.Credits,
			&t.ID, &t.Name, &t.StartsOn, &t.EndsOn, &status, &pct, &letter, &pts); err != nil {
			rows.Close()
			return 0, nil, err
		}
		x.Credits, x.Assessments = x.Course.Credits, []myAssessment{}
		if status.Valid {
			x.Final = &finalSummary{Status: status.String, Percent: json.Number(pct.String), Letter: letter.String, Points: json.Number(pts.String)}
		}
		i, ok := idx[t.ID]
		if !ok {
			i = len(out.Terms)
			idx[t.ID] = i
			out.Terms = append(out.Terms, myGradesTerm{Term: t, Credits: "0", Sections: []myGradesSection{}})
		}
		out.Terms[i].Sections = append(out.Terms[i].Sections, x)
		secIDs = append(secIDs, x.Section.ID)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, nil, err
	}
	if len(secIDs) == 0 {
		return http.StatusOK, out, nil
	}
	// assessments with the caller's score; unpublished ones decide `running` but are never output
	type key struct{ term, sec int }
	where := map[string]key{}
	for ti, t := range out.Terms {
		for si, x := range t.Sections {
			where[x.Section.ID] = key{ti, si}
		}
	}
	scale, err := loadScale(ctx, tx)
	if err != nil {
		return 0, nil, err
	}
	rows, err = tx.QueryContext(ctx, `select a.section_id::text, a.id, a.name, a.weight::text, a.max_points::text, coalesce(to_char(a.due_on, 'YYYY-MM-DD'), ''), a.published, sc.points::text
		from assessments a left join scores sc on sc.assessment_id = a.id and sc.user_id = $1
		where a.section_id = any($2::uuid[]) order by a.section_id, a.position, a.created_at, a.id`, s.UserID, secIDs)
	if err != nil {
		return 0, nil, err
	}
	parts, blocked := map[string][]part{}, map[string]bool{}
	for rows.Next() {
		var sec string
		var a myAssessment
		var published bool
		var pts sql.NullString
		if err := rows.Scan(&sec, &a.ID, &a.Name, &a.Weight, &a.MaxPoints, &a.DueOn, &published, &pts); err != nil {
			rows.Close()
			return 0, nil, err
		}
		a.Points = json.Number(pts.String)
		if pts.Valid {
			if !published {
				blocked[sec] = true
			}
			parts[sec] = append(parts[sec], part{rat(a.Weight), rat(a.MaxPoints), rat(a.Points)})
		}
		if published {
			k := where[sec]
			x := &out.Terms[k.term].Sections[k.sec]
			x.Assessments = append(x.Assessments, a)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, nil, err
	}
	for sec, k := range where {
		x := &out.Terms[k.term].Sections[k.sec]
		if x.Final != nil || blocked[sec] {
			continue
		}
		if p, ok := percentOf(parts[sec]); ok && len(parts[sec]) > 0 {
			if row, ok := letterFor(scale, p); ok {
				x.Running = &runningGrade{round2(p), row.Letter}
			}
		}
	}
	// GPA: numeric end to end; the rollup row (term_id null) is the cumulative one and exists even with no input rows
	rows, err = tx.QueryContext(ctx, `select s.term_id::text, round(sum(fg.points * fg.credits) / nullif(sum(fg.credits), 0), 2)::text, coalesce(sum(fg.credits), 0)::text
		from final_grades fg join sections s on s.id = fg.section_id
		where fg.user_id = $1 and fg.status = 'graded'
		group by rollup (s.term_id)`, s.UserID)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var term, gpa sql.NullString
		var credits string
		if err := rows.Scan(&term, &gpa, &credits); err != nil {
			return 0, nil, err
		}
		if !term.Valid {
			out.CumulativeGPA, out.CumulativeCredits = json.Number(gpa.String), json.Number(credits)
		} else if i, ok := idx[term.String]; ok {
			out.Terms[i].GPA, out.Terms[i].Credits = json.Number(gpa.String), json.Number(credits)
		}
	}
	return http.StatusOK, out, rows.Err()
}
