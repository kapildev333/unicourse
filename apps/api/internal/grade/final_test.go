package grade

import (
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
)

type fgRow struct {
	UserID, Email, Status, Letter, OverrideReason string
	Percent, Points, Credits                      json.Number
}

func (e *env) finals(c string) []fgRow {
	e.T.Helper()
	st, body := e.raw(e.Admin, "GET", "/sections/"+e.SectionA+"/final-grades", nil)
	if st != 200 {
		e.T.Fatal(st, body)
	}
	var l struct{ Items []fgRow }
	d := json.NewDecoder(strings.NewReader(body))
	d.UseNumber()
	if err := d.Decode(&l); err != nil {
		e.T.Fatal(err)
	}
	return l.Items
}

func (e *env) finalize(body any) (int, string) {
	e.T.Helper()
	return e.raw(e.Admin, "POST", "/sections/"+e.SectionA+"/final-grades", body)
}

// twoAssessments: 40 + 60; Stud scored only the 40 (80), Stud2 80 and 90.
func (e *env) twoAssessments() (mid, fin string) {
	mid, fin = e.assessment("Midterm", "100", "40"), e.assessment("Final", "100", "60")
	e.score(mid, "stud@a.edu", "80")
	e.score(mid, "stud2@a.edu", "80")
	e.score(fin, "stud2@a.edu", "90")
	return
}

func (e *env) countFinals() (n int) {
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error { return tx.QueryRow(`select count(*) from final_grades`).Scan(&n) })
	return
}

func TestFinalizeWeights(t *testing.T) {
	e := newEnv(t)
	st, body := e.finalize(map[string]any{})
	if code, f := errDetails([]byte(body)); st != 400 || code != "validation_failed" || len(f) != 1 || f[0] != "weights" || !strings.Contains(body, "is 0.00") {
		t.Errorf("no assessments: %d %s", st, body)
	}
	e.assessment("A", "100", "40")
	e.assessment("B", "100", "50")
	if st, body := e.finalize(map[string]any{}); st != 400 || !strings.Contains(body, "must sum to 100.00 (is 90.00)") {
		t.Errorf("90: %d %s", st, body)
	}
	e.assessment("C", "100", "10")
	e.assessment("Zero", "100", "0")
	if st, body := e.finalize(map[string]any{}); st != 201 {
		t.Errorf("100 plus a 0-weight: %d %s", st, body)
	}
}

func TestFinalize(t *testing.T) {
	e := newEnv(t)
	mid, fin := e.twoAssessments()
	st, body := e.finalize(map[string]any{})
	if st != 201 {
		t.Fatal(st, body)
	}
	got := e.finals("")
	if len(got) != 2 || got[0].Email != "stud2@a.edu" {
		t.Fatal(body)
	}
	s2, s := got[0], got[1]
	if s.Percent != "32.00" || s.Letter != "F" || s.Points != "0.00" || s.Credits != "3.0" || s.Status != "graded" {
		t.Errorf("stud: %+v", s)
	}
	if s2.Percent != "86.00" || s2.Letter != "B" || s2.Points != "3.00" {
		t.Errorf("stud2: %+v", s2)
	}
	var pub int
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select count(*) from assessments where published and id in ($1, $2)`, mid, fin).Scan(&pub)
	})
	if pub != 2 {
		t.Errorf("published %d", pub)
	}
	n, _, after := e.audit("final_grade.finalize", e.SectionA)
	if n != 1 || !strings.Contains(after.String, mid) || !strings.Contains(after.String, `"grades"`) || !strings.Contains(after.String, e.UserID["stud@a.edu"]) {
		t.Errorf("audit %d %s", n, after.String)
	}
	if st, _ := e.finalize(map[string]any{}); st != 409 {
		t.Errorf("second finalize %d", st)
	}
}

func TestFinalizeBoundaryPin(t *testing.T) {
	e := newEnv(t)
	a := e.assessment("Only", "200", "100")
	e.score(a, "stud@a.edu", "185.99")
	if st, body := e.finalize(map[string]any{}); st != 201 {
		t.Fatal(st, body)
	}
	for _, f := range e.finals("") {
		if f.Email == "stud@a.edu" && (f.Percent != "93.00" || f.Letter != "A-" || f.Points != "3.70") {
			t.Errorf("%+v", f)
		}
	}
	// 185.99 / 200 = 92.995 exactly; the stored value is rounded, the letter used the unrounded one.
}

func TestFinalizeOverrides(t *testing.T) {
	e := newEnv(t)
	e.twoAssessments()
	stud, stud2 := e.UserID["stud@a.edu"], e.UserID["stud2@a.edu"]
	fields := func(ov ...map[string]any) []string {
		st, body := e.finalize(map[string]any{"overrides": ov})
		_, f := errDetails([]byte(body))
		if st != 400 {
			t.Fatalf("want 400, got %d %s", st, body)
		}
		return f
	}
	same := func(got []string, want ...string) {
		t.Helper()
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("fields %v, want %v", got, want)
		}
	}
	same(fields(map[string]any{"userId": stud, "letter": "Q", "reason": "x"}), "overrides[0].letter")
	same(fields(map[string]any{"userId": e.UserID["other@a.edu"], "letter": "A", "reason": "x"}), "overrides[0].userId")
	same(fields(map[string]any{"userId": stud, "letter": "A"}), "overrides[0].reason")
	same(fields(map[string]any{"userId": stud, "status": "incomplete", "letter": "A", "reason": "x"}), "overrides[0].letter")
	same(fields(map[string]any{"userId": stud, "reason": "x"}), "overrides[0]")
	if n := e.countFinals(); n != 0 {
		t.Errorf("rows after 400: %d", n)
	}
	var pub int
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select count(*) from assessments where published`).Scan(&pub)
	})
	if pub != 0 {
		t.Error("assessments published after a 400")
	}
	e.assessment("Extra", "10", "5") // weights now 105
	same(fields(map[string]any{"userId": stud, "letter": "Q", "reason": "x"}), "weights", "overrides[0].letter")
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error { _, err := tx.Exec(`delete from assessments where name = 'Extra'`); return err })
	st, body := e.finalize(map[string]any{"overrides": []map[string]any{
		{"userId": stud, "status": "withdrawn", "reason": "left"},
		{"userId": stud2, "letter": "A", "reason": " regrade "}}})
	if st != 201 {
		t.Fatal(st, body)
	}
	got := e.finals("")
	if got[1].Status != "withdrawn" || got[1].Percent != "" || got[1].Letter != "" || got[1].Points != "" || got[1].OverrideReason != "left" || got[1].Credits != "3.0" {
		t.Errorf("withdrawn: %+v", got[1])
	}
	if got[0].Letter != "A" || got[0].Points != "4.00" || got[0].Percent != "86.00" || got[0].OverrideReason != "regrade" {
		t.Errorf("letter override: %+v", got[0])
	}
}
