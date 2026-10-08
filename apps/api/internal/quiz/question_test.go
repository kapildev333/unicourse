package quiz

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

type qm = map[string]any

func choiceQ(points string, correct ...bool) qm {
	cs := []qm{}
	for i, c := range correct {
		cs = append(cs, qm{"text": fmt.Sprintf("Choice %d", i), "correct": c})
	}
	return qm{"kind": "choice", "prompt": "Pick", "points": json.Number(points), "choices": cs}
}

func blankQ(points string, answers ...string) qm {
	return qm{"kind": "blank", "prompt": "Fill", "points": json.Number(points), "answers": answers}
}

type qres struct {
	Items []struct {
		ID       string
		Kind     string
		Points   json.Number
		Position int
		Multiple bool
		Choices  []map[string]any
		Answers  *[]string
	}
}

func (e *env) maxPoints(id string) (v string) {
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select max_points::text from assessments where id = $1`, id).Scan(&v)
	})
	return
}

func TestQuestionsPut(t *testing.T) {
	e := newEnv(t)
	id := e.quiz("-1 hour", "1 hour")
	items := []qm{choiceQ("2", true, false, false), choiceQ("1.5", true, true), blankQ("3.25", "Paris", " New  York ")}
	st, b := e.raw(e.Admin, "PUT", e.qpath(id), qm{"items": items})
	if st != 200 {
		t.Fatalf("%d %s", st, b)
	}
	var got qres
	json.Unmarshal([]byte(b), &got)
	if len(got.Items) != 3 {
		t.Fatalf("%s", b)
	}
	first := got.Items[0].ID
	for i, q := range got.Items {
		if q.Position != i+1 || q.ID == "" {
			t.Errorf("item %d %+v", i, q)
		}
	}
	if got.Items[0].Points != "2.00" || got.Items[1].Points != "1.50" || got.Items[2].Points != "3.25" {
		t.Errorf("points %s", b)
	}
	if got.Items[0].Multiple || !got.Items[1].Multiple || got.Items[2].Multiple {
		t.Errorf("multiple %s", b)
	}
	if got.Items[2].Answers == nil || !reflect.DeepEqual(*got.Items[2].Answers, []string{"Paris", "New  York"}) {
		t.Errorf("answers %s", b)
	}
	if got.Items[0].Answers == nil || len(*got.Items[0].Answers) != 0 || len(got.Items[2].Choices) != 0 {
		t.Errorf("empty key shapes %s", b)
	}
	if strings.Contains(b, "null") {
		t.Errorf("null in %s", b)
	}
	if mp := e.maxPoints(id); mp != "6.75" {
		t.Errorf("maxPoints %s", mp)
	}
	// GET by staff equals the PUT body; ids regenerate on the next PUT
	if st, g := e.raw(e.TA, "GET", e.qpath(id), nil); st != 200 || g != b {
		t.Errorf("get %d %s", st, g)
	}
	st, b = e.raw(e.Admin, "PUT", e.qpath(id), qm{"items": items})
	json.Unmarshal([]byte(b), &got)
	if st != 200 || got.Items[0].ID == first {
		t.Errorf("ids not regenerated %d", st)
	}
	// non-quiz assessment and unknown ids are 404
	var manual string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`insert into assessments (university_id, section_id, name, max_points, weight, position) values ($1,$2,'M',10,10,99) returning id`, e.Fx.UniversityA, e.SectionA).Scan(&manual)
	})
	for _, bad := range []string{manual, "nope", "00000000-0000-0000-0000-000000000000"} {
		for _, m := range []string{"GET", "PUT"} {
			if st, _ := e.raw(e.Admin, m, e.qpath(bad), qm{"items": []qm{}}); st != 404 {
				t.Errorf("%s %s -> %d", m, bad, st)
			}
		}
	}
	if st, _ := e.raw(e.Admin, "PUT", "/sections/"+id+"/assessments/"+id+"/questions", qm{"items": []qm{}}); st != 404 {
		t.Errorf("bad section %d", st)
	}
	// unknown keys and a missing items are 400
	if st, _ := e.raw(e.Admin, "PUT", e.qpath(id), qm{"items": []qm{}, "x": 1}); st != 400 {
		t.Errorf("unknown key %d", st)
	}
	st, b = e.raw(e.Admin, "PUT", e.qpath(id), qm{})
	if c, f := errDetails([]byte(b)); st != 400 || c != "validation_failed" || !reflect.DeepEqual(f, []string{"items"}) {
		t.Errorf("missing items %d %s", st, b)
	}
}

func TestQuestionsValidation(t *testing.T) {
	e := newEnv(t)
	id := e.quiz("-1 hour", "1 hour")
	long := func(n int) string { return strings.Repeat("a", n) }
	many := func(n int) []qm {
		cs := []qm{}
		for i := 0; i < n; i++ {
			cs = append(cs, qm{"text": "c", "correct": i == 0})
		}
		return cs
	}
	cases := []struct {
		name  string
		item  qm
		field string
		msg   string
	}{
		{"kind", qm{"kind": "essay", "prompt": "p", "points": 1}, "items[0].kind", "must be choice or blank"},
		{"prompt empty", qm{"kind": "blank", "prompt": "  ", "points": 1, "answers": []string{"a"}}, "items[0].prompt", "is required"},
		{"prompt long", qm{"kind": "blank", "prompt": long(2001), "points": 1, "answers": []string{"a"}}, "items[0].prompt", "must be at most 2000 characters"},
		{"prompt NUL", qm{"kind": "blank", "prompt": "a\u0000", "points": 1, "answers": []string{"a"}}, "items[0].prompt", "must not contain NUL"},
		{"points zero", qm{"kind": "blank", "prompt": "p", "points": 0, "answers": []string{"a"}}, "items[0].points", "must be between 0.01 and 1000"},
		{"points 1001", qm{"kind": "blank", "prompt": "p", "points": 1001, "answers": []string{"a"}}, "items[0].points", "must be between 0.01 and 1000"},
		{"points 3 decimals", qm{"kind": "blank", "prompt": "p", "points": json.Number("1.005"), "answers": []string{"a"}}, "items[0].points", "must have at most 2 decimals"},
		{"points missing", qm{"kind": "blank", "prompt": "p", "answers": []string{"a"}}, "items[0].points", "is required"},
		{"one choice", qm{"kind": "choice", "prompt": "p", "points": 1, "choices": many(1)}, "items[0].choices", "must have 2 to 10 choices"},
		{"no choices", qm{"kind": "choice", "prompt": "p", "points": 1}, "items[0].choices", "must have 2 to 10 choices"},
		{"eleven choices", qm{"kind": "choice", "prompt": "p", "points": 1, "choices": many(11)}, "items[0].choices", "must have 2 to 10 choices"},
		{"no correct", qm{"kind": "choice", "prompt": "p", "points": 1, "choices": []qm{{"text": "a"}, {"text": "b"}}}, "items[0].choices", "must have at least one correct choice"},
		{"choice text empty", qm{"kind": "choice", "prompt": "p", "points": 1, "choices": []qm{{"text": "a", "correct": true}, {"text": " "}}}, "items[0].choices[1].text", "is required"},
		{"choice text long", qm{"kind": "choice", "prompt": "p", "points": 1, "choices": []qm{{"text": long(501), "correct": true}, {"text": "b"}}}, "items[0].choices[0].text", "must be at most 500 characters"},
		{"choice text control", qm{"kind": "choice", "prompt": "p", "points": 1, "choices": []qm{{"text": "a\nb", "correct": true}, {"text": "b"}}}, "items[0].choices[0].text", "must not contain control characters"},
		{"choice with answers", qm{"kind": "choice", "prompt": "p", "points": 1, "choices": many(2), "answers": []string{"a"}}, "items[0].answers", "only for fill in the blank"},
		{"blank with choices", qm{"kind": "blank", "prompt": "p", "points": 1, "choices": many(2), "answers": []string{"a"}}, "items[0].choices", "only for multiple choice"},
		{"blank no answers", qm{"kind": "blank", "prompt": "p", "points": 1}, "items[0].answers", "must have 1 to 20 accepted answers"},
		{"blank 21 answers", qm{"kind": "blank", "prompt": "p", "points": 1, "answers": make([]string, 21)}, "items[0].answers", "must have 1 to 20 accepted answers"},
		{"answer empty", qm{"kind": "blank", "prompt": "p", "points": 1, "answers": []string{"a", " "}}, "items[0].answers[1]", "is required"},
		{"answer long", qm{"kind": "blank", "prompt": "p", "points": 1, "answers": []string{long(201)}}, "items[0].answers[0]", "must be at most 200 characters"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			st, b := e.raw(e.Admin, "PUT", e.qpath(id), qm{"items": []qm{c.item}})
			var body struct {
				Error struct {
					Code    string
					Details []struct{ Field, Message string }
				}
			}
			json.Unmarshal([]byte(b), &body)
			found := false
			for _, d := range body.Error.Details {
				found = found || (d.Field == c.field && d.Message == c.msg)
			}
			if st != 400 || body.Error.Code != "validation_failed" || !found {
				t.Fatalf("%d %s, want %s %q", st, b, c.field, c.msg)
			}
		})
	}
	// every error at once, with indexes
	st, b := e.raw(e.Admin, "PUT", e.qpath(id), qm{"items": []qm{blankQ("1", "ok"), {"kind": "x", "prompt": "", "points": 0}, qm{"kind": "choice", "prompt": "p", "points": 1}}})
	_, fields := errDetails([]byte(b))
	want := []string{"items[1].kind", "items[1].prompt", "items[1].points", "items[2].choices"}
	if st != 400 || !reflect.DeepEqual(fields, want) {
		t.Errorf("%d %v want %v", st, fields, want)
	}
	if n := e.questionCount(id); n != 0 {
		t.Errorf("a rejected put wrote %d rows", n)
	}
}

func (e *env) questionCount(id string) (n int) {
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select count(*) from questions where assessment_id = $1`, id).Scan(&n)
	})
	return
}

func TestQuestionsLimit(t *testing.T) {
	e := newEnv(t)
	id := e.quiz("-1 hour", "1 hour")
	items := make([]qm, 101)
	for i := range items {
		items[i] = blankQ("1", "a")
	}
	st, b := e.raw(e.Admin, "PUT", e.qpath(id), qm{"items": items})
	if c, f := errDetails([]byte(b)); st != 400 || c != "validation_failed" || !reflect.DeepEqual(f, []string{"items"}) || !strings.Contains(b, "must have at most 100 questions") {
		t.Fatalf("%d %s", st, b)
	}
	if st, b := e.raw(e.Admin, "PUT", e.qpath(id), qm{"items": items[:100]}); st != 200 {
		t.Fatalf("100: %d %s", st, b)
	}
}

func TestQuestionsMaxPoints(t *testing.T) {
	e := newEnv(t)
	id := e.quiz("-1 hour", "1 hour")
	if mp := e.maxPoints(id); mp != "1.00" {
		t.Fatalf("start %s", mp)
	}
	e.putQuestions(id, []qm{blankQ("0.10", "a"), blankQ("0.20", "a"), blankQ("2", "a")})
	if mp := e.maxPoints(id); mp != "2.30" {
		t.Errorf("sum %s", mp)
	}
	e.putQuestions(id, []qm{})
	if mp := e.maxPoints(id); mp != "1.00" {
		t.Errorf("empty %s", mp)
	}
	if n := e.questionCount(id); n != 0 {
		t.Errorf("rows %d", n)
	}
	// exactly 10000 passes, one cent more fails and changes nothing
	ten := make([]qm, 10)
	for i := range ten {
		ten[i] = blankQ("1000", "a")
	}
	e.putQuestions(id, ten)
	if mp := e.maxPoints(id); mp != "10000.00" {
		t.Errorf("10000: %s", mp)
	}
	ten = append(ten, blankQ("0.01", "a"))
	st, b := e.raw(e.Admin, "PUT", e.qpath(id), qm{"items": ten})
	if c, f := errDetails([]byte(b)); st != 400 || c != "validation_failed" || !reflect.DeepEqual(f, []string{"items"}) || !strings.Contains(b, "total points must be at most 10000") {
		t.Fatalf("%d %s", st, b)
	}
	if mp, n := e.maxPoints(id), e.questionCount(id); mp != "10000.00" || n != 10 {
		t.Errorf("rejected put changed %s %d", mp, n)
	}
}

func TestQuestionsLockedAfterAttempt(t *testing.T) {
	e := newEnv(t)
	id := e.quiz("-1 hour", "1 hour")
	e.putQuestions(id, []qm{blankQ("1", "a")})
	e.attempt(id, "stud@a.edu", false)
	st, b := e.raw(e.Admin, "PUT", e.qpath(id), qm{"items": []qm{}})
	if st != 409 || !strings.Contains(b, "questions are locked once a student has started") {
		t.Fatalf("%d %s", st, b)
	}
	// the lock is checked before the body: an invalid body is still 409
	if st, _ := e.raw(e.Admin, "PUT", e.qpath(id), qm{"items": []qm{{"kind": "x"}}}); st != 409 {
		t.Errorf("invalid body %d", st)
	}
	if n := e.questionCount(id); n != 1 {
		t.Errorf("rows %d", n)
	}
}

func TestQuestionsStudentWindow(t *testing.T) {
	e := newEnv(t)
	up := e.quiz("1 hour", "2 hours")
	open := e.quiz("-1 hour", "1 hour")
	closed := e.quiz("-2 hours", "-1 hour")
	for _, id := range []string{up, open, closed} {
		e.putQuestions(id, []qm{choiceQ("1", true, true, false), blankQ("2", "Paris")})
	}
	if st, b := e.raw(e.Stud, "GET", e.qpath(up), nil); st != 409 || !strings.Contains(b, "not open yet") {
		t.Errorf("upcoming %d %s", st, b)
	}
	if st, b := e.raw(e.Stud, "GET", e.qpath(closed), nil); st != 409 || !strings.Contains(b, "closed") {
		t.Errorf("closed %d %s", st, b)
	}
	// staff read any time, with keys
	for _, id := range []string{up, closed} {
		if st, b := e.raw(e.Inst, "GET", e.qpath(id), nil); st != 200 || !strings.Contains(b, `"correct":true`) || !strings.Contains(b, `"Paris"`) {
			t.Errorf("staff %d %s", st, b)
		}
	}
	// open: 200, no keys, multiple kept
	st, b := e.raw(e.Stud, "GET", e.qpath(open), nil)
	if st != 200 || strings.Contains(b, "correct") || strings.Contains(b, "Paris") || strings.Contains(b, `"answers"`) || !strings.Contains(b, `"multiple":true`) || !strings.Contains(b, `"multiple":false`) {
		t.Errorf("open %d %s", st, b)
	}
	// published without an own submitted attempt: still hidden
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`update assessments set published = true where id = $1`, open)
		return err
	})
	if st, b := e.raw(e.Stud, "GET", e.qpath(open), nil); st != 200 || strings.Contains(b, "correct") || strings.Contains(b, "Paris") {
		t.Errorf("published, no attempt %d %s", st, b)
	}
	// another student's submitted attempt does not help
	e.attempt(open, "stud2@a.edu", true)
	if st, b := e.raw(e.Stud, "GET", e.qpath(open), nil); st != 200 || strings.Contains(b, "Paris") {
		t.Errorf("other student's attempt %d %s", st, b)
	}
	// submitted + unpublished: readable after closing, hidden
	e.attempt(closed, "stud@a.edu", true)
	if st, b := e.raw(e.Stud, "GET", e.qpath(closed), nil); st != 200 || strings.Contains(b, "correct") || strings.Contains(b, "Paris") {
		t.Errorf("submitted, unpublished %d %s", st, b)
	}
	// submitted + published: keys shown
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`update assessments set published = true where id = $1`, closed)
		return err
	})
	if st, b := e.raw(e.Stud, "GET", e.qpath(closed), nil); st != 200 || !strings.Contains(b, `"correct":true`) || !strings.Contains(b, `"Paris"`) {
		t.Errorf("submitted, published %d %s", st, b)
	}
	// an unsubmitted attempt on a closed quiz is still 409 until the sweeper submits it
	e.attempt(closed, "stud2@a.edu", false)
	if st, b := e.raw(e.Stud2, "GET", e.qpath(closed), nil); st != 409 || !strings.Contains(b, "closed") {
		t.Errorf("unsubmitted %d %s", st, b)
	}
}

func TestQuestionsAudit(t *testing.T) {
	e := newEnv(t)
	id := e.quiz("-1 hour", "1 hour")
	e.putQuestions(id, []qm{blankQ("1.5", "a"), blankQ("2", "a")})
	e.putQuestions(id, []qm{blankQ("4", "a")})
	e.putQuestions(id, []qm{blankQ("4", "a")})
	var n int
	var before, after, actor string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		if err := tx.QueryRow(`select count(*) from audit_log where action = 'question.update' and entity_type = 'assessment' and entity_id = $1`, id).Scan(&n); err != nil {
			return err
		}
		return tx.QueryRow(`select before::text, after::text, actor_user_id::text from audit_log where action = 'question.update' and entity_id = $1 order by at, id limit 1 offset 1`, id).Scan(&before, &after, &actor)
	})
	if n != 3 || actor != e.Fx.AdminA {
		t.Fatalf("n %d actor %s", n, actor)
	}
	var b, a map[string]any
	json.Unmarshal([]byte(before), &b)
	json.Unmarshal([]byte(after), &a)
	if b["questionCount"] != 2.0 || b["totalPoints"] != "3.50" || a["questionCount"] != 1.0 || a["totalPoints"] != "4.00" {
		t.Errorf("before %s after %s", before, after)
	}
	// a rejected put writes none
	e.raw(e.Admin, "PUT", e.qpath(id), qm{"items": []qm{{"kind": "x"}}})
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select count(*) from audit_log where action = 'question.update' and entity_id = $1`, id).Scan(&n)
	})
	if n != 3 {
		t.Errorf("rejected put audited: %d", n)
	}
}

func TestQuestionsAuthz(t *testing.T) {
	e := newEnv(t)
	id := e.quiz("-1 hour", "1 hour")
	e.putQuestions(id, []qm{blankQ("1", "a")})
	put := qm{"items": []qm{blankQ("1", "a")}}
	// stud2 is enrolled too; "member" is a plain member of A in no section; BAdmin belongs to B only
	rows := []struct {
		who      string
		get, put int
	}{
		{"admin", 200, 200}, {"dain", 200, 200}, {"daout", 403, 403}, {"inst", 200, 200}, {"ta", 200, 200},
		{"stud", 200, 403}, {"stud2", 200, 403}, {"other", 403, 403}, {"badmin", 403, 403},
	}
	for _, r := range rows {
		c := e.as(r.who)
		if st, b := e.raw(c, "GET", e.qpath(id), nil); st != r.get {
			t.Errorf("%s GET %d %s", r.who, st, b)
		}
		if st, b := e.raw(c, "PUT", e.qpath(id), put); st != r.put {
			t.Errorf("%s PUT %d %s", r.who, st, b)
		}
	}
	if st, _ := e.raw(nil, "GET", e.qpath(id), nil); st != 401 {
		t.Errorf("anonymous %d", st)
	}
}
