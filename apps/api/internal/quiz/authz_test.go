package quiz

import (
	"database/sql"
	"testing"
)

// TestQuizAuthz is the role matrix of design section 7 for every quiz route.
func TestQuizAuthz(t *testing.T) {
	e := newEnv(t)
	editors := []string{"admin", "dain", "inst", "ta"}
	outsiders := []string{"daout", "other", "badmin"}
	q, _ := e.submitted("stud") // stud has a submitted attempt
	fresh, _ := e.openQuiz()    // no attempts: questions PUT is allowed
	start, _ := e.openQuiz()    // stud has not started: POST and PUT me
	studID := e.UserID["stud@a.edu"]
	items := []any{map[string]any{"kind": "blank", "prompt": "Capital", "points": 1, "answers": []any{"Paris"}}}

	type call struct {
		name, method, path string
		body               any
	}
	check := func(c call, who string, want int) {
		t.Helper()
		if st, b := e.raw(e.as(who), c.method, c.path, c.body); st != want {
			t.Errorf("%s as %s: %d, want %d: %s", c.name, who, st, want, b)
		}
	}
	all := func(c call, who []string, want int) {
		t.Helper()
		for _, w := range who {
			check(c, w, want)
		}
	}

	qget := call{"questions GET", "GET", e.qpath(q), nil}
	all(qget, editors, 200)
	all(qget, []string{"stud", "stud2"}, 200)
	all(qget, outsiders, 403)

	qput := call{"questions PUT", "PUT", e.qpath(fresh), map[string]any{"items": items}}
	all(qput, []string{"stud", "stud2"}, 403)
	all(qput, outsiders, 403)
	all(qput, editors, 200)

	list := call{"attempts GET", "GET", e.apath(q), nil}
	all(list, editors, 200)
	all(list, []string{"stud", "stud2"}, 403)
	all(list, outsiders, 403)

	post := call{"attempts POST", "POST", e.apath(start), nil}
	all(post, editors, 403)
	all(post, outsiders, 403)
	check(post, "stud", 201)

	one := e.apath(q) + "/" + studID
	get := call{"attempt GET", "GET", one, nil}
	all(get, editors, 200)
	check(get, "stud", 200)
	check(call{"attempt GET me", "GET", e.apath(q) + "/me", nil}, "stud", 200)
	check(get, "stud2", 403)
	all(get, outsiders, 403)

	patch := call{"attempt PATCH", "PATCH", one, map[string]any{"feedback": "ok"}}
	all(patch, []string{"stud", "stud2"}, 403)
	all(patch, outsiders, 403)
	all(patch, editors, 200)

	put := call{"attempt PUT me", "PUT", e.apath(start) + "/me", map[string]any{"answers": map[string]any{}}}
	all(put, editors, 403)
	all(put, outsiders, 403)
	check(put, "stud2", 404) // enrolled, no attempt
	check(put, "stud", 200)

	// a quiz requested under another section's path is a 404, even for admin
	var sec2 string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		if err := tx.QueryRow(`insert into sections (university_id, course_id, term_id, number) values ($1,$2,$3,'002') returning id`, e.Fx.UniversityA, e.CourseA, e.Fx.TermA).Scan(&sec2); err != nil {
			return err
		}
		_, err := tx.Exec(`insert into enrollments (university_id, section_id, user_id) values ($1,$2,$3)`, e.Fx.UniversityA, sec2, e.UserID["other@a.edu"])
		return err
	})
	under := "/sections/" + sec2 + "/assessments/" + q
	for _, c := range []call{
		{"questions GET", "GET", under + "/questions", nil},
		{"questions PUT", "PUT", under + "/questions", map[string]any{"items": items}},
		{"attempts GET", "GET", under + "/attempts", nil},
		{"attempts POST", "POST", under + "/attempts", nil},
		{"attempt GET", "GET", under + "/attempts/" + studID, nil},
		{"attempt PATCH", "PATCH", under + "/attempts/" + studID, map[string]any{"feedback": "x"}},
		{"attempt PUT me", "PUT", under + "/attempts/me", map[string]any{"answers": map[string]any{}}},
	} {
		all(c, []string{"admin", "inst", "other"}, 404) // the quiz check comes before the role check
	}
	// a student enrolled in another section only: 403 on this section's quiz
	for _, c := range []call{qget, list, post, get, patch, put} {
		check(c, "other", 403)
	}
	// a section of another university or a bad id is a 404
	check(call{"unknown section", "GET", "/sections/00000000-0000-0000-0000-000000000000/assessments/" + q + "/attempts", nil}, "admin", 404)
	check(call{"bad section", "GET", "/sections/nope/assessments/" + q + "/questions", nil}, "admin", 404)
}
