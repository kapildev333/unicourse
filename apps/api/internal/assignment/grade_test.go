package assignment

import (
	"database/sql"
	"strings"
	"testing"
)

func TestGradeSubmission(t *testing.T) {
	e := newEnv(t)
	a := e.open()
	stud := e.UserID["stud@a.edu"]
	path := e.subPath(a, "/"+stud)
	// no submission yet
	if st, _ := e.call(e.TA, "PATCH", path, map[string]any{"score": 1}); st != 404 {
		t.Fatalf("no submission: %d", st)
	}
	e.put(e.Stud, a, "work")
	st, raw := e.call(e.TA, "PATCH", path, map[string]any{"score": 87.5, "feedback": "  nice  "})
	m := jmap(t, raw)
	if st != 200 || m["score"] != 87.5 || m["feedback"] != "nice" || m["gradedAt"] == nil || m["body"] != "work" {
		t.Fatalf("grade: %d %s", st, raw)
	}
	var pts, by string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select points::text, graded_by::text from scores where assessment_id = $1 and user_id = $2`, a, stud).Scan(&pts, &by)
	})
	if pts != "87.50" || by != e.UserID["ta@a.edu"] {
		t.Fatalf("scores row %s by %s", pts, by)
	}
	// regrade by the instructor after the close; feedback absent keeps the text; the scores row follows
	e.setWindow(a, "-2 hours", "-1 hour")
	st, raw = e.call(e.Inst, "PATCH", path, map[string]any{"score": 90})
	m = jmap(t, raw)
	if st != 200 || m["score"] != 90.0 || m["feedback"] != "nice" {
		t.Fatalf("regrade: %d %s", st, raw)
	}
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select points::text, graded_by::text from scores where assessment_id = $1 and user_id = $2`, a, stud).Scan(&pts, &by)
	})
	if pts != "90.00" || by != e.UserID["inst@a.edu"] || e.count(`select count(*) from scores`) != 1 {
		t.Fatalf("scores row %s by %s", pts, by)
	}
	// the student replacing a graded submission keeps the grade (design decision 12), within the window
	b := e.open()
	e.put(e.Stud, b, "v1")
	e.call(e.TA, "PATCH", e.subPath(b, "/"+stud), map[string]any{"score": 10})
	e.put(e.Stud, b, "v2")
	_, raw = e.call(e.TA, "GET", e.subPath(b, "/"+stud), nil)
	if m = jmap(t, raw); m["score"] != 10.0 || m["body"] != "v2" {
		t.Fatalf("replace after grade: %s", raw)
	}
}

func TestGradeBonus(t *testing.T) {
	e := newEnv(t) // max points are 200
	a := e.open()
	e.put(e.Stud, a, "work")
	path := e.subPath(a, "/"+e.UserID["stud@a.edu"])
	for _, v := range []any{0, 400, 400.00, 199.99} {
		if st, raw := e.call(e.Inst, "PATCH", path, map[string]any{"score": v}); st != 200 {
			t.Errorf("%v: %d %s", v, st, raw)
		}
	}
	for name, body := range map[string]string{
		"above":      `{"score": 400.01}`,
		"way above":  `{"score": 401}`,
		"negative":   `{"score": -1}`,
		"3 decimals": `{"score": 1.001}`,
		"string":     `{"score": "abc"}`,
		"missing":    `{"feedback": "x"}`,
		"unknown":    `{"score": 1, "kind": "x"}`,
	} {
		st, raw := e.rawJSON(e.Inst, "PATCH", e.subPath(a, "/"+e.UserID["stud@a.edu"]), body)
		if st != 400 {
			t.Errorf("%s: %d %s", name, st, raw)
		}
		if name != "unknown" && name != "string" {
			if code, f := errDetails(raw); code != "validation_failed" || len(f) != 1 || f[0] != "score" {
				t.Errorf("%s: %s", name, raw)
			}
		}
	}
	// feedback: 5000 ok, 5001 rejected, null rejected
	if st, raw := e.call(e.Inst, "PATCH", path, map[string]any{"score": 1, "feedback": strings.Repeat("é", 5000)}); st != 200 {
		t.Errorf("5000: %d %s", st, raw)
	}
	for _, body := range []string{`{"score":1,"feedback":"` + strings.Repeat("é", 5001) + `"}`, `{"score":1,"feedback":null}`} {
		st, raw := e.rawJSON(e.Inst, "PATCH", path, body)
		if _, f := errDetails(raw); st != 400 || len(f) != 1 || f[0] != "feedback" {
			t.Errorf("feedback: %d %s", st, raw)
		}
	}
	// nothing was written by the rejected requests
	if got := e.count(`select count(*) from submissions where score = 1 and char_length(feedback) = 5000`); got != 1 {
		t.Errorf("state: %d", got)
	}
}

func TestGradeAudit(t *testing.T) {
	e := newEnv(t)
	a := e.open()
	e.put(e.Stud, a, "work")
	stud := e.UserID["stud@a.edu"]
	e.call(e.TA, "PATCH", e.subPath(a, "/"+stud), map[string]any{"score": 50})
	e.call(e.Inst, "PATCH", e.subPath(a, "/"+stud), map[string]any{"score": 60.5})
	e.call(e.Inst, "PATCH", e.subPath(a, "/"+stud), map[string]any{"score": 999}) // rejected: no row
	var n int
	var before, after, actor string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		if err := tx.QueryRow(`select count(*) from audit_log where action = 'submission.grade' and entity_id = $1 and entity_type = 'assessment'`, a).Scan(&n); err != nil {
			return err
		}
		return tx.QueryRow(`select before::text, after::text, actor_user_id::text from audit_log where action = 'submission.grade' and entity_id = $1 order by created_at desc, id desc limit 1`, a).Scan(&before, &after, &actor)
	})
	norm := func(s string) string { return strings.ReplaceAll(s, " ", "") }
	if n != 2 || norm(before) != `{"score":50.00,"userId":"`+stud+`"}` || norm(after) != `{"score":60.50,"userId":"`+stud+`"}` || actor != e.UserID["inst@a.edu"] {
		t.Fatalf("n %d before %s after %s actor %s", n, before, after, actor)
	}
}
