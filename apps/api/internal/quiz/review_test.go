package quiz

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	"github.com/kapildev333/unicourse/apps/api/internal/pg"
)

// submitted starts and submits stud's attempt with both answers right (5.25) and returns the quiz and question ids.
func (e *env) submitted(who string) (id string, qs []string) {
	e.T.Helper()
	id, qs = e.openQuiz()
	e.submitFor(who, id, qs)
	return id, qs
}

func (e *env) submitFor(who, id string, qs []string) {
	e.T.Helper()
	c := e.as(who)
	e.call(c, "POST", e.apath(id), nil)
	if st, b := e.raw(c, "PUT", e.apath(id)+"/me", map[string]any{"submit": true, "answers": map[string]any{
		qs[0]: map[string]any{"choices": []int{0}}, qs[1]: map[string]any{"text": "Paris"}}}); st != 200 {
		e.T.Fatalf("submit %d %s", st, b)
	}
}

func (e *env) review(id, who string) string { return e.apath(id) + "/" + e.UserID[who+"@a.edu"] }

func (e *env) publish(id string) {
	e.T.Helper()
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`update assessments set published = true where id = $1`, id)
		return err
	})
}

func TestReviewRecompute(t *testing.T) {
	e := newEnv(t)
	id, qs := e.submitted("stud")
	e.publish(id)
	st, b := e.call(e.Inst, "PATCH", e.review(id, "stud"), map[string]any{
		"grading":  map[string]any{qs[0]: map[string]any{"points": 1, "feedback": "half"}},
		"feedback": "  Good work\nkeep going  ",
	})
	if st != 200 {
		t.Fatalf("review %d %s", st, b)
	}
	m := jmap(t, b)
	g := m["grading"].(map[string]any)
	c1, b1 := g[qs[0]].(map[string]any), g[qs[1]].(map[string]any)
	if m["score"].(float64) != 4.25 || m["feedback"] != "Good work\nkeep going" || m["reviewedAt"] == nil {
		t.Fatalf("shape %s", b)
	}
	if c1["points"].(float64) != 1 || c1["correct"] != false || c1["feedback"] != "half" || b1["points"].(float64) != 3.25 || b1["correct"] != true {
		t.Fatalf("grading %s", b)
	}
	var score, cell, by, revBy string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		if err := tx.QueryRow(`select score::text, reviewed_by::text from attempts where assessment_id = $1`, id).Scan(&score, &revBy); err != nil {
			return err
		}
		return tx.QueryRow(`select points::text, graded_by::text from scores where assessment_id = $1 and user_id = $2`, id, e.UserID["stud@a.edu"]).Scan(&cell, &by)
	})
	if score != "4.25" || cell != "4.25" || by != e.UserID["inst@a.edu"] || revBy != by {
		t.Fatalf("score %s cell %s by %s reviewer %s", score, cell, by, revBy)
	}
	// full points flips correct; a later review without feedback keeps the earlier text
	st, b = e.call(e.TA, "PATCH", e.review(id, "stud"), map[string]any{"grading": map[string]any{qs[0]: map[string]any{"points": 2}}})
	m = jmap(t, b)
	if st != 200 || m["score"].(float64) != 5.25 || m["grading"].(map[string]any)[qs[0]].(map[string]any)["correct"] != true || m["feedback"] != "Good work\nkeep going" {
		t.Fatalf("second %d %s", st, b)
	}
	// the published student sees the new points and feedback
	st, b = e.call(e.Stud, "GET", e.review(id, "stud"), nil)
	m = jmap(t, b)
	if st != 200 || m["score"].(float64) != 5.25 || m["feedback"] != "Good work\nkeep going" || !strings.Contains(string(b), "half") {
		t.Fatalf("student %d %s", st, b)
	}
	// feedback alone, and clearing it
	if st, b = e.call(e.Admin, "PATCH", e.review(id, "stud"), map[string]any{"feedback": ""}); st != 200 || jmap(t, b)["feedback"] != "" {
		t.Fatalf("clear %d %s", st, b)
	}
}

func TestReviewBounds(t *testing.T) {
	e := newEnv(t)
	id, qs := e.submitted("stud")
	p := e.review(id, "stud")
	cases := map[string]map[string]any{
		"grading." + qs[0] + ".points":   {"grading": map[string]any{qs[0]: map[string]any{"points": 2.5}}},
		"grading." + qs[1] + ".points":   {"grading": map[string]any{qs[1]: map[string]any{"points": -1}}},
		"grading.nope":                   {"grading": map[string]any{"nope": map[string]any{"points": 1}}},
		"feedback":                       {"feedback": strings.Repeat("x", 5001)},
		"grading." + qs[0] + ".feedbak":  {"grading": map[string]any{qs[0]: map[string]any{"points": 1, "feedbak": "typo"}}},
		"grading." + qs[0] + ".feedback": {"grading": map[string]any{qs[0]: map[string]any{"points": 1, "feedback": strings.Repeat("x", 2001)}}},
	}
	cases["feedback"] = map[string]any{"feedback": 5}
	for field, body := range cases {
		if st, b := e.call(e.Inst, "PATCH", p, body); st != 400 {
			t.Errorf("%s: %d %s", field, st, b)
		} else if _, f := errDetails(b); len(f) != 1 || f[0] != field {
			t.Errorf("%s: fields %v", field, f)
		}
	}
	if st, b := e.raw(e.Inst, "PATCH", p, map[string]any{"grading": map[string]any{qs[0]: map[string]any{"points": 1.005}}}); st != 400 {
		t.Errorf("3 decimals %d %s", st, b)
	}
	if st, b := e.raw(e.Inst, "PATCH", p, map[string]any{}); st != 400 || !strings.Contains(b, "no fields to update") {
		t.Errorf("empty %d %s", st, b)
	}
	if st, b := e.raw(e.Inst, "PATCH", p, map[string]any{"score": 3}); st != 400 || !strings.Contains(b, "unknown field") {
		t.Errorf("unknown %d %s", st, b)
	}
	var score string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select score::text from attempts where assessment_id = $1`, id).Scan(&score)
	})
	if score != "5.25" {
		t.Fatalf("changed by a rejected review: %s", score)
	}
}

func TestReviewUnsubmitted(t *testing.T) {
	e := newEnv(t)
	id, _ := e.openQuiz()
	if st, b := e.raw(e.Inst, "PATCH", e.review(id, "stud"), map[string]any{"feedback": "x"}); st != 404 {
		t.Fatalf("no attempt %d %s", st, b)
	}
	e.call(e.Stud, "POST", e.apath(id), nil)
	if st, b := e.raw(e.Inst, "PATCH", e.review(id, "stud"), map[string]any{"feedback": "x"}); st != 409 || !strings.Contains(b, "the attempt is not submitted") {
		t.Fatalf("unsubmitted %d %s", st, b)
	}
	if st, b := e.raw(e.Inst, "PATCH", e.apath(id)+"/not-a-uuid", map[string]any{"feedback": "x"}); st != 404 {
		t.Fatalf("bad id %d %s", st, b)
	}
	if st, b := e.raw(e.Stud, "PATCH", e.review(id, "stud"), map[string]any{"feedback": "x"}); st != 403 {
		t.Fatalf("student %d %s", st, b)
	}
}

func TestReviewAudit(t *testing.T) {
	e := newEnv(t)
	id, qs := e.submitted("stud")
	if st, b := e.raw(e.Inst, "PATCH", e.review(id, "stud"), map[string]any{"grading": map[string]any{qs[0]: map[string]any{"points": 0}}}); st != 200 {
		t.Fatalf("%d %s", st, b)
	}
	n, before, after := e.audit("attempt.review", id)
	if n != 1 || !strings.Contains(before.String, "5.25") || !strings.Contains(after.String, "3.25") || !strings.Contains(after.String, e.UserID["stud@a.edu"]) {
		t.Fatalf("audit %d %s -> %s", n, before.String, after.String)
	}
	var actor string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select actor_user_id::text from audit_log where action = 'attempt.review' and entity_id = $1`, id).Scan(&actor)
	})
	if actor != e.UserID["inst@a.edu"] {
		t.Fatalf("actor %s", actor)
	}
}

func TestAttemptsList(t *testing.T) {
	e := newEnv(t)
	id, qs := e.openQuiz()
	if st, b := e.raw(e.Inst, "GET", e.apath(id), nil); st != 200 || b != "{\"items\":[]}\n" && b != "{\"items\":[]}" {
		t.Fatalf("empty %d %q", st, b)
	}
	e.submitFor("stud2", id, qs)
	e.call(e.Stud, "POST", e.apath(id), nil)
	// an attempt of a user who is not enrolled (left the section) is not listed
	e.attempt(id, "other@a.edu", false)
	pg.WithPlatformTx(context.Background(), e.DB, func(tx *sql.Tx) error {
		_, err := tx.Exec(`update users set display_name = case email when 'stud@a.edu' then 'Zed' else 'Amy' end where email in ('stud@a.edu', 'stud2@a.edu')`)
		return err
	})
	st, b := e.call(e.TA, "GET", e.apath(id), nil)
	if st != 200 {
		t.Fatalf("%d %s", st, b)
	}
	items := jmap(t, b)["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("rows %s", b)
	}
	first, second := items[0].(map[string]any), items[1].(map[string]any)
	if first["email"] != "stud2@a.edu" || second["email"] != "stud@a.edu" || first["displayName"] != "Amy" || first["startedAt"] == nil {
		t.Fatalf("order %s", b)
	}
	byEmail := map[string]map[string]any{first["email"].(string): first, second["email"].(string): second}
	s2, s1 := byEmail["stud2@a.edu"], byEmail["stud@a.edu"]
	if s2["submittedAt"] == nil || s2["score"].(float64) != 5.25 || s2["autoSubmitted"] != false || s2["userId"] != e.UserID["stud2@a.edu"] {
		t.Fatalf("submitted row %v", s2)
	}
	for _, k := range []string{"submittedAt", "score", "reviewedAt"} {
		if _, ok := s1[k]; ok {
			t.Fatalf("%s on unsubmitted row %v", k, s1)
		}
	}
	for _, who := range []string{"stud", "other", "daout", "badmin"} {
		if st, b := e.raw(e.as(who), "GET", e.apath(id), nil); st != 403 {
			t.Fatalf("%s %d %s", who, st, b)
		}
	}
}

func TestReviewSetsManual(t *testing.T) {
	e := newEnv(t)
	id, qs := e.submitted("stud") // q1 2.00, q2 3.25
	manual := func() map[string]bool {
		var raw []byte
		e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
			return tx.QueryRow(`select grading from attempts where assessment_id = $1`, id).Scan(&raw)
		})
		g := map[string]Mark{}
		json.Unmarshal(raw, &g)
		return map[string]bool{qs[0]: g[qs[0]].Manual, qs[1]: g[qs[1]].Manual}
	}
	// same points + feedback: no manual
	if st, b := e.call(e.Inst, "PATCH", e.review(id, "stud"), map[string]any{"grading": map[string]any{qs[0]: map[string]any{"points": 2, "feedback": "ok"}}}); st != 200 {
		t.Fatalf("%d %s", st, b)
	}
	if m := manual(); m[qs[0]] || m[qs[1]] {
		t.Fatalf("manual after feedback only %v", m)
	}
	if st, b := e.call(e.Inst, "PATCH", e.review(id, "stud"), map[string]any{"grading": map[string]any{qs[1]: map[string]any{"points": 1}}}); st != 200 {
		t.Fatalf("%d %s", st, b)
	}
	if m := manual(); m[qs[0]] || !m[qs[1]] {
		t.Fatalf("manual after change %v", m)
	}
	// a second review with the same points keeps it
	if st, b := e.call(e.Inst, "PATCH", e.review(id, "stud"), map[string]any{"grading": map[string]any{qs[1]: map[string]any{"points": 1, "feedback": "x"}}}); st != 200 {
		t.Fatalf("%d %s", st, b)
	}
	if m := manual(); !m[qs[1]] {
		t.Fatalf("manual cleared %v", m)
	}
}
