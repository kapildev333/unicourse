package grade

import (
	"database/sql"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func (e *env) aPath(id string) string { return "/sections/" + e.SectionA + "/assessments/" + id }

func TestAssessmentCRUD(t *testing.T) {
	e := newEnv(t)
	st, raw := e.raw(e.Inst, "POST", "/sections/"+e.SectionA+"/assessments", map[string]any{"name": " Midterm ", "maxPoints": 100, "weight": 40})
	m := jmap(t, []byte(raw))
	if st != 201 || m["published"] != false || m["position"] != float64(0) || m["name"] != "Midterm" || strings.Contains(raw, "dueOn") ||
		!strings.Contains(raw, `"maxPoints":100.00`) || !strings.Contains(raw, `"weight":40.00`) {
		t.Fatalf("create: %d %s", st, raw)
	}
	id := m["id"].(string)
	st, raw = e.raw(e.Inst, "POST", "/sections/"+e.SectionA+"/assessments", map[string]any{"name": "Final", "maxPoints": 50.5, "weight": 60})
	if m2 := jmap(t, []byte(raw)); st != 201 || m2["position"] != float64(1) {
		t.Fatalf("second: %d %s", st, raw)
	}
	st, raw = e.raw(e.Inst, "PATCH", e.aPath(id), map[string]any{"name": "Mid", "weight": 30, "published": true, "position": 5, "dueOn": "2026-10-20"})
	if st != 200 || !strings.Contains(raw, `"name":"Mid"`) || !strings.Contains(raw, `"weight":30.00`) || !strings.Contains(raw, `"published":true`) ||
		!strings.Contains(raw, `"position":5`) || !strings.Contains(raw, `"dueOn":"2026-10-20"`) {
		t.Fatalf("patch: %d %s", st, raw)
	}
	st, raw = e.raw(e.Inst, "PATCH", e.aPath(id), map[string]any{"dueOn": nil})
	if st != 200 || strings.Contains(raw, "dueOn") || strings.Contains(raw, "null") {
		t.Fatalf("clear dueOn: %d %s", st, raw)
	}
	if st, raw = e.raw(e.Inst, "PATCH", e.aPath(id), map[string]any{"dueOn": ""}); st != 400 || !strings.Contains(raw, `"dueOn"`) {
		t.Fatalf("empty dueOn: %d %s", st, raw)
	}
	e.score(id, "stud@a.edu", "80")
	if st, raw = e.raw(e.Inst, "DELETE", e.aPath(id), nil); st != 204 {
		t.Fatalf("delete: %d %s", st, raw)
	}
	var n int
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error { return tx.QueryRow(`select count(*) from scores`).Scan(&n) })
	if n != 0 {
		t.Fatalf("%d scores left after delete", n)
	}
	if st, _ = e.raw(e.Inst, "DELETE", e.aPath(id), nil); st != 404 {
		t.Fatalf("delete again: %d", st)
	}
}

func TestAssessmentValidation(t *testing.T) {
	e := newEnv(t)
	post := func(b map[string]any) (int, []string) {
		st, raw := e.call(e.Admin, "POST", "/sections/"+e.SectionA+"/assessments", b)
		_, f := errDetails(raw)
		return st, f
	}
	ok := func() map[string]any { return map[string]any{"name": "N", "maxPoints": 10, "weight": 10} }
	with := func(k string, v any) map[string]any { b := ok(); b[k] = v; return b }
	for name, c := range map[string]struct {
		body map[string]any
		want []string
	}{
		"name empty":   {with("name", ""), []string{"name"}},
		"name long":    {with("name", strings.Repeat("é", 101)), []string{"name"}},
		"max zero":     {with("maxPoints", 0), []string{"maxPoints"}},
		"max big":      {with("maxPoints", 10000.01), []string{"maxPoints"}},
		"max decimals": {with("maxPoints", 1.234), []string{"maxPoints"}},
		"weight neg":   {with("weight", -1), []string{"weight"}},
		"weight big":   {with("weight", 100.01), []string{"weight"}},
		"due month":    {with("dueOn", "2026-13-01"), []string{"dueOn"}},
		"due format":   {with("dueOn", "20261001"), []string{"dueOn"}},
		"position":     {with("position", -1), []string{"position"}},
		"many":         {map[string]any{"name": "", "maxPoints": 0, "weight": 101, "dueOn": "x", "position": 10001}, []string{"name", "maxPoints", "weight", "dueOn", "position"}},
	} {
		if st, f := post(c.body); st != 400 || !reflect.DeepEqual(f, c.want) {
			t.Errorf("%s: %d %v", name, st, f)
		}
	}
	if st, raw := e.call(e.Admin, "POST", "/sections/"+e.SectionA+"/assessments", map[string]any{"Name": "x", "maxPoints": 1, "weight": 1}); st != 400 {
		t.Errorf("wrong-case key on POST: %d %s", st, raw)
	}
	st, raw := e.raw(e.Admin, "POST", "/sections/"+e.SectionA+"/assessments", ok())
	id := jmap(t, []byte(raw))["id"].(string)
	if st != 201 {
		t.Fatal(raw)
	}
	for name, c := range map[string]struct {
		body map[string]any
		want []string
	}{
		"empty":     {map[string]any{}, []string{""}},
		"null":      {map[string]any{"weight": nil}, []string{"weight"}},
		"nulls":     {map[string]any{"name": nil, "weight": nil}, []string{"name", "weight"}},
		"empty due": {map[string]any{"weight": 101, "dueOn": ""}, []string{"weight", "dueOn"}},
		"bad body":  {map[string]any{"maxPoints": 0}, []string{"maxPoints"}},
	} {
		st, raw := e.call(e.Admin, "PATCH", e.aPath(id), c.body)
		if _, f := errDetails(raw); st != 400 || !reflect.DeepEqual(f, c.want) {
			t.Errorf("patch %s: %d %s", name, st, raw)
		}
	}
	if st, raw := e.call(e.Admin, "PATCH", e.aPath(id), map[string]any{"Name": "x"}); st != 400 {
		t.Errorf("wrong-case key: %d %s", st, raw)
	}
}

func TestAssessmentNameConflict(t *testing.T) {
	e := newEnv(t)
	e.assessment("Quiz", "10", "10")
	other := e.assessment("Other", "10", "10")
	if st, raw := e.raw(e.Admin, "POST", "/sections/"+e.SectionA+"/assessments", map[string]any{"name": "Quiz", "maxPoints": 1, "weight": 1}); st != 409 {
		t.Errorf("post: %d %s", st, raw)
	}
	if st, raw := e.raw(e.Admin, "PATCH", e.aPath(other), map[string]any{"name": "Quiz"}); st != 409 {
		t.Errorf("patch: %d %s", st, raw)
	}
}

func TestWeightsDraftAnySum(t *testing.T) {
	e := newEnv(t)
	for i, w := range []int{40, 70, 0} {
		if st, raw := e.raw(e.Admin, "POST", "/sections/"+e.SectionA+"/assessments", map[string]any{"name": string(rune('A' + i)), "maxPoints": 10, "weight": w}); st != 201 {
			t.Fatalf("weight %d: %d %s", w, st, raw)
		}
	}
}

func TestAssessmentAuthz(t *testing.T) {
	e := newEnv(t)
	pub := e.assessment("Pub", "10", "10")
	e.assessment("Draft", "10", "10")
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`update assessments set published = true where id = $1`, pub)
		return err
	})
	list := "/sections/" + e.SectionA + "/assessments"
	for _, role := range roles {
		read, write := 200, 403
		switch role {
		case "daout", "other":
			read = 403
		case "admin", "dain", "inst":
			write = 201
		}
		st, raw := e.raw(e.as(role), "GET", list, nil)
		if st != read {
			t.Errorf("GET %s: %d %s", role, st, raw)
		}
		if role == "stud" && (!strings.Contains(raw, "Pub") || strings.Contains(raw, "Draft")) {
			t.Errorf("student list: %s", raw)
		}
		st, raw = e.raw(e.as(role), "POST", list, map[string]any{"name": "N-" + role, "maxPoints": 1, "weight": 1})
		if st != write {
			t.Errorf("POST %s: %d %s", role, st, raw)
		}
		wp := map[int]int{201: 200, 403: 403}[write]
		if st, raw = e.raw(e.as(role), "PATCH", e.aPath(pub), map[string]any{"position": 3}); st != wp {
			t.Errorf("PATCH %s: %d %s", role, st, raw)
		}
	}
	for _, role := range roles { // delete last: it removes the assessment
		want := 403
		if role == "admin" || role == "dain" || role == "inst" {
			want = 204
		}
		id := pub
		if want == 204 {
			id = e.assessment("D-"+role, "1", "1")
		}
		if st, raw := e.raw(e.as(role), "DELETE", e.aPath(id), nil); st != want {
			t.Errorf("DELETE %s: %d %s", role, st, raw)
		}
	}
	// 404s: unknown section, another section's assessment, B's section under slug a
	var otherSec string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`insert into sections (university_id, course_id, term_id, number) values ($1,$2,$3,'002') returning id`, e.Fx.UniversityA, e.CourseA, e.Fx.TermA).Scan(&otherSec)
	})
	for _, p := range []string{"/sections/nope/assessments", "/sections/00000000-0000-0000-0000-000000000000/assessments"} {
		if st, raw := e.raw(e.Admin, "GET", p, nil); st != 404 {
			t.Errorf("%s: %d %s", p, st, raw)
		}
	}
	if st, _ := e.raw(e.Admin, "PATCH", e.aPath("nope"), map[string]any{"position": 1}); st != 404 {
		t.Errorf("bad assessment id: %d", st)
	}
	if st, _ := e.raw(e.Admin, "PATCH", "/sections/"+otherSec+"/assessments/"+pub, map[string]any{"position": 1}); st != 404 {
		t.Errorf("another section's assessment: %d", st)
	}
	var bSec string
	e.tx(e.Fx.UniversityB, func(tx *sql.Tx) error {
		if err := tx.QueryRow(`insert into terms (university_id, name, starts_on, ends_on) values ($1,'T','2026-01-01','2026-02-01') returning id`, e.Fx.UniversityB).Scan(&bSec); err != nil {
			return err
		}
		return tx.QueryRow(`insert into sections (university_id, course_id, term_id, number) values ($1,$2,$3,'1') returning id`, e.Fx.UniversityB, e.CourseB, bSec).Scan(&bSec)
	})
	if st, _ := e.raw(e.Admin, "GET", "/sections/"+bSec+"/assessments", nil); st != 404 {
		t.Errorf("B's section: %d", st)
	}
}

func TestAssessmentAudit(t *testing.T) {
	e := newEnv(t)
	_, raw := e.raw(e.Admin, "POST", "/sections/"+e.SectionA+"/assessments", map[string]any{"name": "Mid", "maxPoints": 100, "weight": 40})
	id := jmap(t, []byte(raw))["id"].(string)
	e.raw(e.Admin, "PATCH", e.aPath(id), map[string]any{"weight": 35})
	e.raw(e.Admin, "DELETE", e.aPath(id), nil)
	if n, b, a := e.audit("assessment.create", id); n != 1 || b.Valid || !strings.Contains(a.String, `"name": "Mid"`) || !strings.Contains(a.String, "40.00") {
		t.Errorf("create %d %v %v", n, b, a)
	}
	if n, b, a := e.audit("assessment.update", id); n != 1 || !strings.Contains(b.String, "40.00") || !strings.Contains(a.String, "35.00") {
		t.Errorf("update %d %v %v", n, b, a)
	}
	if n, b, a := e.audit("assessment.delete", id); n != 1 || !strings.Contains(b.String, "35.00") || a.Valid {
		t.Errorf("delete %d %v %v", n, b, a)
	}
}

func (e *env) post(c *http.Cookie, b map[string]any) (int, string) {
	return e.raw(c, "POST", "/sections/"+e.SectionA+"/assessments", b)
}

func win(extra map[string]any) map[string]any {
	b := map[string]any{"kind": "quiz", "name": "Q", "weight": 5, "opensAt": "2026-10-12T09:00:00Z", "closesAt": "2026-10-13T03:30:00Z"}
	for k, v := range extra {
		b[k] = v
	}
	return b
}

func TestAssessmentKindDefaultsManual(t *testing.T) {
	e := newEnv(t)
	st, raw := e.post(e.Inst, map[string]any{"name": "M", "maxPoints": 10, "weight": 1})
	if st != 201 || !strings.Contains(raw, `"kind":"manual"`) || !strings.Contains(raw, `"description":""`) || strings.Contains(raw, "opensAt") || strings.Contains(raw, "questionCount") {
		t.Fatalf("%d %s", st, raw)
	}
}

func TestAssessmentWindowRules(t *testing.T) {
	e := newEnv(t)
	for name, c := range map[string]struct {
		body map[string]any
		want []string
	}{
		"manual with opensAt":     {map[string]any{"name": "M", "maxPoints": 1, "weight": 1, "opensAt": "2026-10-12T09:00:00Z"}, []string{"opensAt"}},
		"manual with closesAt":    {map[string]any{"name": "M", "maxPoints": 1, "weight": 1, "closesAt": "2026-10-12T09:00:00Z"}, []string{"closesAt"}},
		"quiz without window":     {map[string]any{"kind": "quiz", "name": "Q", "weight": 1}, []string{"opensAt", "closesAt"}},
		"assignment no closesAt":  {map[string]any{"kind": "assignment", "name": "A", "maxPoints": 10, "weight": 1, "opensAt": "2026-10-12T09:00:00Z"}, []string{"closesAt"}},
		"bad time":                {win(map[string]any{"opensAt": "tomorrow"}), []string{"opensAt"}},
		"closes before opens":     {win(map[string]any{"closesAt": "2026-10-12T09:00:00Z"}), []string{"closesAt"}},
		"closes before opens (2)": {win(map[string]any{"closesAt": "2026-10-01T09:00:00Z"}), []string{"closesAt"}},
		"quiz maxPoints":          {win(map[string]any{"maxPoints": 10}), []string{"maxPoints"}},
		"quiz dueOn":              {win(map[string]any{"dueOn": "2026-10-12"}), []string{"dueOn"}},
		"assignment dueOn":        {win(map[string]any{"kind": "assignment", "maxPoints": 10, "dueOn": "2026-10-12"}), []string{"dueOn"}},
		"bad kind":                {win(map[string]any{"kind": "exam"}), []string{"kind"}},
		"description long":        {win(map[string]any{"description": strings.Repeat("é", 10001)}), []string{"description"}},
	} {
		st, raw := e.post(e.Inst, c.body)
		if _, f := errDetails([]byte(raw)); st != 400 || !reflect.DeepEqual(f, c.want) {
			t.Errorf("%s: %d %s", name, st, raw)
		}
	}
	if st, raw := e.post(e.Inst, win(map[string]any{"description": strings.Repeat("é", 10000)})); st != 201 {
		t.Errorf("description at limit: %d %s", st, raw)
	}
}

func TestAssessmentQuizDueOnInUniversityTimezone(t *testing.T) {
	e := newEnv(t) // the fixture university is America/New_York
	// 2026-10-12 23:30 in New York (EDT, UTC-4) is 03:30 UTC on the 13th
	st, raw := e.post(e.Inst, win(map[string]any{"opensAt": "2026-10-12T09:00:00-04:00", "closesAt": "2026-10-12T23:30:00-04:00", "description": "  Read ch. 1\n\nthen answer  "}))
	m := jmap(t, []byte(raw))
	if st != 201 || m["dueOn"] != "2026-10-12" || m["closesAt"] != "2026-10-13T03:30:00Z" || m["opensAt"] != "2026-10-12T13:00:00Z" ||
		m["maxPoints"] != 1.0 || m["kind"] != "quiz" || m["questionCount"] != 0.0 || m["description"] != "Read ch. 1\n\nthen answer" {
		t.Fatalf("%d %s", st, raw)
	}
	id := m["id"].(string)
	// PATCH recomputes dueOn from the new closesAt
	st, raw = e.raw(e.Inst, "PATCH", e.aPath(id), map[string]any{"closesAt": "2026-10-20T02:00:00Z"})
	if m = jmap(t, []byte(raw)); st != 200 || m["dueOn"] != "2026-10-19" {
		t.Fatalf("patch: %d %s", st, raw)
	}
	if st, raw = e.raw(e.Inst, "PATCH", e.aPath(id), map[string]any{"closesAt": "2026-10-01T00:00:00Z"}); st != 400 || !strings.Contains(raw, "after opensAt") {
		t.Fatalf("patch window: %d %s", st, raw)
	}
	for _, b := range []map[string]any{{"maxPoints": 5}, {"dueOn": "2026-10-20"}, {"dueOn": nil}} {
		if st, raw = e.raw(e.Inst, "PATCH", e.aPath(id), b); st != 400 {
			t.Errorf("patch %v: %d %s", b, st, raw)
		}
	}
	// assignment keeps its own maxPoints
	st, raw = e.post(e.Inst, win(map[string]any{"name": "A", "kind": "assignment", "maxPoints": 20}))
	if m = jmap(t, []byte(raw)); st != 201 || m["maxPoints"] != 20.0 || m["dueOn"] != "2026-10-12" || strings.Contains(raw, "questionCount") {
		t.Fatalf("assignment: %d %s", st, raw)
	}
}

func TestAssessmentKindImmutable(t *testing.T) {
	e := newEnv(t)
	_, raw := e.post(e.Inst, win(nil))
	id := jmap(t, []byte(raw))["id"].(string)
	st, raw := e.raw(e.Inst, "PATCH", e.aPath(id), map[string]any{"kind": "manual"})
	if f := func() []string { _, f := errDetails([]byte(raw)); return f }(); st != 400 || !reflect.DeepEqual(f, []string{"kind"}) {
		t.Fatalf("%d %s", st, raw)
	}
}

func TestAssessmentKindRoles(t *testing.T) {
	e := newEnv(t)
	for _, role := range roles {
		quiz, manual := 403, 403
		switch role {
		case "admin", "dain", "inst":
			quiz, manual = 201, 201
		case "ta":
			quiz = 201
		}
		if st, raw := e.post(e.as(role), win(map[string]any{"name": "Q-" + role})); st != quiz {
			t.Errorf("quiz %s: %d %s", role, st, raw)
		}
		if st, raw := e.post(e.as(role), win(map[string]any{"kind": "assignment", "maxPoints": 5, "name": "A-" + role})); st != quiz {
			t.Errorf("assignment %s: %d %s", role, st, raw)
		}
		if st, raw := e.post(e.as(role), map[string]any{"name": "M-" + role, "maxPoints": 1, "weight": 1}); st != manual {
			t.Errorf("manual %s: %d %s", role, st, raw)
		}
	}
	// TA edits and deletes a quiz but not a manual assessment; an invalid kind is 400 for a TA, not 403
	_, raw := e.post(e.Inst, win(map[string]any{"name": "TQ"}))
	qid := jmap(t, []byte(raw))["id"].(string)
	mid := e.assessment("Man", "10", "10")
	if st, raw := e.raw(e.TA, "PATCH", e.aPath(qid), map[string]any{"position": 4}); st != 200 {
		t.Errorf("ta patch quiz: %d %s", st, raw)
	}
	if st, raw := e.raw(e.TA, "PATCH", e.aPath(mid), map[string]any{"position": 4}); st != 403 {
		t.Errorf("ta patch manual: %d %s", st, raw)
	}
	if st, raw := e.raw(e.TA, "DELETE", e.aPath(mid), nil); st != 403 {
		t.Errorf("ta delete manual: %d %s", st, raw)
	}
	if st, raw := e.post(e.TA, win(map[string]any{"kind": "exam"})); st != 400 {
		t.Errorf("ta bad kind: %d %s", st, raw)
	}
	if st, raw := e.raw(e.Stud, "PATCH", e.aPath(qid), map[string]any{"position": 4}); st != 403 {
		t.Errorf("stud patch quiz: %d %s", st, raw)
	}
	if st, raw := e.raw(e.TA, "DELETE", e.aPath(qid), nil); st != 204 {
		t.Errorf("ta delete quiz: %d %s", st, raw)
	}
}

func TestAssessmentStudentListKinds(t *testing.T) {
	e := newEnv(t)
	e.post(e.Inst, win(map[string]any{"name": "DraftQuiz"}))
	e.post(e.Inst, win(map[string]any{"name": "DraftAssign", "kind": "assignment", "maxPoints": 5}))
	e.assessment("DraftManual", "10", "10")
	pub := e.assessment("PubManual", "10", "10")
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`update assessments set published = true where id = $1`, pub)
		return err
	})
	_, raw := e.raw(e.Stud, "GET", "/sections/"+e.SectionA+"/assessments", nil)
	for _, want := range []string{"DraftQuiz", "DraftAssign", "PubManual"} {
		if !strings.Contains(raw, want) {
			t.Errorf("student misses %s: %s", want, raw)
		}
	}
	if strings.Contains(raw, "DraftManual") {
		t.Errorf("student sees unpublished manual: %s", raw)
	}
	_, raw = e.raw(e.TA, "GET", "/sections/"+e.SectionA+"/assessments", nil)
	if !strings.Contains(raw, "DraftManual") {
		t.Errorf("staff misses manual: %s", raw)
	}
}
