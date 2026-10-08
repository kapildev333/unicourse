package quiz

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/kapildev333/unicourse/apps/api/internal/pg"
)

func (e *env) sweeper(batch int) *Sweeper {
	s := NewSweeper(e.DB, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if batch > 0 {
		s.Batch = batch
	}
	return s
}

// users inserts n plain users and returns their ids.
func (e *env) users(n int) (ids []string) {
	e.T.Helper()
	err := pg.WithPlatformTx(context.Background(), e.DB, func(tx *sql.Tx) error {
		rows, err := tx.Query(`insert into users (email, display_name) select 'sw' || g || '@a.edu', 'S' || g from generate_series(1, $1::int) g returning id`, n)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			rows.Scan(&id)
			ids = append(ids, id)
		}
		return rows.Err()
	})
	if err != nil {
		e.T.Fatal(err)
	}
	return ids
}

func (e *env) startFor(uni, quizID, userID string) {
	e.T.Helper()
	e.tx(uni, func(tx *sql.Tx) error {
		_, err := tx.Exec(`insert into attempts (university_id, assessment_id, user_id) values ($1, $2, $3)`, uni, quizID, userID)
		return err
	})
}

func TestSweepSubmitsClosed(t *testing.T) {
	e := newEnv(t)
	id, qs := e.openQuiz()
	e.call(e.Stud, "POST", e.apath(id), nil)
	e.call(e.Stud, "PUT", e.apath(id)+"/me", map[string]any{"answers": map[string]any{qs[0]: map[string]any{"choices": []int{0}}}})
	e.closeQuiz(id)
	n, err := e.sweeper(0).Sweep(context.Background())
	if n != 1 || err != nil {
		t.Fatalf("swept %d %v", n, err)
	}
	var score, gradedBy string
	var auto, atClose bool
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		if err := tx.QueryRow(`select a.score::text, a.auto_submitted, a.submitted_at = q.closes_at from attempts a join assessments q on q.id = a.assessment_id where a.assessment_id = $1`, id).Scan(&score, &auto, &atClose); err != nil {
			return err
		}
		return tx.QueryRow(`select graded_by::text from scores where assessment_id = $1 and user_id = $2 and points = 2`, id, e.UserID["stud@a.edu"]).Scan(&gradedBy)
	})
	if score != "2.00" || !auto || !atClose || gradedBy != e.UserID["stud@a.edu"] {
		t.Fatalf("score %s auto %v atClose %v by %s", score, auto, atClose, gradedBy)
	}
	var actor string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select actor_user_id::text from audit_log where action = 'attempt.submit' and entity_id = $1`, id).Scan(&actor)
	})
	if _, _, after := e.audit("attempt.submit", id); actor != e.UserID["stud@a.edu"] || after.String == "" || !strings.Contains(after.String, `"auto": true`) {
		t.Fatalf("audit actor %s after %s", actor, after.String)
	}
	// the student reads the swept attempt
	st, b := e.call(e.Stud, "GET", e.apath(id)+"/me", nil)
	if m := jmap(t, b); st != 200 || m["autoSubmitted"] != true || m["submittedAt"] == nil {
		t.Fatalf("read %d %s", st, b)
	}
	if n, err := e.sweeper(0).Sweep(context.Background()); n != 0 || err != nil {
		t.Fatalf("second pass %d %v", n, err)
	}
}

func TestSweepLeavesOpen(t *testing.T) {
	e := newEnv(t)
	open, _ := e.openQuiz()
	e.call(e.Stud, "POST", e.apath(open), nil)
	closed, _ := e.openQuiz()
	e.attempt(closed, "stud@a.edu", true)
	e.closeQuiz(closed)
	var before string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select submitted_at::text from attempts where assessment_id = $1`, closed).Scan(&before)
	})
	if n, err := e.sweeper(0).Sweep(context.Background()); n != 0 || err != nil {
		t.Fatalf("swept %d %v", n, err)
	}
	var openSubmitted bool
	var after string
	var scores int
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		if err := tx.QueryRow(`select submitted_at is not null from attempts where assessment_id = $1`, open).Scan(&openSubmitted); err != nil {
			return err
		}
		if err := tx.QueryRow(`select count(*) from scores`).Scan(&scores); err != nil {
			return err
		}
		return tx.QueryRow(`select submitted_at::text from attempts where assessment_id = $1`, closed).Scan(&after)
	})
	if openSubmitted || after != before || scores != 0 {
		t.Fatalf("open %v before %s after %s scores %d", openSubmitted, before, after, scores)
	}
}

func TestSweepNoAttemptNoScore(t *testing.T) {
	e := newEnv(t)
	id, _ := e.openQuiz()
	e.closeQuiz(id)
	if n, err := e.sweeper(0).Sweep(context.Background()); n != 0 || err != nil {
		t.Fatalf("swept %d %v", n, err)
	}
	var scores, attempts int
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		tx.QueryRow(`select count(*) from scores`).Scan(&scores)
		return tx.QueryRow(`select count(*) from attempts`).Scan(&attempts)
	})
	if scores != 0 || attempts != 0 {
		t.Fatalf("scores %d attempts %d", scores, attempts)
	}
}

func TestSweepTwoProcesses(t *testing.T) {
	e := newEnv(t)
	id, _ := e.openQuiz()
	for _, u := range e.users(30) {
		e.startFor(e.Fx.UniversityA, id, u)
	}
	e.closeQuiz(id)
	counts := make([]int, 2)
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i := range counts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counts[i], errs[i] = e.sweeper(4).Sweep(context.Background())
		}()
	}
	wg.Wait()
	if errs[0] != nil || errs[1] != nil || counts[0]+counts[1] != 30 {
		t.Fatalf("counts %v errs %v", counts, errs)
	}
	var submitted, audits, scores int
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		tx.QueryRow(`select count(*) from attempts a join assessments q on q.id = a.assessment_id where a.submitted_at = q.closes_at and a.auto_submitted`).Scan(&submitted)
		tx.QueryRow(`select count(*) from audit_log where action = 'attempt.submit'`).Scan(&audits)
		return tx.QueryRow(`select count(*) from scores`).Scan(&scores)
	})
	if submitted != 30 || audits != 30 || scores != 30 {
		t.Fatalf("submitted %d audits %d scores %d", submitted, audits, scores)
	}
}

func TestSweepEachUniversity(t *testing.T) {
	e := newEnv(t)
	idA, _ := e.openQuiz()
	e.attempt(idA, "stud@a.edu", false)
	e.closeQuiz(idA)
	us := e.users(1)
	var secB, quizB string
	e.tx(e.Fx.UniversityB, func(tx *sql.Tx) error {
		if err := tx.QueryRow(`insert into sections (university_id, course_id, term_id, number) values ($1,$2,$3,'001') returning id`, e.Fx.UniversityB, e.CourseB, e.TermB).Scan(&secB); err != nil {
			return err
		}
		if err := tx.QueryRow(`insert into assessments (university_id, section_id, kind, name, max_points, weight, position, opens_at, closes_at)
			values ($1, $2, 'quiz', 'QB', 1, 10, 0, now() - interval '2 hours', now() - interval '1 hour') returning id`, e.Fx.UniversityB, secB).Scan(&quizB); err != nil {
			return err
		}
		_, err := tx.Exec(`insert into questions (university_id, assessment_id, kind, prompt, points, position, answers) values ($1, $2, 'blank', 'P', 1, 1, '["x"]')`, e.Fx.UniversityB, quizB)
		return err
	})
	e.startFor(e.Fx.UniversityB, quizB, us[0])
	if n, err := e.sweeper(0).Sweep(context.Background()); n != 2 || err != nil {
		t.Fatalf("swept %d %v", n, err)
	}
	var b int
	e.tx(e.Fx.UniversityB, func(tx *sql.Tx) error {
		return tx.QueryRow(`select count(*) from scores where assessment_id = $1`, quizB).Scan(&b)
	})
	if b != 1 {
		t.Fatalf("B scores %d", b)
	}
}
