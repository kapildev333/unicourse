package quiz

import (
	"database/sql"
	"encoding/json"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"unicode"

	"github.com/kapildev333/unicourse/apps/api/internal/audit"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

type questionList struct {
	Items []Question `json:"items"`
}

// hideKeys is the student projection before the keys are released: no correct flags, no accepted answers.
// Multiple stays (it tells checkbox from radio without naming a choice).
func (q *Question) hideKeys() {
	for i := range q.Choices {
		q.Choices[i].Correct = nil
	}
	q.Answers = nil
}

func loadQuestions(r *http.Request, tx *sql.Tx, assessmentID string) ([]Question, error) {
	rows, err := tx.QueryContext(r.Context(), `select id, kind, prompt, points::text, position, choices, answers
		from questions where assessment_id = $1 order by position`, assessmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Question{}
	for rows.Next() {
		var q Question
		var choices, answers []byte
		if err := rows.Scan(&q.ID, &q.Kind, &q.Prompt, &q.Points, &q.Position, &choices, &answers); err != nil {
			return nil, err
		}
		as := []string{}
		if err := json.Unmarshal(choices, &q.Choices); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(answers, &as); err != nil {
			return nil, err
		}
		if q.Choices == nil {
			q.Choices = []Choice{}
		}
		q.Answers = &as
		n := 0
		for _, c := range q.Choices {
			if c.Correct != nil && *c.Correct {
				n++
			}
		}
		q.Multiple = n > 1
		out = append(out, q)
	}
	return out, rows.Err()
}

func listQuestions(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	secID := r.PathValue("id")
	if err := sectionExists(r, tx, secID); err != nil {
		return 0, nil, err
	}
	w, err := loadWork(r, tx, secID, r.PathValue("assessmentId"), "quiz", "")
	if err != nil {
		return 0, nil, err
	}
	staff, err := editor(r, tx, s, secID)
	if err != nil {
		return 0, nil, err
	}
	submitted := false
	if !staff {
		ok, err := enrolled(r, tx, secID, s.UserID)
		if err := allow(ok, err); err != nil {
			return 0, nil, err
		}
		if err := tx.QueryRowContext(r.Context(), `select exists(select 1 from attempts where assessment_id = $1 and user_id = $2 and submitted_at is not null)`,
			w.ID, s.UserID).Scan(&submitted); err != nil {
			return 0, nil, err
		}
		if !submitted && w.Upcoming {
			return 0, nil, httpx.Conflict("not open yet")
		}
		if !submitted && w.Closed {
			return 0, nil, httpx.Conflict("closed")
		}
	}
	qs, err := loadQuestions(r, tx, w.ID)
	if err != nil {
		return 0, nil, err
	}
	if !staff && !(w.Published && submitted) {
		for i := range qs {
			qs[i].hideKeys()
		}
	}
	return http.StatusOK, questionList{qs}, nil
}

type choiceIn struct {
	Text    string `json:"text"`
	Correct bool   `json:"correct"`
}

type questionIn struct {
	Kind    string      `json:"kind"`
	Prompt  string      `json:"prompt"`
	Points  json.Number `json:"points"`
	Choices []choiceIn  `json:"choices"`
	Answers []string    `json:"answers"`
}

const maxQuestions = 100

// checkQuestions validates every item (all errors at once, field order of design P18-L1 4.4) and returns the canonical items.
func checkQuestions(items []questionIn) ([]questionIn, []httpx.FieldError) {
	var d []httpx.FieldError
	if len(items) > maxQuestions {
		return nil, []httpx.FieldError{{Field: "items", Message: "must have at most 100 questions"}}
	}
	out := make([]questionIn, len(items))
	sum := new(big.Rat)
	for i, it := range items {
		p := "items[" + strconv.Itoa(i) + "]."
		add := func(f, m string) { d = append(d, httpx.FieldError{Field: p + f, Message: m}) }
		fe := func(f *httpx.FieldError) {
			if f != nil {
				add(f.Field, f.Message)
			}
		}
		o := questionIn{Kind: it.Kind}
		if it.Kind != "choice" && it.Kind != "blank" {
			add("kind", "must be choice or blank")
		}
		var f *httpx.FieldError
		o.Prompt, f = tenant.TrimLen("prompt", it.Prompt, 2000)
		fe(f)
		pts, f := tenant.Decimal("points", it.Points, 2, "0.01", "1000")
		fe(f)
		o.Points = json.Number(pts)
		if f == nil {
			r, _ := new(big.Rat).SetString(pts)
			sum.Add(sum, r)
		}
		switch it.Kind {
		case "choice":
			correct := 0
			n := len(it.Choices)
			if n < 2 || n > 10 {
				add("choices", "must have 2 to 10 choices")
			}
			for j, c := range it.Choices {
				field := "choices[" + strconv.Itoa(j) + "].text"
				t, f := tenant.TrimLen(field, c.Text, 500)
				if f == nil && strings.IndexFunc(t, unicode.IsControl) >= 0 {
					f = &httpx.FieldError{Field: field, Message: "must not contain control characters"}
				}
				fe(f)
				o.Choices = append(o.Choices, choiceIn{t, c.Correct})
				if c.Correct {
					correct++
				}
			}
			if n >= 2 && n <= 10 && correct == 0 {
				add("choices", "must have at least one correct choice")
			}
			if len(it.Answers) > 0 {
				add("answers", "only for fill in the blank")
			}
		case "blank":
			if len(it.Choices) > 0 {
				add("choices", "only for multiple choice")
			}
			if n := len(it.Answers); n < 1 || n > 20 {
				add("answers", "must have 1 to 20 accepted answers")
			}
			for j, a := range it.Answers {
				t, f := tenant.TrimLen("answers["+strconv.Itoa(j)+"]", a, 200)
				fe(f)
				o.Answers = append(o.Answers, t)
			}
		}
		out[i] = o
	}
	if len(d) == 0 && sum.Cmp(big.NewRat(10000, 1)) > 0 {
		d = append(d, httpx.FieldError{Field: "items", Message: "total points must be at most 10000"})
	}
	return out, d
}

func putQuestions(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	secID := r.PathValue("id")
	if err := sectionExists(r, tx, secID); err != nil {
		return 0, nil, err
	}
	w, err := loadWork(r, tx, secID, r.PathValue("assessmentId"), "quiz", "for update")
	if err != nil {
		return 0, nil, err
	}
	ok, err := editor(r, tx, s, secID)
	if err := allow(ok, err); err != nil {
		return 0, nil, err
	}
	var started bool
	if err := tx.QueryRowContext(r.Context(), `select exists(select 1 from attempts where assessment_id = $1)`, w.ID).Scan(&started); err != nil {
		return 0, nil, err
	}
	if started {
		return 0, nil, httpx.Conflict("questions are locked once a student has started")
	}
	var in struct {
		Items *[]questionIn `json:"items"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	if in.Items == nil {
		return 0, nil, httpx.Validation(httpx.FieldError{Field: "items", Message: "is required"})
	}
	items, d := checkQuestions(*in.Items)
	if len(d) > 0 {
		return 0, nil, httpx.Validation(d...)
	}
	// ponytail: one jsonb array instead of the design's unnest of five text[] (no array params elsewhere in the API); same rows
	rows := make([]map[string]any, len(items))
	total := new(big.Rat)
	for i, it := range items {
		if it.Choices == nil {
			it.Choices = []choiceIn{}
		}
		if it.Answers == nil {
			it.Answers = []string{}
		}
		rows[i] = map[string]any{"kind": it.Kind, "prompt": it.Prompt, "points": it.Points, "choices": it.Choices, "answers": it.Answers}
		p, _ := new(big.Rat).SetString(string(it.Points))
		total.Add(total, p)
	}
	raw, err := json.Marshal(rows)
	if err != nil {
		return 0, nil, err
	}
	type summary struct {
		QuestionCount int    `json:"questionCount"`
		TotalPoints   string `json:"totalPoints"`
	}
	var before summary
	if err := tx.QueryRowContext(r.Context(), `select count(*), coalesce(sum(points), 0.00)::text from questions where assessment_id = $1`, w.ID).Scan(&before.QuestionCount, &before.TotalPoints); err != nil {
		return 0, nil, err
	}
	if _, err := tx.ExecContext(r.Context(), `delete from questions where assessment_id = $1`, w.ID); err != nil {
		return 0, nil, err
	}
	if _, err := tx.ExecContext(r.Context(), `insert into questions (university_id, assessment_id, kind, prompt, points, position, choices, answers)
		select $1, $2, x.v->>'kind', x.v->>'prompt', (x.v->>'points')::numeric, x.n::int, x.v->'choices', x.v->'answers'
		from jsonb_array_elements($3::jsonb) with ordinality as x(v, n)`, s.UniversityID, w.ID, string(raw)); err != nil {
		return 0, nil, err
	}
	max := "1" // an empty quiz keeps 1 (the column needs > 0)
	if len(items) > 0 {
		max = total.FloatString(2)
	}
	if _, err := tx.ExecContext(r.Context(), `update assessments set max_points = $2::numeric where id = $1`, w.ID, max); err != nil {
		return 0, nil, err
	}
	if err := audit.Write(r.Context(), tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "question.update", EntityType: "assessment", EntityID: w.ID,
		Before: before, After: summary{len(items), total.FloatString(2)}}); err != nil {
		return 0, nil, err
	}
	qs, err := loadQuestions(r, tx, w.ID)
	return http.StatusOK, questionList{qs}, err
}
