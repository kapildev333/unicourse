package course

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"
)

func TestEnrollmentsListAndDelete(t *testing.T) {
	e := newEnv(t)
	path := "/sections/" + e.SectionA + "/enrollments"
	for _, c := range []struct {
		name string
		ck   *http.Cookie
		want int
	}{{"admin", e.Admin, 200}, {"dain", e.DaIn, 200}, {"inst", e.Inst, 200}, {"ta", e.TA, 200}, {"stud", e.Stud, 403}, {"other", e.Other, 403}, {"daout", e.DaOut, 403}} {
		if st, raw := e.call(c.ck, "GET", path, nil); st != c.want {
			t.Errorf("%s list: %d %s", c.name, st, raw)
		}
	}
	st, raw := e.call(e.TA, "GET", path, nil)
	var l struct{ Items []Enrollment }
	if st != 200 || json.Unmarshal(raw, &l) != nil || len(l.Items) != 1 || l.Items[0].UserID != e.UserID["stud@a.edu"] || l.Items[0].Email != "stud@a.edu" || l.Items[0].CreatedAt.IsZero() {
		t.Fatalf("shape: %d %s", st, raw)
	}
	stud := e.UserID["stud@a.edu"]
	if st, _ := e.call(e.Inst, "DELETE", path+"/"+stud, nil); st != 403 {
		t.Errorf("instructor delete: %d", st)
	}
	for _, u := range []string{"nope", "00000000-0000-0000-0000-000000000000", e.UserID["other@a.edu"]} {
		if st, _ := e.call(e.DaIn, "DELETE", path+"/"+u, nil); st != 404 {
			t.Errorf("unknown %s: %d", u, st)
		}
	}
	if st, _ := e.call(e.DaIn, "GET", "/sections/nope/enrollments", nil); st != 404 {
		t.Errorf("unknown section: %d", st)
	}
	var rowID string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select id from enrollments where user_id = $1`, stud).Scan(&rowID)
	})
	if st, raw := e.call(e.DaIn, "DELETE", path+"/"+stud, nil); st != 204 {
		t.Fatalf("delete: %d %s", st, raw)
	}
	if n, before, _ := e.audit("enrollment.delete", rowID); n != 1 || !before.Valid {
		t.Errorf("audit: %d %v", n, before)
	}
	if st, _ := e.call(e.DaIn, "DELETE", path+"/"+stud, nil); st != 404 {
		t.Errorf("second delete: %d", st)
	}
}
