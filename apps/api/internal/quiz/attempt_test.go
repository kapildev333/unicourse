package quiz

import (
	"database/sql"
	"strings"
	"testing"
)

func (e *env) apath(id string) string {
	return "/sections/" + e.SectionA + "/assessments/" + id + "/attempts"
}

// openQuiz is an open quiz with c1 (choice, 2.00, first choice right) and b1 (blank, 3.25, "Paris"); it returns the quiz and question ids.
func (e *env) openQuiz() (id string, qs []string) {
	e.T.Helper()
	id = e.quiz("-1 hour", "1 hour")
	e.putQuestions(id, []any{
		map[string]any{"kind": "choice", "prompt": "Pick", "points": 2, "choices": []any{map[string]any{"text": "A", "correct": true}, map[string]any{"text": "B", "correct": false}}},
		map[string]any{"kind": "blank", "prompt": "Capital of France", "points": 3.25, "answers": []any{"Paris"}},
	})
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		rows, err := tx.Query(`select id from questions where assessment_id = $1 order by position`, id)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var q string
			rows.Scan(&q)
			qs = append(qs, q)
		}
		return rows.Err()
	})
	return id, qs
}

func (e *env) setWindow(id, opens, closes string) {
	e.T.Helper()
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`update assessments set opens_at = now() + $2::interval, closes_at = now() + $3::interval where id = $1`, id, opens, closes)
		return err
	})
}

func TestAttemptStart(t *testing.T) {
	e := newEnv(t)
	up := e.quiz("1 hour", "2 hours")
	if st, b := e.raw(e.Stud, "POST", e.apath(up), nil); st != 409 || !strings.Contains(b, "not open yet") {
		t.Fatalf("upcoming %d %s", st, b)
	}
	cl := e.quiz("-2 hours", "-1 hour")
	if st, b := e.raw(e.Stud, "POST", e.apath(cl), nil); st != 409 || !strings.Contains(b, "closed") {
		t.Fatalf("closed %d %s", st, b)
	}
	empty := e.quiz("-1 hour", "1 hour")
	if st, b := e.raw(e.Stud, "POST", e.apath(empty), nil); st != 409 || !strings.Contains(b, "the quiz has no questions") {
		t.Fatalf("empty %d %s", st, b)
	}
	id, _ := e.openQuiz()
	for _, who := range []string{"inst", "ta", "admin", "other"} {
		if st, b := e.raw(e.as(who), "POST", e.apath(id), nil); st != 403 {
			t.Fatalf("%s %d %s", who, st, b)
		}
	}
	st, b := e.call(e.Stud, "POST", e.apath(id), nil)
	if st != 201 {
		t.Fatalf("start %d %s", st, b)
	}
	m := jmap(t, b)
	if m["userId"] != e.UserID["stud@a.edu"] || m["startedAt"] == nil || m["autoSubmitted"] != false || len(m["answers"].(map[string]any)) != 0 {
		t.Fatalf("shape %s", b)
	}
	for _, k := range []string{"submittedAt", "grading", "score", "feedback", "reviewedAt"} {
		if _, ok := m[k]; ok {
			t.Fatalf("%s present in %s", k, b)
		}
	}
	st, b2 := e.call(e.Stud, "POST", e.apath(id), nil)
	if st != 200 || jmap(t, b2)["startedAt"] != m["startedAt"] {
		t.Fatalf("second %d %s", st, b2)
	}
	if n, _, after := e.audit("attempt.start", id); n != 1 || !strings.Contains(after.String, e.UserID["stud@a.edu"]) {
		t.Fatalf("audit %d %s", n, after.String)
	}
}

func TestAttemptSave(t *testing.T) {
	e := newEnv(t)
	id, qs := e.openQuiz()
	if st, b := e.raw(e.Stud, "PUT", e.apath(id)+"/me", map[string]any{"answers": map[string]any{}}); st != 404 {
		t.Fatalf("no attempt %d %s", st, b)
	}
	e.call(e.Stud, "POST", e.apath(id), nil)
	if st, b := e.raw(e.Inst, "PUT", e.apath(id)+"/me", map[string]any{"answers": map[string]any{}}); st != 403 {
		t.Fatalf("staff %d %s", st, b)
	}
	put := func(a map[string]any) (int, []string) {
		st, b := e.call(e.Stud, "PUT", e.apath(id)+"/me", map[string]any{"answers": a})
		_, f := errDetails(b)
		return st, f
	}
	bad := map[string]map[string]any{
		"answers.nope":                  {"nope": map[string]any{"text": "x"}},
		"answers." + qs[0] + ".choices": {qs[0]: map[string]any{"choices": []int{2}}},
		"answers." + qs[1] + ".text":    {qs[1]: map[string]any{"text": strings.Repeat("x", 1001)}},
	}
	for field, a := range bad {
		if st, f := put(a); st != 400 || len(f) != 1 || f[0] != field {
			t.Errorf("%s: %d %v", field, st, f)
		}
	}
	for _, a := range []map[string]any{
		{qs[0]: map[string]any{"choices": []int{0, 0}}},
		{qs[0]: map[string]any{"text": "x"}},
		{qs[1]: map[string]any{"choices": []int{0}}},
	} {
		if st, f := put(a); st != 400 || len(f) != 1 {
			t.Errorf("%v: %d %v", a, st, f)
		}
	}
	if st, b := e.raw(e.Stud, "PUT", e.apath(id)+"/me", map[string]any{}); st != 400 || !strings.Contains(b, "is required") {
		t.Fatalf("missing answers %d %s", st, b)
	}
	st, b := e.call(e.Stud, "PUT", e.apath(id)+"/me", map[string]any{"answers": map[string]any{qs[0]: map[string]any{"choices": []int{1}}, qs[1]: map[string]any{}}})
	if st != 200 {
		t.Fatalf("save %d %s", st, b)
	}
	a := jmap(t, b)["answers"].(map[string]any)
	if len(a) != 1 || a[qs[0]] == nil {
		t.Fatalf("empty entries kept: %s", b)
	}
	// the whole map is replaced
	_, b = e.call(e.Stud, "PUT", e.apath(id)+"/me", map[string]any{"answers": map[string]any{qs[1]: map[string]any{"text": "Paris"}}})
	if a := jmap(t, b)["answers"].(map[string]any); len(a) != 1 || a[qs[1]] == nil {
		t.Fatalf("not replaced: %s", b)
	}
}

func TestAttemptSubmit(t *testing.T) {
	e := newEnv(t)
	id, qs := e.openQuiz()
	e.call(e.Stud, "POST", e.apath(id), nil)
	st, b := e.call(e.Stud, "PUT", e.apath(id)+"/me", map[string]any{"submit": true, "answers": map[string]any{
		qs[0]: map[string]any{"choices": []int{0}}, qs[1]: map[string]any{"text": " paris "}}})
	if st != 200 {
		t.Fatalf("submit %d %s", st, b)
	}
	m := jmap(t, b)
	if m["submittedAt"] == nil || m["autoSubmitted"] != false {
		t.Fatalf("shape %s", b)
	}
	if _, ok := m["score"]; ok { // unpublished: hidden from the student
		t.Fatalf("score leaked %s", b)
	}
	var score, graded, gradedBy, grading string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		if err := tx.QueryRow(`select score::text, grading::text from attempts where assessment_id = $1`, id).Scan(&score, &grading); err != nil {
			return err
		}
		return tx.QueryRow(`select points::text, graded_by::text from scores where assessment_id = $1 and user_id = $2`, id, e.UserID["stud@a.edu"]).Scan(&graded, &gradedBy)
	})
	if score != "5.25" || graded != "5.25" || gradedBy != e.UserID["stud@a.edu"] || !strings.Contains(grading, "3.25") {
		t.Fatalf("score %s scores %s by %s grading %s", score, graded, gradedBy, grading)
	}
	n, _, after := e.audit("attempt.submit", id)
	if n != 1 || !strings.Contains(after.String, "5.25") || !strings.Contains(after.String, `"auto": false`) {
		t.Fatalf("audit %d %s", n, after.String)
	}
	// submitting again, or saving, is 409
	for _, body := range []map[string]any{{"answers": map[string]any{}}, {"answers": map[string]any{}, "submit": true}} {
		if st, b := e.raw(e.Stud, "PUT", e.apath(id)+"/me", body); st != 409 || !strings.Contains(b, "already submitted") {
			t.Fatalf("again %d %s", st, b)
		}
	}
}

func TestAttemptPutAfterClose(t *testing.T) {
	e := newEnv(t)
	id, qs := e.openQuiz()
	e.call(e.Stud, "POST", e.apath(id), nil)
	e.call(e.Stud, "PUT", e.apath(id)+"/me", map[string]any{"answers": map[string]any{qs[1]: map[string]any{"text": "Paris"}}})
	e.setWindow(id, "-2 hours", "-1 hour")
	st, b := e.raw(e.Stud, "PUT", e.apath(id)+"/me", map[string]any{"submit": true, "answers": map[string]any{qs[1]: map[string]any{"text": "Rome"}}})
	if st != 409 || !strings.Contains(b, "closed; your answers as of the closing time were submitted") {
		t.Fatalf("%d %s", st, b)
	}
	var answers string
	var submitted bool
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select answers::text, submitted_at is not null from attempts where assessment_id = $1`, id).Scan(&answers, &submitted)
	})
	if submitted || !strings.Contains(answers, "Paris") || strings.Contains(answers, "Rome") {
		t.Fatalf("changed: %v %s", submitted, answers)
	}
}

func TestAttemptReadHidesResults(t *testing.T) {
	e := newEnv(t)
	id, qs := e.openQuiz()
	if st, b := e.raw(e.Stud, "GET", e.apath(id)+"/me", nil); st != 404 {
		t.Fatalf("none %d %s", st, b)
	}
	e.call(e.Stud, "POST", e.apath(id), nil)
	e.call(e.Stud, "PUT", e.apath(id)+"/me", map[string]any{"submit": true, "answers": map[string]any{qs[0]: map[string]any{"choices": []int{0}}}})
	st, b := e.call(e.Stud, "GET", e.apath(id)+"/me", nil)
	m := jmap(t, b)
	if st != 200 || m["userId"] != e.UserID["stud@a.edu"] || m["submittedAt"] == nil {
		t.Fatalf("me %d %s", st, b)
	}
	for _, k := range []string{"grading", "score", "feedback"} {
		if _, ok := m[k]; ok {
			t.Fatalf("%s leaked unpublished: %s", k, b)
		}
	}
	// staff see results of any user
	st, b = e.call(e.Inst, "GET", e.apath(id)+"/"+e.UserID["stud@a.edu"], nil)
	m = jmap(t, b)
	if st != 200 || m["score"] != 2.0 || m["grading"] == nil || m["feedback"] != "" {
		t.Fatalf("staff %d %s", st, b)
	}
	if st, _ := e.call(e.Inst, "GET", e.apath(id)+"/not-a-uuid", nil); st != 404 {
		t.Fatalf("staff bad id %d", st)
	}
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`update assessments set published = true where id = $1`, id)
		return err
	})
	st, b = e.call(e.Stud, "GET", e.apath(id)+"/me", nil)
	m = jmap(t, b)
	if st != 200 || m["score"] != 2.0 || m["grading"] == nil || m["feedback"] != "" {
		t.Fatalf("published %d %s", st, b)
	}
	// a started but unsubmitted attempt shows no results even when published
	id2, _ := e.openQuiz()
	e.call(e.Stud2, "POST", e.apath(id2), nil)
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`update assessments set published = true where id = $1`, id2)
		return err
	})
	_, b = e.call(e.Stud2, "GET", e.apath(id2)+"/me", nil)
	if m := jmap(t, b); m["grading"] != nil || m["score"] != nil {
		t.Fatalf("unsubmitted shows results: %s", b)
	}
	// another student's attempt, and a non-student
	if st, _ := e.call(e.Stud2, "GET", e.apath(id)+"/"+e.UserID["stud@a.edu"], nil); st != 403 {
		t.Fatalf("other student %d", st)
	}
	if st, _ := e.call(e.Other, "GET", e.apath(id)+"/me", nil); st != 403 {
		t.Fatalf("other %d", st)
	}
}
