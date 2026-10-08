package assignment

import (
	"database/sql"
	"strings"
	"testing"
	"time"
)

func TestSubmissionPut(t *testing.T) {
	e := newEnv(t)
	a := e.open()
	st, raw := e.put(e.Stud, a, "  first draft  ")
	m := jmap(t, raw)
	if st != 200 || m["body"] != "first draft" || m["userId"] != e.UserID["stud@a.edu"] || len(m["files"].([]any)) != 0 {
		t.Fatalf("create: %d %s", st, raw)
	}
	if _, ok := m["score"]; ok {
		t.Fatalf("score leaked: %s", raw)
	}
	t1, _ := time.Parse(time.RFC3339Nano, m["submittedAt"].(string))
	st, raw = e.put(e.Stud, a, "second")
	m = jmap(t, raw)
	t2, _ := time.Parse(time.RFC3339Nano, m["submittedAt"].(string))
	if st != 200 || m["body"] != "second" || !t2.After(t1) || e.count(`select count(*) from submissions`) != 1 {
		t.Fatalf("replace: %d %s", st, raw)
	}
	// GET returns the same
	if st, raw = e.call(e.Stud, "GET", e.subPath(a, "/me"), nil); st != 200 || jmap(t, raw)["body"] != "second" {
		t.Fatalf("get: %d %s", st, raw)
	}
}

func TestSubmissionWindow(t *testing.T) {
	e := newEnv(t)
	up, closed := e.assignment("1 hour", "2 hours"), e.assignment("-2 hours", "-1 hour")
	for id, msg := range map[string]string{up: "not open yet", closed: "closed"} {
		st, raw := e.put(e.Stud, id, "x")
		if code, _ := errDetails(raw); st != 409 || code != "conflict" || !strings.Contains(string(raw), msg) {
			t.Errorf("%s: %d %s", msg, st, raw)
		}
	}
	if e.count(`select count(*) from submissions`) != 0 {
		t.Error("a row was written outside the window")
	}
	// the window closing moves a stored submission out of reach
	a := e.open()
	e.put(e.Stud, a, "kept")
	e.setWindow(a, "-2 hours", "-1 hour")
	if st, _ := e.put(e.Stud, a, "late"); st != 409 {
		t.Errorf("after close: %d", st)
	}
	if st, raw := e.call(e.Stud, "GET", e.subPath(a, "/me"), nil); st != 200 || jmap(t, raw)["body"] != "kept" {
		t.Errorf("read after close: %d %s", st, raw)
	}
}

func TestSubmissionNotAnAssignment(t *testing.T) {
	e := newEnv(t)
	manual := e.assessmentManual()
	for _, id := range []string{manual, "x", "00000000-0000-0000-0000-000000000000"} {
		if st, _ := e.put(e.Stud, id, "x"); st != 404 {
			t.Errorf("%s: %d", id, st)
		}
	}
}

func TestSubmissionBodyLimit(t *testing.T) {
	e := newEnv(t)
	a := e.open()
	if st, raw := e.put(e.Stud, a, strings.Repeat("é", 20000)); st != 200 {
		t.Fatalf("20000: %d %s", st, raw)
	}
	st, raw := e.put(e.Stud, a, strings.Repeat("é", 20001))
	if code, f := errDetails(raw); st != 400 || code != "validation_failed" || len(f) != 1 || f[0] != "body" {
		t.Fatalf("20001: %d %s", st, raw)
	}
	if st, raw = e.put(e.Stud, a, "a\x00b"); st != 400 {
		t.Fatalf("NUL: %d %s", st, raw)
	}
	for _, body := range []any{`{}`, `{"body":null}`, `{"body":"x","extra":1}`} {
		if st, raw = e.rawJSON(e.Stud, "PUT", e.subPath(a, "/me"), body.(string)); st != 400 {
			t.Errorf("%s: %d %s", body, st, raw)
		}
	}
}

func TestSubmissionEmptyRejected(t *testing.T) {
	e := newEnv(t)
	a := e.open()
	for _, body := range []string{"", "  \n "} {
		st, raw := e.put(e.Stud, a, body)
		if code, f := errDetails(raw); st != 400 || code != "validation_failed" || len(f) != 1 || f[0] != "body" {
			t.Fatalf("%q: %d %s", body, st, raw)
		}
	}
	if e.count(`select count(*) from submissions`) != 0 {
		t.Fatal("empty submission stored")
	}
	// an empty body is fine when a file is held
	if st, raw := e.upload(e.Stud, a, "a.txt", strings.NewReader("x")); st != 201 {
		t.Fatalf("upload: %d %s", st, raw)
	}
	if st, raw := e.put(e.Stud, a, ""); st != 200 {
		t.Fatalf("empty body with a file: %d %s", st, raw)
	}
}

func TestSubmissionLastFileRemovedDeletesEmpty(t *testing.T) {
	e := newEnv(t)
	a := e.open()
	_, raw := e.upload(e.Stud, a, "a.txt", strings.NewReader("x"))
	f := fileIDs(t, raw)[0]
	// grade it so the scores row exists
	e.call(e.Inst, "PATCH", e.subPath(a, "/"+e.UserID["stud@a.edu"]), map[string]any{"score": 5})
	if st, _ := e.call(e.Stud, "DELETE", e.subPath(a, "/me/files/"+f), nil); st != 204 {
		t.Fatalf("delete: %d", st)
	}
	if e.count(`select count(*) from submissions`) != 0 || e.count(`select count(*) from scores`) != 1 {
		t.Fatal("empty submission kept or score removed")
	}
	if st, _ := e.call(e.Stud, "GET", e.subPath(a, "/me"), nil); st != 404 {
		t.Fatalf("get: %d", st)
	}
	// with text the submission stays
	_, raw = e.upload(e.Stud, a, "a.txt", strings.NewReader("x"))
	e.put(e.Stud, a, "text")
	if st, _ := e.call(e.Stud, "DELETE", e.subPath(a, "/me/files/"+fileIDs(t, raw)[0]), nil); st != 204 || e.count(`select count(*) from submissions`) != 1 {
		t.Fatal("submission with text was deleted")
	}
}

func TestSubmissionsList(t *testing.T) {
	e := newEnv(t)
	a := e.open()
	e.put(e.Stud2, a, "b")
	e.put(e.Stud, a, "a")
	e.upload(e.Stud, a, "a.txt", strings.NewReader("x"))
	e.upload(e.Stud, a, "b.txt", strings.NewReader("x"))
	// a submission of a student who is no longer enrolled is not listed
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`delete from enrollments where user_id = $1`, e.UserID["stud2@a.edu"])
		return err
	})
	e.put(e.Stud, a, "a2")
	st, raw := e.call(e.Inst, "GET", e.subPath(a, ""), nil)
	items := jmap(t, raw)["items"].([]any)
	if st != 200 || len(items) != 1 {
		t.Fatalf("%d %s", st, raw)
	}
	row := items[0].(map[string]any)
	if row["email"] != "stud@a.edu" || row["displayName"] != "N stud@a.edu" || row["fileCount"] != float64(2) {
		t.Fatalf("%s", raw)
	}
	// order by name
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`insert into enrollments (university_id, section_id, user_id) values ($1, $2, $3)`, e.Fx.UniversityA, e.SectionA, e.UserID["stud2@a.edu"])
		return err
	})
	_, raw = e.call(e.Inst, "GET", e.subPath(a, ""), nil)
	items = jmap(t, raw)["items"].([]any)
	// "N stud2@a.edu" sorts before "N stud@a.edu"
	if len(items) != 2 || items[0].(map[string]any)["email"] != "stud2@a.edu" {
		t.Fatalf("not ordered by name: %s", raw)
	}
}

func TestSubmissionReadHidesGrade(t *testing.T) {
	e := newEnv(t)
	a := e.open()
	e.put(e.Stud, a, "work")
	path := e.subPath(a, "/"+e.UserID["stud@a.edu"])
	if st, raw := e.call(e.TA, "PATCH", path, map[string]any{"score": 150, "feedback": "good"}); st != 200 {
		t.Fatalf("grade: %d %s", st, raw)
	}
	// staff always see the grade
	if _, raw := e.call(e.TA, "GET", path, nil); jmap(t, raw)["score"] != 150.0 || jmap(t, raw)["feedback"] != "good" {
		t.Fatalf("staff: %s", raw)
	}
	for _, p := range []string{e.subPath(a, "/me"), path} {
		_, raw := e.call(e.Stud, "GET", p, nil)
		m := jmap(t, raw)
		if _, ok := m["score"]; ok || m["feedback"] != nil || m["gradedAt"] != nil || m["body"] != "work" {
			t.Fatalf("unpublished: %s", raw)
		}
	}
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`update assessments set published = true where id = $1`, a)
		return err
	})
	_, raw := e.call(e.Stud, "GET", e.subPath(a, "/me"), nil)
	m := jmap(t, raw)
	if m["score"] == nil || m["feedback"] != "good" || m["gradedAt"] == nil {
		t.Fatalf("published: %s", raw)
	}
	// the PUT response follows the same projection
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`update assessments set published = false where id = $1`, a)
		return err
	})
	if _, raw = e.put(e.Stud, a, "again"); jmap(t, raw)["score"] != nil {
		t.Fatalf("put leaked: %s", raw)
	}
}
