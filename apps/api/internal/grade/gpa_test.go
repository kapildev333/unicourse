package grade

import (
	"database/sql"
	"strings"
	"testing"
)

// section creates a course (credits) and a section in term, enrolls Stud and, when letter != "", gives Stud a final grade.
func (e *env) section(code, term, credits, status, letter, points string) (secID string) {
	e.T.Helper()
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		var course string
		if err := tx.QueryRow(`insert into courses (university_id, department_id, code, title, credits) values ($1,$2,$3,$3,$4::numeric) returning id`, e.Fx.UniversityA, e.Fx.DeptA, code, credits).Scan(&course); err != nil {
			return err
		}
		if err := tx.QueryRow(`insert into sections (university_id, course_id, term_id, number) values ($1,$2,$3,'001') returning id`, e.Fx.UniversityA, course, term).Scan(&secID); err != nil {
			return err
		}
		if _, err := tx.Exec(`insert into enrollments (university_id, section_id, user_id) values ($1,$2,$3)`, e.Fx.UniversityA, secID, e.UserID["stud@a.edu"]); err != nil {
			return err
		}
		if status == "" {
			return nil
		}
		var l, p *string
		if letter != "" {
			l, p = &letter, &points
		}
		_, err := tx.Exec(`insert into final_grades (university_id, section_id, user_id, status, percent, letter, points, credits, finalized_by)
			values ($1,$2,$3,$4,case when $5::text is null then null else 90 end,$5,$6::numeric,$7::numeric,$8)`, e.Fx.UniversityA, secID, e.UserID["stud@a.edu"], status, l, p, credits, e.Fx.AdminA)
		return err
	})
	return
}

func TestMyGradesPublishedOnly(t *testing.T) {
	e := newEnv(t)
	hidden := e.assessment("Hidden quiz", "10", "10")
	e.score(hidden, "stud@a.edu", "5")
	get := func() string {
		st, body := e.raw(e.Stud, "GET", "/me/grades", nil)
		if st != 200 {
			t.Fatal(st, body)
		}
		return body
	}
	if body := get(); strings.Contains(body, "Hidden quiz") || strings.Contains(body, `"running"`) || !strings.Contains(body, `"unpublishedCount":1`) {
		t.Errorf("unpublished leaked: %s", body)
	}
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`update assessments set published = true where id = $1`, hidden)
		return err
	})
	body := get()
	if !strings.Contains(body, "Hidden quiz") || !strings.Contains(body, `"points":5.00`) || !strings.Contains(body, `"running":{"percent":50.00,"letter":"F"}`) || !strings.Contains(body, `"unpublishedCount":0`) {
		t.Errorf("published: %s", body)
	}
	// an unpublished assessment without a score does not block running
	e.assessment("Draft", "10", "10")
	if body := get(); !strings.Contains(body, `"running"`) || strings.Contains(body, "Draft") || !strings.Contains(body, `"unpublishedCount":1`) {
		t.Errorf("draft: %s", body)
	}
	// a scored unpublished one does
	draft := e.assessment("Draft 2", "10", "10")
	e.score(draft, "stud@a.edu", "10")
	if body := get(); strings.Contains(body, `"running"`) || strings.Contains(body, "Draft 2") || !strings.Contains(body, `"unpublishedCount":2`) {
		t.Errorf("scored draft: %s", body)
	}
}

func TestMyGradesGPA(t *testing.T) {
	e := newEnv(t)
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`delete from sections where id = $1`, e.SectionA) // Stud's only section is the fixture's
		return err
	})
	e.section("G1", e.Fx.TermA, "6.7", "graded", "A-", "3.70")
	g2 := e.section("G2", e.Fx.TermA, "7.3", "graded", "B", "3.00")
	e.section("G3", e.Fx.TermA, "3.0", "incomplete", "", "")
	e.section("G4", e.TermA2, "3.0", "graded", "B+", "3.30")
	st, body := e.raw(e.Stud, "GET", "/me/grades", nil)
	if st != 200 {
		t.Fatal(st, body)
	}
	// 3.70 x 6.7 + 3.00 x 7.3 = 46.69 / 14.0 = 3.335 -> 3.34; cumulative (46.69 + 9.90) / 17.0 = 3.329 -> 3.33
	for _, want := range []string{`"gpa":3.34`, `"credits":14.0`, `"gpa":3.30`, `"cumulativeGpa":3.33`, `"cumulativeCredits":17.0`} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %s in %s", want, body)
		}
	}
	if strings.Index(body, "Spring 2027") > strings.Index(body, "Fall 2026") {
		t.Error("terms must be newest first")
	}
	// the snapshot credits weigh the GPA: editing the course afterwards moves nothing...
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`update courses set credits = 3.0 where code = 'G2'`)
		return err
	})
	if _, again := e.raw(e.Stud, "GET", "/me/grades", nil); !strings.Contains(again, `"gpa":3.34`) {
		t.Errorf("course edit moved the GPA: %s", again)
	}
	// ...the final grade's own credits do.
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`update final_grades set credits = 3.0 where section_id = $1`, g2)
		return err
	})
	if _, again := e.raw(e.Stud, "GET", "/me/grades", nil); !strings.Contains(again, `"gpa":3.48`) { // (24.79 + 9.00) / 9.7
		t.Errorf("%s", again)
	}
}

func TestMyGradesFrozenCredits(t *testing.T) {
	e := newEnv(t)
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`delete from sections where id = $1`, e.SectionA)
		return err
	})
	e.section("F1", e.Fx.TermA, "4.0", "graded", "A", "4.00")
	e.section("F2", e.Fx.TermA, "2.0", "graded", "B", "3.00")
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`update courses set credits = 9.0 where code = 'F1'`)
		return err
	})
	_, body := e.raw(e.Stud, "GET", "/me/grades", nil)
	// the sections show 4.0 and 2.0 (frozen), the course itself 9.0 (catalog), and the term total is their sum
	if !strings.Contains(body, `"credits":4.0,"final"`) || !strings.Contains(body, `"credits":2.0,"final"`) || !strings.Contains(body, `"credits":6.0,"sections"`) {
		t.Errorf("%s", body)
	}
}

func TestMyGradesShape(t *testing.T) {
	e := newEnv(t)
	if st, body := e.raw(e.Other, "GET", "/me/grades", nil); st != 200 || strings.TrimSpace(body) != `{"terms":[],"cumulativeCredits":0}` {
		t.Errorf("empty: %d %s", st, body)
	}
	if st, body := e.raw(e.Inst, "GET", "/me/grades", nil); st != 200 || !strings.Contains(body, `"terms":[]`) {
		t.Errorf("staff only: %d %s", st, body)
	}
	e.finalized()
	st, body := e.raw(e.Stud, "GET", "/me/grades", nil)
	if st != 200 || !strings.Contains(body, `"final":{"status":"graded"`) || strings.Contains(body, `"running"`) {
		t.Errorf("finalized: %d %s", st, body)
	}
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`delete from enrollments where user_id = $1`, e.UserID["stud@a.edu"])
		return err
	})
	if _, body := e.raw(e.Stud, "GET", "/me/grades", nil); !strings.Contains(body, `"cumulativeGpa"`) || !strings.Contains(body, e.SectionA) {
		t.Errorf("after unenroll: %s", body)
	}
	// university B's data is invisible: B's section id never appears
	if _, body := e.raw(e.Admin, "GET", "/me/grades", nil); strings.Contains(body, e.CourseB) {
		t.Error("cross-tenant leak")
	}
}
