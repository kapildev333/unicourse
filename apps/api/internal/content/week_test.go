package content

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
)

type weekJSON struct {
	ID        string
	Title     string
	Notes     string
	Position  int
	Materials []matJSON
}

type matJSON struct {
	ID, WeekID, Title, FileName, ContentType string
	SizeBytes                                int64
	Position                                 int
	UploadedBy                               struct{ UserID, DisplayName, Email string }
}

func (e *env) weeks() []weekJSON {
	e.T.Helper()
	st, raw := e.call(e.Admin, "GET", "/sections/"+e.SectionA+"/weeks", nil)
	var b struct{ Items []weekJSON }
	if st != 200 || json.Unmarshal(raw, &b) != nil {
		e.T.Fatalf("list weeks: %d %s", st, raw)
	}
	return b.Items
}

// order returns the week titles by position and fails if positions are not 1..n.
func (e *env) order() string {
	e.T.Helper()
	var t []string
	for i, w := range e.weeks() {
		if w.Position != i+1 {
			e.T.Fatalf("positions not dense: %+v", e.weeks())
		}
		t = append(t, w.Title)
	}
	return strings.Join(t, ",")
}

func (e *env) weekPath(id string) string { return "/sections/" + e.SectionA + "/weeks/" + id }

func TestWeekCRUD(t *testing.T) {
	e := newEnv(t)
	var ids []string
	for i := 1; i <= 3; i++ {
		st, raw := e.call(e.Inst, "POST", "/sections/"+e.SectionA+"/weeks", map[string]any{"title": fmt.Sprint("Week ", i)})
		var w weekJSON
		if st != 201 || json.Unmarshal(raw, &w) != nil || w.Position != i || w.Notes != "" || !strings.Contains(string(raw), `"materials":[]`) {
			t.Fatalf("create %d: %d %s", i, st, raw)
		}
		ids = append(ids, w.ID)
	}
	st, raw := e.call(e.TA, "PATCH", e.weekPath(ids[1]), map[string]any{"title": " Intro ", "notes": "Read\nchapter 1"})
	var w weekJSON
	if st != 200 || json.Unmarshal(raw, &w) != nil || w.Title != "Intro" || w.Notes != "Read\nchapter 1" || w.Position != 2 {
		t.Fatalf("patch: %d %s", st, raw)
	}
	e.material(ids[0], "M1")
	got := e.weeks()
	if len(got) != 3 || got[1].Title != "Intro" || len(got[0].Materials) != 1 || got[0].Materials[0].UploadedBy.Email != "inst@a.edu" || got[0].Materials[0].UploadedBy.DisplayName != "N inst@a.edu" {
		t.Fatalf("list: %+v", got)
	}
	n, _, after := e.audit("week.create", ids[0])
	if n != 1 || !strings.Contains(after.String, `"position": 1`) {
		t.Fatalf("audit %d %s", n, after.String)
	}
	if n, b, a := e.audit("week.update", ids[1]); n != 1 || !strings.Contains(b.String, "Week 2") || !strings.Contains(a.String, "Intro") {
		t.Fatalf("audit update %d %s %s", n, b.String, a.String)
	}
}

func TestWeekMoveRenumbers(t *testing.T) {
	e := newEnv(t)
	var ids []string
	for i := 1; i <= 5; i++ {
		ids = append(ids, e.week(fmt.Sprint(i)))
	}
	move := func(i, pos int) int {
		st, _ := e.call(e.Inst, "PATCH", e.weekPath(ids[i-1]), map[string]any{"position": pos})
		return st
	}
	stamps := func() (m map[string]string) {
		m = map[string]string{}
		e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
			rows, err := tx.Query(`select title, updated_at::text from weeks`)
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
				var k, v string
				rows.Scan(&k, &v)
				m[k] = v
			}
			return rows.Err()
		})
		return
	}
	was := stamps()
	if st := move(2, 4); st != 200 {
		t.Fatal(st)
	}
	if now := stamps(); now["1"] != was["1"] || now["5"] != was["5"] || now["2"] == was["2"] || now["3"] == was["3"] {
		t.Fatalf("updated_at: %v -> %v", was, now)
	}
	if got := e.order(); got != "1,3,4,2,5" {
		t.Fatalf("after 2->4: %s", got)
	}
	if st := move(5, 1); st != 200 {
		t.Fatal(st)
	}
	if got := e.order(); got != "5,1,3,4,2" {
		t.Fatalf("after 5->1: %s", got)
	}
	for _, p := range []int{0, 6} {
		st, raw := e.call(e.Inst, "PATCH", e.weekPath(ids[0]), map[string]any{"position": p})
		if code, f := errDetails(raw); st != 400 || code != "validation_failed" || len(f) != 1 || f[0] != "position" || !strings.Contains(string(raw), "must be between 1 and 5") {
			t.Fatalf("position %d: %d %s", p, st, raw)
		}
	}
}

func TestWeekDelete(t *testing.T) {
	e := newEnv(t)
	w := []string{e.week("A"), e.week("B"), e.week("C"), e.week("D")}
	var ids []string
	for i := 0; i < 2; i++ {
		st, raw := e.upload(e.Inst, w[1], "f.pdf", strings.NewReader("data"), "")
		if st != 201 {
			t.Fatalf("%d %s", st, raw)
		}
		ids = append(ids, jmap(t, raw)["id"].(string))
	}
	for _, id := range ids {
		if _, err := os.Stat(e.blobPath(id)); err != nil {
			t.Fatal(err)
		}
	}
	if st, raw := e.call(e.TA, "DELETE", e.weekPath(w[1]), nil); st != 204 || len(raw) != 0 {
		t.Fatalf("delete: %d %s", st, raw)
	}
	if got := e.order(); got != "A,C,D" {
		t.Fatalf("order %s", got)
	}
	for _, id := range ids {
		if _, err := os.Stat(e.blobPath(id)); !os.IsNotExist(err) {
			t.Fatalf("blob %s survived: %v", id, err)
		}
	}
	if n := e.count(`select count(*) from materials`); n != 0 {
		t.Fatalf("%d material rows", n)
	}
	_, before, _ := e.audit("week.delete", w[1])
	if strings.Count(before.String, `"sha256"`) != 2 {
		t.Fatalf("audit before %s", before.String)
	}
}

func TestWeekLimitsAndValidation(t *testing.T) {
	e := newEnv(t)
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`insert into weeks (university_id, section_id, title, position) select $1, $2, 'w' || g, g from generate_series(1, 100) g`, e.Fx.UniversityA, e.SectionA)
		return err
	})
	post := func(b any) (int, []byte) { return e.call(e.Inst, "POST", "/sections/"+e.SectionA+"/weeks", b) }
	if st, raw := post(map[string]any{"title": "x"}); st != 400 || !strings.Contains(string(raw), `"weeks"`) {
		t.Fatalf("101st: %d %s", st, raw)
	}
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`delete from weeks where position > 1`)
		return err
	})
	for name, b := range map[string]map[string]any{
		"empty": {"title": ""}, "long": {"title": strings.Repeat("a", 101)}, "tab": {"title": "a\tb"}, "notes": {"title": "x", "notes": strings.Repeat("n", 5001)},
	} {
		if st, raw := post(b); st != 400 {
			t.Errorf("%s: %d %s", name, st, raw)
		}
	}
	id := e.week("V")
	if st, raw := e.call(e.Inst, "PATCH", e.weekPath(id), map[string]any{}); st != 400 {
		t.Fatalf("{}: %d %s", st, raw)
	} else if _, f := errDetails(raw); len(f) != 1 || f[0] != "" {
		t.Fatalf("{} fields %v", f)
	}
	for _, f := range []string{"title", "notes", "position"} {
		st, raw := e.call(e.Inst, "PATCH", e.weekPath(id), map[string]any{f: nil})
		if _, fs := errDetails(raw); st != 400 || len(fs) != 1 || fs[0] != f || !strings.Contains(string(raw), "must not be null") {
			t.Errorf("null %s: %d %s", f, st, raw)
		}
	}
	// another section's week, a malformed id and an unknown one are 404
	var other string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`insert into weeks (university_id, section_id, title, position) values ($1, $2, 'o', 1) returning id`, e.Fx.UniversityA, e.SectionA2).Scan(&other)
	})
	for _, p := range []string{other, "x", "00000000-0000-0000-0000-000000000000"} {
		for _, m := range []string{"PATCH", "DELETE"} {
			if st, _ := e.call(e.Inst, m, e.weekPath(p), map[string]any{"title": "t"}); st != 404 {
				t.Errorf("%s %s: %d", m, p, st)
			}
		}
	}
}

func TestWeekConcurrentCreates(t *testing.T) {
	e := newEnv(t)
	var wg sync.WaitGroup
	sts := make([]int, 10)
	for i := range sts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sts[i], _ = e.call(e.Inst, "POST", "/sections/"+e.SectionA+"/weeks", map[string]any{"title": fmt.Sprint("w", i)})
		}()
	}
	wg.Wait()
	for i, st := range sts {
		if st != 201 {
			t.Fatalf("create %d: %d", i, st)
		}
	}
	if n := len(e.weeks()); n != 10 {
		t.Fatalf("%d weeks", n)
	}
	e.order() // fails unless positions are exactly 1..10
}

func TestWeekAuthz(t *testing.T) {
	e := newEnv(t)
	sec := "/sections/" + e.SectionA
	ok := [8]int{200, 200, 403, 200, 200, 200, 403, 403}
	edit := func(success int) [8]int { return [8]int{success, success, 403, success, success, 403, 403, 403} }
	// each mutating cell gets a fresh week so earlier deletes do not turn later cells into 404s
	for _, row := range []struct {
		name, method string
		path         func() string
		body         any
		want         [8]int
	}{
		{"GET weeks", "GET", func() string { return sec + "/weeks" }, nil, ok},
		{"POST weeks", "POST", func() string { return sec + "/weeks" }, map[string]any{"title": "t"}, edit(201)},
		{"PATCH week", "PATCH", func() string { return e.weekPath(e.week("p")) }, map[string]any{"title": "t"}, edit(200)},
		{"DELETE week", "DELETE", func() string { return e.weekPath(e.week("d")) }, nil, edit(204)},
	} {
		for i, role := range roles {
			if st, raw := e.call(e.as(role), row.method, row.path(), row.body); st != row.want[i] {
				t.Errorf("%s as %s: %d, want %d (%s)", row.name, role, st, row.want[i], raw)
			}
		}
	}
	if st, _ := e.call(e.Stud, "GET", "/sections/"+e.SectionA2+"/weeks", nil); st != 403 {
		t.Errorf("Stud on SectionA2: %d", st)
	}
	for _, p := range []string{"00000000-0000-0000-0000-000000000000", "x", e.SectionB()} {
		if st, _ := e.call(e.Admin, "GET", "/sections/"+p+"/weeks", nil); st != 404 {
			t.Errorf("section %s: %d", p, st)
		}
	}
}
