package grade

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

const usScale = `{"rows":[{"letter":"A","minPercent":93.00,"points":4.00},{"letter":"A-","minPercent":90.00,"points":3.70},{"letter":"B+","minPercent":87.00,"points":3.30},{"letter":"B","minPercent":83.00,"points":3.00},{"letter":"B-","minPercent":80.00,"points":2.70},{"letter":"C+","minPercent":77.00,"points":2.30},{"letter":"C","minPercent":73.00,"points":2.00},{"letter":"C-","minPercent":70.00,"points":1.70},{"letter":"D+","minPercent":67.00,"points":1.30},{"letter":"D","minPercent":63.00,"points":1.00},{"letter":"D-","minPercent":60.00,"points":0.70},{"letter":"F","minPercent":0.00,"points":0.00}]}`

func TestScaleSeeded(t *testing.T) {
	e := newEnv(t)
	st, raw := e.raw(e.Stud, "GET", "/grade-scale", nil)
	if st != 200 || strings.TrimSpace(raw) != usScale {
		t.Fatalf("%d %s", st, raw)
	}
}

func canadian() map[string]any {
	type r = map[string]any
	return map[string]any{"rows": []r{
		{"letter": "A+", "minPercent": 90, "points": 4.3}, {"letter": "A", "minPercent": 85, "points": 4.0}, {"letter": "A-", "minPercent": 80, "points": 3.7},
		{"letter": "B+", "minPercent": 77, "points": 3.3}, {"letter": "B", "minPercent": 73, "points": 3.0}, {"letter": "B-", "minPercent": 70, "points": 2.7},
		{"letter": "C+", "minPercent": 67, "points": 2.3}, {"letter": "C", "minPercent": 63, "points": 2.0}, {"letter": "C-", "minPercent": 60, "points": 1.7},
		{"letter": "D+", "minPercent": 57, "points": 1.3}, {"letter": "D", "minPercent": 53, "points": 1.0}, {"letter": "D-", "minPercent": 50, "points": 0.7},
		{"letter": "F", "minPercent": 0, "points": 0},
	}}
}

func TestScalePut(t *testing.T) {
	e := newEnv(t)
	st, raw := e.raw(e.Admin, "PUT", "/grade-scale", canadian())
	if st != 200 || !strings.HasPrefix(raw, `{"rows":[{"letter":"A+","minPercent":90.00,"points":4.30}`) || !strings.Contains(raw, `{"letter":"F","minPercent":0.00,"points":0.00}]}`) {
		t.Fatalf("put: %d %s", st, raw)
	}
	if st2, raw2 := e.raw(e.Stud, "GET", "/grade-scale", nil); st2 != 200 || raw2 != raw {
		t.Fatalf("get: %d %s", st2, raw2)
	}
	n, before, after := e.audit("grade_scale.update", e.Fx.UniversityA)
	var b, a struct{ Rows []any }
	json.Unmarshal([]byte(before.String), &b)
	json.Unmarshal([]byte(after.String), &a)
	if n != 1 || len(b.Rows) != 12 || len(a.Rows) != 13 {
		t.Fatalf("audit %d before %d after %d", n, len(b.Rows), len(a.Rows))
	}
}

func TestScaleValidation(t *testing.T) {
	e := newEnv(t)
	row := func(l string, m, p any) map[string]any {
		return map[string]any{"letter": l, "minPercent": m, "points": p}
	}
	rows := func(r ...map[string]any) map[string]any { return map[string]any{"rows": r} }
	many := func(n int) map[string]any {
		var r []map[string]any
		for i := 0; i < n; i++ {
			r = append(r, row(fmt.Sprint("L", i), i, 1))
		}
		return rows(r...)
	}
	for name, c := range map[string]struct {
		body any
		want []string
	}{
		"empty":        {map[string]any{"rows": []any{}}, []string{"rows"}},
		"51":           {many(51), []string{"rows"}},
		"dup letter":   {rows(row("A", 90, 4), row(" A ", 0, 0)), []string{"rows[1].letter"}},
		"dup min":      {rows(row("A", 90, 4), row("B", 90.00, 3), row("C", 0, 0)), []string{"rows[1].minPercent"}},
		"no zero":      {rows(row("A", 90, 4)), []string{"rows"}},
		"min range":    {rows(row("A", 100.01, 4), row("B", 0, 0)), []string{"rows[0].minPercent"}},
		"min decimals": {rows(row("A", 90.123, 4), row("B", 0, 0)), []string{"rows[0].minPercent"}},
		"points":       {rows(row("A", 90, 10), row("B", 0, 0)), []string{"rows[0].points"}},
		"letter empty": {rows(row("", 90, 4), row("B", 0, 0)), []string{"rows[0].letter"}},
		"letter long":  {rows(row(strings.Repeat("é", 11), 90, 4), row("B", 0, 0)), []string{"rows[0].letter"}},
		"unknown key":  {map[string]any{"rows": []any{map[string]any{"letter": "A", "minPercent": 0, "points": 0, "x": 1}}}, []string{"x"}},
		"together":     {rows(row("", 90, 4), row("B", 0, 10)), []string{"rows[0].letter", "rows[1].points"}},
	} {
		st, raw := e.call(e.Admin, "PUT", "/grade-scale", c.body)
		_, f := errDetails(raw)
		if st != 400 || !reflect.DeepEqual(f, c.want) {
			t.Errorf("%s: %d %s", name, st, raw)
		}
		if _, got := e.raw(e.Admin, "GET", "/grade-scale", nil); strings.TrimSpace(got) != usScale {
			t.Fatalf("%s: stored scale changed by a 400: %s", name, got)
		}
	}
	if st, raw := e.call(e.Admin, "PUT", "/grade-scale", many(50)); st != 200 {
		t.Fatalf("50 rows: %d %s", st, raw)
	}
}

func TestScaleAuthz(t *testing.T) {
	e := newEnv(t)
	for _, role := range roles {
		if st, raw := e.raw(e.as(role), "GET", "/grade-scale", nil); st != 200 {
			t.Errorf("GET %s: %d %s", role, st, raw)
		}
		want := 403
		if role == "admin" {
			want = 200
		}
		if st, raw := e.raw(e.as(role), "PUT", "/grade-scale", canadian()); st != want {
			t.Errorf("PUT %s: %d %s", role, st, raw)
		}
	}
	if st, _ := e.raw(e.cookie(e.Fx.AdminB), "PUT", "/grade-scale", canadian()); st != 403 {
		t.Errorf("B's admin on slug a: %d", st)
	}
}
