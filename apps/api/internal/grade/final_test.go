package grade

import (
	"database/sql"
	"encoding/json"
	"net/http"
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

// compact drops the spaces jsonb adds, so audit payloads compare like response bodies.
func compact(s sql.NullString) string { return strings.ReplaceAll(s.String, " ", "") }

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
	if n != 1 || !strings.Contains(after.String, mid) || !strings.Contains(after.String, `"grades"`) || !strings.Contains(after.String, e.UserID["stud@a.edu"]) ||
		!strings.Contains(compact(after), `"letter":"B"`) || !strings.Contains(compact(after), `"letter":"F"`) {
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
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`update courses set credits = 4.5 where id = $1`, e.CourseA)
		return err
	})
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
	for _, f := range got {
		if f.Credits != "4.5" {
			t.Errorf("credits %s, want the course's 4.5", f.Credits)
		}
	}
	if got[1].Status != "withdrawn" || got[1].Percent != "" || got[1].Letter != "" || got[1].Points != "" || got[1].OverrideReason != "left" {
		t.Errorf("withdrawn: %+v", got[1])
	}
	if got[0].Letter != "A" || got[0].Points != "4.00" || got[0].Percent != "86.00" || got[0].OverrideReason != "regrade" {
		t.Errorf("letter override: %+v", got[0])
	}
}

func (e *env) finalized() {
	e.T.Helper()
	e.twoAssessments()
	if st, body := e.finalize(map[string]any{}); st != 201 {
		e.T.Fatal(st, body)
	}
}

func (e *env) patchFinal(c *http.Cookie, email string, body any) (int, string) {
	return e.raw(c, "PATCH", "/sections/"+e.SectionA+"/final-grades/"+e.UserID[email], body)
}

func TestFinalizeSnapshot(t *testing.T) {
	e := newEnv(t)
	e.finalized()
	_, before := e.raw(e.Admin, "GET", "/sections/"+e.SectionA+"/final-grades", nil)
	var mid string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select id from assessments where name = 'Midterm'`).Scan(&mid)
	})
	if st, body := e.raw(e.Admin, "PUT", "/sections/"+e.SectionA+"/assessments/"+mid+"/scores", map[string]any{"scores": []map[string]any{{"userId": e.UserID["stud@a.edu"], "points": 100}}}); st != 200 {
		t.Fatal(st, body)
	}
	rows := []map[string]any{{"letter": "A", "minPercent": 95, "points": 4}, {"letter": "F", "minPercent": 0, "points": 0}}
	if st, body := e.raw(e.Admin, "PUT", "/grade-scale", map[string]any{"rows": rows}); st != 200 {
		t.Fatal(st, body)
	}
	if _, after := e.raw(e.Admin, "GET", "/sections/"+e.SectionA+"/final-grades", nil); after != before {
		t.Errorf("snapshot changed:\n%s\n%s", before, after)
	}
	if st, _ := e.raw(e.Admin, "DELETE", "/sections/"+e.SectionA+"/final-grades", nil); st != 204 {
		t.Fatal(st)
	}
	if e.countFinals() != 0 {
		t.Error("rows after reopen")
	}
	n, bef, _ := e.audit("final_grade.reopen", e.SectionA)
	if n != 1 || !strings.Contains(compact(bef), e.UserID["stud2@a.edu"]) || !strings.Contains(compact(bef), `"letter":"B"`) {
		t.Errorf("reopen audit %d %s", n, compact(bef))
	}
	if st, _ := e.raw(e.Admin, "DELETE", "/sections/"+e.SectionA+"/final-grades", nil); st != 404 {
		t.Errorf("second reopen %d", st)
	}
	if st, body := e.finalize(map[string]any{}); st != 201 {
		t.Fatal(st, body)
	}
	for _, f := range e.finals("") {
		if f.Email == "stud@a.edu" && (f.Percent != "40.00" || f.Letter != "F") { // 100 x 0.4, scale now A 95 / F 0
			t.Errorf("recomputed: %+v", f)
		}
	}
}

func TestOverridePatch(t *testing.T) {
	e := newEnv(t)
	e.finalized() // stud: 32.00 F, stud2: 86.00 B
	st, body := e.patchFinal(e.Inst, "stud2@a.edu", map[string]any{"letter": "A", "reason": "regrade"})
	if st != 200 || !strings.Contains(body, `"points":4.00`) || !strings.Contains(body, `"percent":86.00`) || !strings.Contains(body, `"overrideReason":"regrade"`) {
		t.Fatal(st, body)
	}
	var rowID string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select id from final_grades where user_id = $1`, e.UserID["stud2@a.edu"]).Scan(&rowID)
	})
	n, bef, aft := e.audit("final_grade.override", rowID)
	if n != 1 || !strings.Contains(compact(bef), `"letter":"B"`) || !strings.Contains(compact(aft), `"letter":"A"`) || !strings.Contains(compact(aft), e.SectionA) {
		t.Errorf("audit %d %s %s", n, compact(bef), compact(aft))
	}
	if st, body := e.patchFinal(e.Admin, "stud2@a.edu", map[string]any{"status": "incomplete", "reason": "doc"}); st != 200 || strings.Contains(body, "percent") || strings.Contains(body, "letter") || strings.Contains(body, "points") {
		t.Errorf("incomplete: %d %s", st, body)
	}
	if st, body := e.patchFinal(e.Admin, "stud2@a.edu", map[string]any{"letter": "B", "reason": "x"}); st != 400 || !strings.Contains(body, `"letter"`) {
		t.Errorf("letter on incomplete: %d %s", st, body)
	}
	if st, body := e.patchFinal(e.Admin, "stud2@a.edu", map[string]any{"status": "graded", "reason": "back"}); st != 200 || !strings.Contains(body, `"percent":86.00`) || !strings.Contains(body, `"letter":"B"`) || !strings.Contains(body, `"points":3.00`) {
		t.Errorf("back to graded: %d %s", st, body)
	}
	for name, c := range map[string]struct {
		body   map[string]any
		status int
		field  string
	}{
		"reason only":    {map[string]any{"reason": "x"}, 400, ""},
		"no reason":      {map[string]any{"letter": "A"}, 400, "reason"},
		"null letter":    {map[string]any{"letter": nil, "reason": "x"}, 400, "letter"},
		"unknown letter": {map[string]any{"letter": "Q", "reason": "x"}, 400, "letter"},
	} {
		st, body := e.patchFinal(e.Admin, "stud2@a.edu", c.body)
		if _, f := errDetails([]byte(body)); st != c.status || len(f) != 1 || f[0] != c.field {
			t.Errorf("%s: %d %s", name, st, body)
		}
	}
	if st, _ := e.raw(e.Admin, "PATCH", "/sections/"+e.SectionA+"/final-grades/"+e.UserID["other@a.edu"], map[string]any{"letter": "A", "reason": "x"}); st != 404 {
		t.Errorf("unknown final: %d", st)
	}
}

func TestOverridePatchConflict(t *testing.T) {
	e := newEnv(t)
	e.finalized()
	if st, body := e.patchFinal(e.Admin, "stud2@a.edu", map[string]any{"status": "withdrawn", "reason": "left"}); st != 200 {
		t.Fatal(st, body)
	}
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error { _, err := tx.Exec(`update assessments set weight = 0`); return err })
	st, body := e.patchFinal(e.Admin, "stud2@a.edu", map[string]any{"status": "graded", "reason": "back"})
	if code, _ := errDetails([]byte(body)); st != 409 || code != "conflict" || !strings.Contains(body, "weights sum to 0") {
		t.Errorf("%d %s", st, body)
	}
}

func TestFinalAuthz(t *testing.T) {
	e := newEnv(t)
	e.twoAssessments()
	sec := "/sections/" + e.SectionA + "/final-grades"
	read := map[string]int{"admin": 200, "dain": 200, "daout": 403, "inst": 200, "ta": 200, "stud": 403, "other": 403}
	write := map[string]bool{"admin": true, "dain": true, "inst": true}
	for _, role := range roles {
		if st, _ := e.raw(e.as(role), "GET", sec, nil); st != read[role] {
			t.Errorf("GET %s: %d", role, st)
		}
	}
	for _, role := range roles { // forbidden first; a permitted role then finalizes, patches and reopens in turn
		if write[role] {
			continue
		}
		for _, c := range []struct{ m, p string }{{"POST", sec}, {"DELETE", sec}, {"PATCH", sec + "/" + e.UserID["stud@a.edu"]}} {
			if st, _ := e.raw(e.as(role), c.m, c.p, map[string]any{"letter": "A", "reason": "x"}); st != 403 {
				t.Errorf("%s %s %s: %d", role, c.m, c.p, st)
			}
		}
	}
	for _, role := range []string{"admin", "dain", "inst"} {
		if st, body := e.raw(e.as(role), "POST", sec, map[string]any{}); st != 201 {
			t.Errorf("%s POST: %d %s", role, st, body)
		}
		if st, _ := e.patchFinal(e.as(role), "stud@a.edu", map[string]any{"status": "withdrawn", "reason": "x"}); st != 200 {
			t.Errorf("%s PATCH: %d", role, st)
		}
		if st, _ := e.raw(e.as(role), "DELETE", sec, nil); st != 204 {
			t.Errorf("%s DELETE: %d", role, st)
		}
	}
}

// A same-letter override keeps the recorded points even after the scale moved on (snapshot).
func TestOverrideSameLetterKeepsPoints(t *testing.T) {
	e := newEnv(t)
	e.finalized() // stud2: 86.00 B, 3.00
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`update grade_scales set points = 3.50 where letter = 'B'`)
		return err
	})
	st, body := e.patchFinal(e.Inst, "stud2@a.edu", map[string]any{"letter": "B", "reason": "confirmed"})
	if st != 200 || !strings.Contains(body, `"points":3.00`) || !strings.Contains(body, `"overrideReason":"confirmed"`) {
		t.Fatal(st, body)
	}
}

func TestFinalizeNotifies(t *testing.T) {
	e := newEnv(t)
	e.twoAssessments()
	want := []string{"stud2@a.edu grades_finalized", "stud@a.edu grades_finalized"}
	for i := 1; i <= 2; i++ {
		if st, body := e.finalize(map[string]any{}); st != 201 {
			t.Fatal(st, body)
		}
		if got := e.notes("grades_finalized"); len(got) != 2*i || got[0] != want[0] {
			t.Fatalf("round %d: %v", i, got)
		}
		if got := e.notes("scores_published"); len(got) != 0 {
			t.Fatalf("scores_published: %v", got)
		}
		if st, _ := e.raw(e.Admin, "DELETE", "/sections/"+e.SectionA+"/final-grades", nil); st != 204 {
			t.Fatal(st)
		}
	}
}
