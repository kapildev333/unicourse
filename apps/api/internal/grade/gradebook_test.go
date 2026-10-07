package grade

import (
	"database/sql"
	"encoding/json"
	"math/big"
	"strings"
	"testing"
)

func pt(w, max, p int64) part {
	return part{big.NewRat(w, 1), big.NewRat(max, 1), big.NewRat(p, 1)}
}

func TestPercentOf(t *testing.T) {
	eq := func(name string, parts []part, want int64) {
		t.Helper()
		got, ok := percentOf(parts)
		if !ok || got.Cmp(big.NewRat(want, 1)) != 0 {
			t.Errorf("%s: got %v %v, want %d", name, got, ok, want)
		}
	}
	eq("only the scored 40", []part{pt(40, 100, 80)}, 80)
	eq("both scored", []part{pt(40, 100, 80), pt(60, 100, 90)}, 86)
	eq("0-weight part", []part{pt(40, 100, 80), pt(0, 100, 0)}, 80)
	eq("bonus", []part{pt(100, 100, 150)}, 150)
	third := []part{{big.NewRat(1, 3), big.NewRat(50, 1), big.NewRat(25, 1)}, {big.NewRat(2, 3), big.NewRat(50, 1), big.NewRat(25, 1)}}
	eq("thirds", third, 50)
	if _, ok := percentOf([]part{pt(0, 100, 50)}); ok {
		t.Error("sum of weights 0 must be false")
	}
}

var testScale = func() []ScaleRow {
	var s []ScaleRow
	for _, r := range [][3]string{{"A", "93.00", "4.00"}, {"A-", "90.00", "3.70"}, {"B+", "87.00", "3.30"}, {"B", "83.00", "3.00"}, {"B-", "80.00", "2.70"},
		{"C+", "77.00", "2.30"}, {"C", "73.00", "2.00"}, {"C-", "70.00", "1.70"}, {"D+", "67.00", "1.30"}, {"D", "63.00", "1.00"}, {"D-", "60.00", "0.70"}, {"F", "0.00", "0.00"}} {
		s = append(s, ScaleRow{r[0], json.Number(r[1]), json.Number(r[2])})
	}
	return s
}()

func TestLetterBoundaries(t *testing.T) {
	r := func(s string) *big.Rat { v, _ := new(big.Rat).SetString(s); return v }
	for pct, want := range map[string]string{"92.99": "A-", "93": "A", "92.995": "A-", "59.99": "F", "60": "D-", "0": "F", "150": "A"} {
		if got, ok := letterFor(testScale, r(pct)); !ok || got.Letter != want {
			t.Errorf("%s: got %q %v, want %s", pct, got.Letter, ok, want)
		}
	}
	if _, ok := letterFor(nil, r("50")); ok {
		t.Error("empty scale must be false")
	}
	if round2(r("92.995")) != "93.00" {
		t.Error("92.995 must round to 93.00")
	}
}

func TestRound2AndStats(t *testing.T) {
	r := func(s string) *big.Rat { v, _ := new(big.Rat).SetString(s); return v }
	if round2(r("2.345")) != "2.35" || round2(r("2.344")) != "2.34" {
		t.Error("round2 half up")
	}
	run := func(p, l string) GradebookStudent { return GradebookStudent{running: r(p), Letter: l} }
	odd := stats([]GradebookStudent{run("80", "B-"), run("90", "A-"), run("100", "A")}, testScale)
	if odd.Mean != "90.00" || odd.Median != "90.00" || odd.Letters["A"] != 1 || odd.Letters["F"] != 0 || len(odd.Letters) != 12 {
		t.Errorf("odd: %+v", odd)
	}
	even := stats([]GradebookStudent{run("80", "B-"), run("91", "A-"), {Final: &FinalGrade{Status: "withdrawn"}}}, testScale)
	if even.Mean != "85.50" || even.Median != "85.50" {
		t.Errorf("even: %+v", even)
	}
	fin := stats([]GradebookStudent{{Final: &FinalGrade{Status: "graded", Percent: "70.00", Letter: "Z"}, running: r("10")}}, testScale)
	if fin.Mean != "70.00" || fin.Letters["Z"] != 1 {
		t.Errorf("final wins, own key: %+v", fin)
	}
	none := stats(nil, testScale)
	if none.Mean != "" || none.Median != "" {
		t.Errorf("empty: %+v", none)
	}
}

func TestGradebook(t *testing.T) {
	e := newEnv(t)
	mid, fin := e.assessment("Midterm", "100", "40"), e.assessment("Final", "100", "60")
	e.score(mid, "stud@a.edu", "80")
	e.score(mid, "stud2@a.edu", "80")
	e.score(fin, "stud2@a.edu", "90")
	st, body := e.raw(e.Admin, "GET", "/sections/"+e.SectionA+"/gradebook", nil)
	if st != 200 {
		t.Fatal(st, body)
	}
	var g struct {
		Assessments []Assessment
		Students    []map[string]any
		Stats       map[string]any
	}
	json.Unmarshal([]byte(body), &g)
	if len(g.Assessments) != 2 || len(g.Students) != 2 {
		t.Fatal(body)
	}
	s2, s1 := g.Students[0], g.Students[1] // by email: stud2@ sorts before stud@
	if s1["percent"] != 80.0 || s1["letter"] != "B-" || s2["percent"] != 86.0 || s2["letter"] != "B" {
		t.Errorf("percent/letter: %v %v", s1, s2)
	}
	if _, ok := s1["final"]; ok || len(s1["scores"].(map[string]any)) != 1 || len(s2["scores"].(map[string]any)) != 2 {
		t.Errorf("scores/final: %v %v", s1, s2)
	}
	for _, want := range []string{`"percent":80.00`, `"percent":86.00`, `"mean":83.00`, `"median":83.00`, `"A":0`, `"B":1`, `"B-":1`} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %s in %s", want, body)
		}
	}
}

func TestGradebookLetterUnrounded(t *testing.T) { // 185.99 / 200 = 92.995: shown as 93.00, lettered A- from the unrounded value
	e := newEnv(t)
	e.score(e.assessment("Only", "200", "100"), "stud@a.edu", "185.99")
	if _, body := e.raw(e.Admin, "GET", "/sections/"+e.SectionA+"/gradebook", nil); !strings.Contains(body, `"percent":93.00,"letter":"A-"`) {
		t.Errorf("%s", body)
	}
}

func TestGradebookAuthz(t *testing.T) {
	e := newEnv(t)
	want := map[string]int{"admin": 200, "dain": 200, "daout": 403, "inst": 200, "ta": 200, "stud": 403, "other": 403}
	for _, role := range roles {
		if st, _ := e.raw(e.as(role), "GET", "/sections/"+e.SectionA+"/gradebook", nil); st != want[role] {
			t.Errorf("%s: %d, want %d", role, st, want[role])
		}
	}
	if st, _ := e.raw(e.Admin, "GET", "/sections/"+e.Fx.UniversityB+"/gradebook", nil); st != 404 {
		t.Errorf("foreign id: %d", st)
	}
}

func TestGradebookCSV(t *testing.T) {
	e := newEnv(t)
	mid, fin := e.assessment("Midterm", "100", "40"), e.assessment("Final", "100", "60")
	e.score(mid, "stud@a.edu", "87.5")
	e.score(mid, "stud2@a.edu", "80")
	e.score(fin, "stud2@a.edu", "90")
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`update users set display_name = '=HYPERLINK("x")' where email = 'stud@a.edu'`)
		return err
	})
	path := "/sections/" + e.SectionA + "/gradebook"
	st, h, b := e.callAccept(e.Inst, path, "text/csv")
	want := "\ufeffemail,displayName,Midterm,Final,percent,letter,final\n" +
		"stud2@a.edu,N stud2@a.edu,80.00,90.00,86.00,B,\n" +
		"stud@a.edu,'=HYPERLINK(\"\"x\"\"),87.50,,87.50,B+,\n"
	got := string(b)
	if st != 200 || h.Get("Content-Type") != "text/csv; charset=utf-8" || h.Get("Content-Disposition") != `attachment; filename="gradebook.csv"` {
		t.Fatalf("%d %v %s", st, h, got)
	}
	if got != want {
		t.Fatalf("csv:\n%q\nwant:\n%q", got, want)
	}
	if st, h, b := e.callAccept(e.Inst, path, "application/json"); st != 200 || !strings.HasPrefix(h.Get("Content-Type"), "application/json") || !strings.HasPrefix(string(b), "{") {
		t.Errorf("json: %d %v", st, h)
	}
	if st, h, _ := e.callAccept(e.Inst, path, ""); st != 200 || !strings.HasPrefix(h.Get("Content-Type"), "application/json") {
		t.Errorf("no accept: %d %v", st, h)
	}
	if st, h, b := e.callAccept(e.Stud, path, "text/csv"); st != 403 || !strings.HasPrefix(h.Get("Content-Type"), "application/json") || !strings.Contains(string(b), "forbidden") {
		t.Errorf("student: %d %v %s", st, h, b)
	}
	// finalize: B column, and an override to withdrawn
	if st, body := e.raw(e.Admin, "POST", path[:len(path)-len("gradebook")]+"final-grades", map[string]any{
		"overrides": []map[string]any{{"userId": e.UserID["stud@a.edu"], "status": "withdrawn", "reason": "left"}}}); st != 201 {
		t.Fatal(st, body)
	}
	_, _, b = e.callAccept(e.Admin, path, "text/csv")
	if !strings.Contains(string(b), ",86.00,B,B\n") || !strings.Contains(string(b), ",withdrawn\n") {
		t.Errorf("final column:\n%s", b)
	}
}
