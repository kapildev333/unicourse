package quiz

import (
	"database/sql"
	"strings"
	"testing"
	"time"
)

// limit sets the time limit (minutes) of a quiz by SQL.
func (e *env) limit(id string, minutes int) {
	e.T.Helper()
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`update assessments set time_limit_minutes = $2 where id = $1`, id, minutes)
		return err
	})
}

// startedAgo inserts an unsubmitted attempt that started the given interval ago.
func (e *env) startedAgo(id, email, ago string) {
	e.T.Helper()
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`insert into attempts (university_id, assessment_id, user_id, started_at) values ($1, $2, $3, now() - $4::interval)`,
			e.Fx.UniversityA, id, e.UserID[email], ago)
		return err
	})
}

func TestDeadlineSQL(t *testing.T) {
	e := newEnv(t)
	open := e.quiz("-1 hour", "1 hour")
	e.startedAgo(open, "stud@a.edu", "10 minutes")
	deadline := func() (d, closes, started time.Time) {
		e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
			return tx.QueryRow(`select `+deadlineSQL+`, q.closes_at, a.started_at from attempts a join assessments q on q.id = a.assessment_id where q.id = $1`, open).Scan(&d, &closes, &started)
		})
		return
	}
	if d, c, _ := deadline(); !d.Equal(c) {
		t.Errorf("no limit: %v, want closes_at %v", d, c)
	}
	e.limit(open, 30)
	if d, _, s := deadline(); !d.Equal(s.Add(30 * time.Minute)) {
		t.Errorf("limit 30: %v, want started_at+30m", d)
	}
	e.limit(open, 120)
	if d, c, _ := deadline(); !d.Equal(c) {
		t.Errorf("limit past close: %v, want closes_at %v", d, c)
	}
}

func TestQuestionsStudentDeadline(t *testing.T) {
	e := newEnv(t)
	timed := e.quiz("-1 hour", "1 hour")
	e.limit(timed, 10)
	e.putQuestions(timed, []qm{choiceQ("1", true, false), blankQ("2", "Paris")})
	if st, b := e.raw(e.Stud, "GET", e.qpath(timed), nil); st != 409 || !strings.Contains(b, "not started") {
		t.Errorf("timed, no attempt: %d %s", st, b)
	}
	e.startedAgo(timed, "stud@a.edu", "11 minutes")
	if st, b := e.raw(e.Stud, "GET", e.qpath(timed), nil); st != 409 || !strings.Contains(b, "closed") {
		t.Errorf("past the limit, window open: %d %s", st, b)
	}
	e.startedAgo(timed, "stud2@a.edu", "5 minutes")
	if st, b := e.raw(e.Stud2, "GET", e.qpath(timed), nil); st != 200 || strings.Contains(b, "correct") || strings.Contains(b, "Paris") {
		t.Errorf("within the limit: %d %s", st, b)
	}
	untimed := e.quiz("-1 hour", "1 hour")
	e.putQuestions(untimed, []qm{blankQ("2", "Paris")})
	if st, b := e.raw(e.Stud, "GET", e.qpath(untimed), nil); st != 200 {
		t.Errorf("untimed, no attempt: %d %s", st, b)
	}
}

func TestDeadlineSQLAlias(t *testing.T) {
	if DeadlineSQL != deadlineSQL {
		t.Fatal("DeadlineSQL must be deadlineSQL")
	}
}
