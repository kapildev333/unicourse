package assignment

import (
	"database/sql"
	"strings"
	"testing"
)

// setLate moves the window by SQL: closes at closes, late until lateUntil ("" = none), penalty percent pct.
func (e *env) setLate(id, closes, lateUntil, pct string) {
	e.T.Helper()
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`update assessments set opens_at = now() - interval '3 hours', closes_at = now() + $2::interval,
			late_until = case when $3 = '' then null else now() + $3::interval end, late_penalty_percent = $4::numeric where id = $1`, id, closes, lateUntil, pct)
		return err
	})
}

func (e *env) lateFlag(a string) (late bool) {
	e.T.Helper()
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select late from submissions where assessment_id = $1 and user_id = $2`, a, e.UserID["stud@a.edu"]).Scan(&late)
	})
	return
}

func TestPenalise(t *testing.T) {
	for _, c := range [][3]string{{"15", "20", "12.00"}, {"12.50", "11", "11.13"}, {"0.01", "50", "0.01"}, {"17.35", "0", "17.35"}, {"20", "100", "0.00"}, {"40", "25", "30.00"}} {
		if got := penalise(c[0], c[1]); got != c[2] {
			t.Errorf("penalise(%s, %s) = %s, want %s", c[0], c[1], got, c[2])
		}
	}
}

func TestSubmissionLateWindow(t *testing.T) {
	e := newEnv(t)
	a := e.open()
	e.setLate(a, "-1 hour", "1 hour", "10")
	st, raw := e.put(e.Stud, a, "v1")
	if m := jmap(t, raw); st != 200 || m["late"] != true || m["penaltyPercent"] != 10.0 {
		t.Fatalf("put: %d %s", st, raw)
	}
	// a later write never clears late, even when the deadline moves back out
	e.setLate(a, "1 day", "2 days", "10")
	if st, raw = e.put(e.Stud, a, "v2"); st != 200 || jmap(t, raw)["late"] != true {
		t.Fatalf("replace: %d %s", st, raw)
	}
	// file add and remove mark late
	for _, op := range []string{"upload", "delete"} {
		b := e.open()
		var f string
		if op == "delete" {
			_, raw := e.upload(e.Stud, b, "a.txt", strings.NewReader("x"))
			f = fileIDs(t, raw)[0]
			e.put(e.Stud, b, "text")
			if e.lateFlag(b) {
				t.Fatalf("on-time write is late")
			}
		}
		e.setLate(b, "-1 hour", "1 hour", "10")
		if op == "upload" {
			if st, raw := e.upload(e.Stud, b, "a.txt", strings.NewReader("x")); st != 201 || jmap(t, raw)["late"] != true {
				t.Fatalf("upload: %d %s", st, raw)
			}
		} else if st, _ := e.call(e.Stud, "DELETE", e.subPath(b, "/me/files/"+f), nil); st != 204 || !e.lateFlag(b) {
			t.Fatalf("delete: %d", st)
		}
	}
}

func TestSubmissionLateAtDeadline(t *testing.T) {
	e := newEnv(t)
	a := e.open()
	e.setLate(a, "0 seconds", "1 hour", "0") // closes_at = the setup's now(), before the request's
	if st, raw := e.put(e.Stud, a, "x"); st != 200 || jmap(t, raw)["late"] != true {
		t.Fatalf("%d %s", st, raw)
	}
}

func TestSubmissionAfterLateUntil(t *testing.T) {
	e := newEnv(t)
	a := e.open()
	e.put(e.Stud, a, "x")
	_, raw := e.upload(e.Stud, a, "a.txt", strings.NewReader("x"))
	f := fileIDs(t, raw)[0]
	e.setLate(a, "-2 hours", "-1 hour", "10")
	if st, raw := e.put(e.Stud, a, "y"); st != 409 || !strings.Contains(string(raw), "closed") {
		t.Errorf("put: %d %s", st, raw)
	}
	if st, raw := e.upload(e.Stud, a, "b.txt", strings.NewReader("x")); st != 409 || !strings.Contains(string(raw), "closed") {
		t.Errorf("upload: %d %s", st, raw)
	}
	if st, raw := e.call(e.Stud, "DELETE", e.subPath(a, "/me/files/"+f), nil); st != 409 || !strings.Contains(string(raw), "closed") {
		t.Errorf("delete: %d %s", st, raw)
	}
	if e.lateFlag(a) {
		t.Error("rejected writes marked late")
	}
}

func TestSubmissionNoLateWindow(t *testing.T) {
	e := newEnv(t)
	a := e.open()
	e.setLate(a, "-1 hour", "", "0")
	if st, raw := e.put(e.Stud, a, "x"); st != 409 || !strings.Contains(string(raw), "closed") {
		t.Fatalf("%d %s", st, raw)
	}
	b := e.assignment("1 hour", "2 hours")
	if st, raw := e.put(e.Stud, b, "x"); st != 409 || !strings.Contains(string(raw), "not open yet") {
		t.Fatalf("%d %s", st, raw)
	}
}

func TestSubmissionOnTimeNotLate(t *testing.T) {
	e := newEnv(t)
	a := e.open()
	e.setLate(a, "1 hour", "2 hours", "10")
	if st, raw := e.put(e.Stud, a, "x"); st != 200 || jmap(t, raw)["late"] != false {
		t.Fatalf("%d %s", st, raw)
	}
}

func TestGradeLate(t *testing.T) {
	for _, c := range []struct {
		pct, score string
		want       string
		final      float64
	}{{"20", "15", "12.00", 12}, {"11", "12.50", "11.13", 11.13}, {"0", "15", "15.00", 15}} {
		e := newEnv(t)
		a := e.open()
		e.setLate(a, "1 hour", "2 hours", c.pct)
		stud := e.UserID["stud@a.edu"]
		e.put(e.Stud, a, "x")
		e.setLate(a, "-1 hour", "1 hour", c.pct)
		e.put(e.Stud, a, "y")
		e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
			_, err := tx.Exec(`update assessments set published = true where id = $1`, a)
			return err
		})
		st, raw := e.call(e.Inst, "PATCH", e.subPath(a, "/"+stud), map[string]any{"score": num(c.score)})
		m := jmap(t, raw)
		var pts, rawScore string
		e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
			return tx.QueryRow(`select s.points::text, b.score::text from scores s, submissions b where s.assessment_id = $1 and s.user_id = $2 and b.assessment_id = s.assessment_id and b.user_id = s.user_id`, a, stud).Scan(&pts, &rawScore)
		})
		if st != 200 || pts != c.want || m["finalScore"] != c.final || m["late"] != true || rawScore != fixed2(c.score) {
			t.Fatalf("%s%%: %d %s pts %s raw %s", c.pct, st, raw, pts, rawScore)
		}
		n, _, after := e.audit("submission.grade", a)
		if n != 1 || !strings.Contains(strings.ReplaceAll(after.String, " ", ""), `"finalScore":`+c.want) || !strings.Contains(strings.ReplaceAll(after.String, " ", ""), `"score":`+fixed2(c.score)) {
			t.Fatalf("audit %d %s", n, after.String)
		}
		// the student sees finalScore only when published
		_, raw = e.call(e.Stud, "GET", e.subPath(a, "/me"), nil)
		if m = jmap(t, raw); m["finalScore"] != c.final || m["late"] != true || m["penaltyPercent"] == nil {
			t.Fatalf("student published: %s", raw)
		}
		e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
			_, err := tx.Exec(`update assessments set published = false where id = $1`, a)
			return err
		})
		_, raw = e.call(e.Stud, "GET", e.subPath(a, "/me"), nil)
		if m = jmap(t, raw); m["finalScore"] != nil || m["score"] != nil || m["late"] != true || m["penaltyPercent"] == nil {
			t.Fatalf("student unpublished: %s", raw)
		}
		// the staff list carries late and finalScore
		_, raw = e.call(e.Inst, "GET", e.subPath(a, ""), nil)
		row := jmap(t, raw)["items"].([]any)[0].(map[string]any)
		if row["late"] != true || row["finalScore"] != c.final {
			t.Fatalf("list row: %s", raw)
		}
	}
}

func TestGradeOnTime(t *testing.T) {
	e := newEnv(t)
	a := e.open()
	e.setLate(a, "1 hour", "2 hours", "20")
	stud := e.UserID["stud@a.edu"]
	e.put(e.Stud, a, "x")
	st, raw := e.call(e.Inst, "PATCH", e.subPath(a, "/"+stud), map[string]any{"score": 15})
	var pts string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select points::text from scores where assessment_id = $1`, a).Scan(&pts)
	})
	if m := jmap(t, raw); st != 200 || pts != "15.00" || m["late"] != false || m["finalScore"] != 15.0 {
		t.Fatalf("%d %s pts %s", st, raw, pts)
	}
	_, raw = e.call(e.Inst, "GET", e.subPath(a, ""), nil)
	if row := jmap(t, raw)["items"].([]any)[0].(map[string]any); row["late"] != false {
		t.Fatalf("list: %s", raw)
	}
}

func TestGradePercentChangedLater(t *testing.T) {
	e := newEnv(t)
	a := e.open()
	stud := e.UserID["stud@a.edu"]
	e.setLate(a, "-1 hour", "1 hour", "20")
	e.put(e.Stud, a, "x")
	path := e.subPath(a, "/"+stud)
	e.call(e.Inst, "PATCH", path, map[string]any{"score": 15})
	e.setLate(a, "-1 hour", "1 hour", "50")
	pts := func() (p string) {
		e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
			return tx.QueryRow(`select points::text from scores where assessment_id = $1`, a).Scan(&p)
		})
		return
	}
	if p := pts(); p != "12.00" {
		t.Fatalf("changed before regrade: %s", p)
	}
	e.call(e.Inst, "PATCH", path, map[string]any{"score": 15})
	if p := pts(); p != "7.50" {
		t.Fatalf("after regrade: %s", p)
	}
}

func num(s string) rawNum { return rawNum(s) }

// rawNum marshals as a bare JSON number.
type rawNum string

func (n rawNum) MarshalJSON() ([]byte, error) { return []byte(n), nil }

// fixed2 renders a decimal with two places, as numeric(…,2)::text does.
func fixed2(s string) string {
	if !strings.Contains(s, ".") {
		return s + ".00"
	}
	if len(s)-strings.Index(s, ".") == 2 {
		return s + "0"
	}
	return s
}
