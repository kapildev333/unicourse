package grade

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

func (e *env) sPath(a string) string { return e.aPath(a) + "/scores" }

type scoreRow struct {
	UserID, GradedBy string
	Points           json.Number
}

func scoreItems(t *testing.T, raw string) []scoreRow {
	t.Helper()
	var b struct{ Items []scoreRow }
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&b); err != nil {
		t.Fatal(raw, err)
	}
	return b.Items
}

func (e *env) put(c *http.Cookie, a string, rows ...map[string]any) (int, string) {
	return e.raw(c, "PUT", e.sPath(a), map[string]any{"scores": rows})
}

func sc(user any, points any) map[string]any { return map[string]any{"userId": user, "points": points} }

func (e *env) pointsOf(a, email string) (p string) {
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		err := tx.QueryRow(`select points::text from scores where assessment_id = $1 and user_id = $2`, a, e.UserID[email]).Scan(&p)
		if err == sql.ErrNoRows {
			p = ""
			return nil
		}
		return err
	})
	return
}

func TestScoresPut(t *testing.T) {
	e := newEnv(t)
	a := e.assessment("Mid", "100", "40")
	st, raw := e.put(e.Inst, a, sc(e.UserID["stud@a.edu"], 80), sc(e.UserID["stud2@a.edu"], 90))
	items := scoreItems(t, raw)
	if st != 200 || len(items) != 2 || items[0].UserID > items[1].UserID || !strings.Contains(raw, `"points":80.00`) || !strings.Contains(raw, `"points":90.00`) {
		t.Fatalf("%d %s", st, raw)
	}
	var at1 string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select graded_at::text from scores where user_id = $1`, e.UserID["stud2@a.edu"]).Scan(&at1)
	})
	st, raw = e.put(e.TA, a, sc(e.UserID["stud@a.edu"], 85), sc(e.UserID["stud2@a.edu"], 90))
	if st != 200 {
		t.Fatalf("%d %s", st, raw)
	}
	for _, r := range scoreItems(t, raw) {
		switch r.UserID {
		case e.UserID["stud@a.edu"]:
			if r.GradedBy != e.UserID["ta@a.edu"] || r.Points != "85.00" {
				t.Errorf("stud row %+v", r)
			}
		case e.UserID["stud2@a.edu"]:
			if r.GradedBy != e.UserID["inst@a.edu"] {
				t.Errorf("stud2 gradedBy changed: %+v", r)
			}
		}
	}
	var at2 string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select graded_at::text from scores where user_id = $1`, e.UserID["stud2@a.edu"]).Scan(&at2)
	})
	if at1 != at2 {
		t.Errorf("unchanged row's graded_at moved: %s -> %s", at1, at2)
	}
}

func TestScoresBulk500(t *testing.T) {
	e := newEnv(t)
	a := e.assessment("Big", "100", "10")
	if err := e.bulk(500); err != nil {
		t.Fatal(err)
	}
	var ids []string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		rows, err := tx.Query(`select user_id::text from enrollments where section_id = $1 order by user_id`, e.SectionA)
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
	if len(ids) < 502 {
		t.Fatalf("%d enrolled", len(ids))
	}
	mk := func(n int) []map[string]any {
		var r []map[string]any
		for _, id := range ids[:n] {
			r = append(r, sc(id, 50))
		}
		return r
	}
	count := func() (n int) {
		e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
			return tx.QueryRow(`select count(*) from scores where assessment_id = $1`, a).Scan(&n)
		})
		return
	}
	if st, raw := e.put(e.Admin, a, mk(501)...); st != 400 || !strings.Contains(raw, `"scores"`) {
		t.Fatalf("501: %d", st)
	}
	bad := mk(500)
	bad[317] = sc(e.UserID["other@a.edu"], 50)
	st, raw := e.put(e.Admin, a, bad...)
	if _, f := errDetails([]byte(raw)); st != 400 || !reflect.DeepEqual(f, []string{"scores[317].userId"}) || count() != 0 {
		t.Fatalf("one bad row: %d %s count %d", st, raw, count())
	}
	if st, raw := e.put(e.Admin, a, mk(500)...); st != 200 || count() != 500 {
		t.Fatalf("500: %d count %d %.200s", st, count(), raw)
	}
}

func TestScoresValidation(t *testing.T) {
	e := newEnv(t)
	a := e.assessment("Mid", "100", "40")
	u := e.UserID["stud@a.edu"]
	for name, c := range map[string]struct {
		rows []map[string]any
		want []string
	}{
		"negative":   {[]map[string]any{sc(u, -1)}, []string{"scores[0].points"}},
		"over bonus": {[]map[string]any{sc(u, 200.01)}, []string{"scores[0].points"}},
		"decimals":   {[]map[string]any{sc(u, 1.234)}, []string{"scores[0].points"}},
		"missing":    {[]map[string]any{{"userId": u}}, []string{"scores[0].points"}},
		"bad uuid":   {[]map[string]any{sc("x", 1)}, []string{"scores[0].userId"}},
		"dup":        {[]map[string]any{sc(u, 1), sc(u, 2)}, []string{"scores[1].userId"}},
		"empty":      {[]map[string]any{}, []string{"scores"}},
		"several":    {[]map[string]any{sc(u, -1), sc("x", 1), sc(e.UserID["other@a.edu"], 1)}, []string{"scores[0].points", "scores[1].userId", "scores[2].userId"}},
	} {
		st, raw := e.put(e.Admin, a, c.rows...)
		if _, f := errDetails([]byte(raw)); st != 400 || !reflect.DeepEqual(f, c.want) {
			t.Errorf("%s: %d %s", name, st, raw)
		}
	}
	if st, raw := e.put(e.Admin, a, sc(u, 200)); st != 200 || !strings.Contains(raw, `"points":200.00`) {
		t.Errorf("exactly 2 x max: %d %s", st, raw)
	}
	if _, raw := e.put(e.Admin, a, sc(u, -1)); !strings.Contains(raw, "must be between 0 and 200.00") {
		t.Errorf("message: %s", raw)
	}
}

func TestScoresDelete(t *testing.T) {
	e := newEnv(t)
	a := e.assessment("Mid", "100", "40")
	e.score(a, "stud@a.edu", "80")
	su, s2 := e.UserID["stud@a.edu"], e.UserID["stud2@a.edu"]
	st, raw := e.put(e.Admin, a, sc(su, nil), sc(s2, 70))
	items := scoreItems(t, raw)
	if st != 200 || len(items) != 1 || items[0].UserID != s2 || e.pointsOf(a, "stud@a.edu") != "" || e.pointsOf(a, "stud2@a.edu") != "70.00" {
		t.Fatalf("%d %s", st, raw)
	}
	if st, raw := e.put(e.Admin, a, sc(su, nil)); st != 200 || len(scoreItems(t, raw)) != 1 {
		t.Fatalf("null without score: %d %s", st, raw)
	}
	st, raw = e.put(e.Admin, a, map[string]any{"userId": su})
	if _, f := errDetails([]byte(raw)); st != 400 || !reflect.DeepEqual(f, []string{"scores[0].points"}) || !strings.Contains(raw, "is required") {
		t.Fatalf("absent points: %d %s", st, raw)
	}
	var after string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select after::text from audit_log where action = 'score.set' and entity_id = $1 order by created_at, id limit 1`, a).Scan(&after)
	})
	var got struct{ Scores []map[string]any }
	json.Unmarshal([]byte(after), &got)
	if len(got.Scores) != 2 || got.Scores[0]["userId"] != su || got.Scores[0]["points"] != nil || got.Scores[1]["points"] != 70.0 {
		t.Errorf("delete audit after: %s", after)
	}
}

func TestScoresAuthz(t *testing.T) {
	e := newEnv(t)
	a := e.assessment("Mid", "100", "40")
	for _, role := range roles {
		want := 403
		switch role {
		case "admin", "dain", "inst", "ta":
			want = 200
		}
		if st, raw := e.put(e.as(role), a, sc(e.UserID["stud@a.edu"], 50)); st != want {
			t.Errorf("%s: %d %s", role, st, raw)
		}
	}
}

func TestScoresAudit(t *testing.T) {
	e := newEnv(t)
	a := e.assessment("Mid", "100", "40")
	su, s2 := e.UserID["stud@a.edu"], e.UserID["stud2@a.edu"]
	e.put(e.Admin, a, sc(su, 80), sc(s2, 90))
	count := func() (n int) {
		e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
			return tx.QueryRow(`select count(*) from audit_log where action = 'score.set' and entity_type = 'assessment' and entity_id = $1`, a).Scan(&n)
		})
		return
	}
	if count() != 1 {
		t.Fatalf("%d audit rows", count())
	}
	rows := func() (before, after sql.NullString) {
		e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
			return tx.QueryRow(`select before::text, after::text from audit_log where action = 'score.set' order by created_at desc, id limit 1`).Scan(&before, &after)
		})
		return
	}
	b, af := rows()
	if b.Valid || !strings.Contains(af.String, `"assessmentId": "`+a+`"`) || !strings.Contains(af.String, "80.00") || !strings.Contains(af.String, "90.00") {
		t.Errorf("first: %v %v", b, af)
	}
	e.put(e.Admin, a, sc(su, 85))
	if count() != 2 {
		t.Fatalf("%d audit rows", count())
	}
	b, af = rows()
	if !b.Valid || !strings.Contains(b.String, "80.00") || strings.Contains(b.String, "90.00") || !strings.Contains(af.String, "85.00") {
		t.Errorf("second: %v %v", b, af)
	}
}

func TestUpsertScore(t *testing.T) {
	e := newEnv(t)
	a := e.assessment("Q", "10", "10")
	stud, inst := e.UserID["stud@a.edu"], e.UserID["inst@a.edu"]
	read := func() (pts, by string, at time.Time) {
		e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
			return tx.QueryRow(`select points::text, graded_by::text, graded_at from scores where assessment_id = $1 and user_id = $2`, a, stud).Scan(&pts, &by, &at)
		})
		return
	}
	up := func(pts, by string) {
		e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
			return UpsertScore(context.Background(), tx, e.Fx.UniversityA, a, stud, pts, by)
		})
	}
	up("7.50", stud)
	pts, by, at := read()
	if pts != "7.50" || by != stud {
		t.Fatalf("insert: %s %s", pts, by)
	}
	up("9.00", inst)
	pts, by, at2 := read()
	if pts != "9.00" || by != inst || at2.Before(at) {
		t.Fatalf("update: %s %s", pts, by)
	}
	var n int
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select count(*) from scores where assessment_id = $1`, a).Scan(&n)
	})
	if n != 1 {
		t.Fatalf("%d rows", n)
	}
}
