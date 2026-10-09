package quiz

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"reflect"
	"strings"
	"testing"
	"time"
)

// rgItems are the openQuiz questions with ids; the callbacks may change them before the PUT.
func rgItems(qs []string, choiceKey bool, blank string, p1, p2 any) []qm {
	return []qm{
		{"id": qs[0], "kind": "choice", "prompt": "Pick", "points": p1, "choices": []qm{{"text": "A", "correct": choiceKey}, {"text": "B", "correct": !choiceKey}}},
		{"id": qs[1], "kind": "blank", "prompt": "Capital of France", "points": p2, "answers": []string{blank}},
	}
}

func (e *env) putRG(id string, items any) (int, string) {
	return e.raw(e.Admin, "PUT", e.qpath(id), qm{"items": items})
}

// submitWrong submits who's attempt with q1 right (A) and q2 wrong.
func (e *env) submitWrong(who, id string, qs []string) {
	e.T.Helper()
	c := e.as(who)
	e.call(c, "POST", e.apath(id), nil)
	if st, b := e.raw(c, "PUT", e.apath(id)+"/me", qm{"submit": true, "answers": qm{qs[0]: qm{"choices": []int{0}}, qs[1]: qm{"text": "Rome"}}}); st != 200 {
		e.T.Fatalf("submit %d %s", st, b)
	}
}

func (e *env) attemptRow(id, who string) (score, grading string, scoreCell, gradedBy sql.NullString) {
	e.T.Helper()
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		if err := tx.QueryRow(`select coalesce(score::text, 'null'), grading::text from attempts where assessment_id = $1 and user_id = $2`, id, e.UserID[who+"@a.edu"]).Scan(&score, &grading); err != nil {
			return err
		}
		return tx.QueryRow(`select max(points::text), max(graded_by::text) from scores where assessment_id = $1 and user_id = $2`, id, e.UserID[who+"@a.edu"]).Scan(&scoreCell, &gradedBy)
	})
	return
}

func TestRegradeKeyFlip(t *testing.T) {
	e := newEnv(t)
	id, qs := e.submitted("stud")  // both right: 5.25
	e.submitWrong("stud2", id, qs) // q1 right, q2 wrong: 2.00
	e.attempt(id, "other@a.edu", false)
	st, b := e.putRG(id, rgItems(qs, true, "Rome", 2, 3.25))
	if st != 200 {
		t.Fatalf("%d %s", st, b)
	}
	s1, g1, c1, by := e.attemptRow(id, "stud")
	s2, _, c2, _ := e.attemptRow(id, "stud2")
	if s1 != "2.00" || s2 != "5.25" || c1.String != "2.00" || c2.String != "5.25" || by.String != e.UserID["admin@a.edu"] {
		t.Fatalf("scores %s %s cells %v %v by %v", s1, s2, c1, c2, by)
	}
	if !strings.Contains(g1, `"correct": false`) {
		t.Errorf("grading %s", g1)
	}
	if s, g, c, _ := e.attemptRow(id, "other"); s != "null" || g != "{}" || c.Valid {
		t.Errorf("in progress touched %s %s %v", s, g, c)
	}
	if n, bf, af := e.audit("attempt.regrade", id); n != 2 || !strings.Contains(bf.String, `"userId"`) || !strings.Contains(af.String, `"score"`) {
		t.Errorf("regrade audit %d %v %v", n, bf, af)
	}
	if n, _, _ := e.audit("question.update", id); n != 2 { // the setup PUT and this one
		t.Errorf("question.update %d", n)
	}
}

func TestRegradeKeepsManual(t *testing.T) {
	e := newEnv(t)
	id, qs := e.submitted("stud")
	if st, b := e.call(e.Inst, "PATCH", e.review(id, "stud"), qm{"grading": qm{qs[1]: qm{"points": 1.5, "feedback": "partly"}, qs[0]: qm{"points": 2, "feedback": "nice"}}}); st != 200 {
		t.Fatalf("review %d %s", st, b)
	}
	if st, b := e.putRG(id, rgItems(qs, false, "Rome", 2, 3.25)); st != 200 {
		t.Fatalf("%d %s", st, b)
	}
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		var raw []byte
		var score string
		if err := tx.QueryRow(`select grading, score::text from attempts where assessment_id = $1`, id).Scan(&raw, &score); err != nil {
			return err
		}
		g := map[string]Mark{}
		json.Unmarshal(raw, &g)
		if g[qs[1]].Points != "1.50" || g[qs[1]].Feedback != "partly" || !g[qs[1]].Manual {
			t.Errorf("manual lost %+v", g[qs[1]])
		}
		if g[qs[0]].Points != "0.00" || g[qs[0]].Feedback != "nice" || g[qs[0]].Manual || g[qs[0]].Correct {
			t.Errorf("q1 %+v", g[qs[0]])
		}
		if score != "1.50" {
			t.Errorf("score %s", score)
		}
		return nil
	})
}

func TestRegradeNoChangeNoAudit(t *testing.T) {
	e := newEnv(t)
	id, qs := e.submitted("stud")
	if st, b := e.putRG(id, rgItems(qs, true, "Paris", 2, 3.25)); st != 200 {
		t.Fatalf("%d %s", st, b)
	}
	if n, _, _ := e.audit("attempt.regrade", id); n != 0 {
		t.Errorf("audit %d", n)
	}
	if n, _, _ := e.audit("question.update", id); n != 2 {
		t.Errorf("question.update %d", n)
	}
}

func TestRegradePoints(t *testing.T) {
	e := newEnv(t)
	id, qs := e.submitted("stud")
	if st, b := e.putRG(id, rgItems(qs, true, "Paris", 4, 3.25)); st != 200 {
		t.Fatalf("%d %s", st, b)
	}
	if s, _, c, _ := e.attemptRow(id, "stud"); s != "7.25" || c.String != "7.25" {
		t.Errorf("score %s %v", s, c)
	}
	if m := e.maxPoints(id); m != "7.25" {
		t.Errorf("max %s", m)
	}
}

func TestRegradeIDsKept(t *testing.T) {
	e := newEnv(t)
	id, qs := e.submitted("stud")
	st, b := e.putRG(id, rgItems(qs, true, "Paris", 3, 1))
	var res qres
	json.Unmarshal([]byte(b), &res)
	if st != 200 || len(res.Items) != 2 || res.Items[0].ID != qs[0] || res.Items[1].ID != qs[1] || res.Items[0].Position != 1 {
		t.Fatalf("%d %s", st, b)
	}
}

func TestRegradeStructureLocked(t *testing.T) {
	e := newEnv(t)
	id, qs := e.submitted("stud")
	other := e.quiz("-1 hour", "1 hour")
	e.putQuestions(other, []qm{blankQ("1", "a")})
	var foreign string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select id from questions where assessment_id = $1`, other).Scan(&foreign)
	})
	base := func() []qm { return rgItems(qs, true, "Paris", 2, 3.25) }
	cases := map[string]func() []qm{
		"more": func() []qm {
			return append(base(), qm{"id": foreign, "kind": "blank", "prompt": "x", "points": 1, "answers": []string{"a"}})
		},
		"fewer":   func() []qm { return base()[:1] },
		"swapped": func() []qm { i := base(); i[0]["id"], i[1]["id"] = qs[1], qs[0]; return i },
		"kind": func() []qm {
			i := base()
			i[1] = qm{"id": qs[1], "kind": "choice", "prompt": "x", "points": 1, "choices": []qm{{"text": "a", "correct": true}, {"text": "b"}}}
			return i
		},
		"choice added": func() []qm {
			i := base()
			i[0]["choices"] = []qm{{"text": "A", "correct": true}, {"text": "B"}, {"text": "C"}}
			return i
		},
		"random uuid": func() []qm { i := base(); i[1]["id"] = "2b0a6a9e-3c2f-4a53-8f2d-6d1c1f0f9b11"; return i },
		"foreign id":  func() []qm { i := base(); i[1]["id"] = foreign; return i },
	}
	for name, f := range cases {
		st, b := e.putRG(id, f())
		if st != 409 || !strings.Contains(b, "questions are locked once a student has started") {
			t.Errorf("%s: %d %s", name, st, b)
		}
	}
}

func TestRegradeMissingID(t *testing.T) {
	e := newEnv(t)
	id, qs := e.submitted("stud")
	items := rgItems(qs, true, "Paris", 2, 3.25)
	delete(items[1], "id")
	st, b := e.putRG(id, items)
	if c, f := errDetails([]byte(b)); st != 400 || c != "validation_failed" || !reflect.DeepEqual(f, []string{"items[1].id"}) {
		t.Fatalf("%d %s", st, b)
	}
	// also a malformed id: fails like an unknown one
	items[1]["id"] = "nope"
	if st, b := e.putRG(id, items); st != 409 {
		t.Errorf("malformed %d %s", st, b)
	}
}

func TestRegradeValidationFirst(t *testing.T) {
	e := newEnv(t)
	id, qs := e.submitted("stud")
	items := rgItems(qs, true, "Paris", 0, 3.25)
	items[0]["id"], items[1]["id"] = qs[1], qs[0]
	st, b := e.putRG(id, items)
	if c, f := errDetails([]byte(b)); st != 400 || c != "validation_failed" || !reflect.DeepEqual(f, []string{"items[0].points"}) {
		t.Fatalf("%d %s", st, b)
	}
}

// TestRegradeCapsManual: a reviewed 1.5 on a 3.25-point question, the question lowered to 1 -> 1.00, correct, manual; raised leaves it.
func TestRegradeCapsManual(t *testing.T) {
	e := newEnv(t)
	id, qs := e.submitted("stud")
	if st, b := e.call(e.Inst, "PATCH", e.review(id, "stud"), qm{"grading": qm{qs[1]: qm{"points": 1.5}}}); st != 200 {
		t.Fatalf("review %d %s", st, b)
	}
	check := func(want string, correct bool, score string) {
		t.Helper()
		_, g, cell, _ := e.attemptRow(id, "stud")
		m := map[string]Mark{}
		json.Unmarshal([]byte(g), &m)
		if string(m[qs[1]].Points) != want || m[qs[1]].Correct != correct || !m[qs[1]].Manual || cell.String != score {
			t.Fatalf("mark %+v cell %v, want %s %v %s", m[qs[1]], cell, want, correct, score)
		}
	}
	if st, b := e.putRG(id, rgItems(qs, true, "Paris", 2, 1)); st != 200 {
		t.Fatalf("%d %s", st, b)
	}
	check("1.00", true, "3.00")
	if st, b := e.putRG(id, rgItems(qs, true, "Paris", 2, 5)); st != 200 {
		t.Fatalf("%d %s", st, b)
	}
	check("1.00", true, "3.00") // raised again: the capped value stays (the reviewed 1.5 is gone)
}

// TestRegradeWaitsForSubmit: a transaction holds an attempt's lock; the regrade waits for it, and the held
// submit finishes before the keys change, so the regrade then regrades it.
func TestRegradeWaitsForSubmit(t *testing.T) {
	e := newEnv(t)
	id, qs := e.openQuiz()
	c := e.as("stud")
	e.call(c, "POST", e.apath(id), nil)
	if st, b := e.raw(c, "PUT", e.apath(id)+"/me", qm{"answers": qm{qs[0]: qm{"choices": []int{0}}, qs[1]: qm{"text": "Paris"}}}); st != 200 {
		t.Fatalf("answers %d %s", st, b)
	}
	locked, release, held := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		held <- pg.WithTx(context.Background(), e.DB, e.Fx.UniversityA, func(tx *sql.Tx) error {
			if _, err := tx.Exec(`select 1 from attempts where assessment_id = $1 for update`, id); err != nil {
				return err
			}
			close(locked)
			<-release
			return submit(context.Background(), tx, e.Fx.UniversityA, id, e.UserID["stud@a.edu"], false)
		})
	}()
	<-locked
	done := make(chan int, 1)
	go func() {
		st, _ := e.putRG(id, rgItems(qs, false, "Rome", 2, 3.25))
		done <- st
	}()
	select {
	case st := <-done:
		t.Fatalf("regrade did not wait: %d", st)
	case <-time.After(500 * time.Millisecond):
	}
	close(release)
	if err := <-held; err != nil {
		t.Fatal(err)
	}
	if st := <-done; st != 200 {
		t.Fatalf("regrade %d", st)
	}
	if s, _, cell, _ := e.attemptRow(id, "stud"); s != "0.00" || cell.String != "0.00" {
		t.Fatalf("score %s cell %v (new keys: both wrong)", s, cell)
	}
}
