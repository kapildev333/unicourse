package content

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestDescription(t *testing.T) {
	e := newEnv(t)
	path := "/sections/" + e.SectionA + "/description"
	st, raw := e.call(e.Inst, "PUT", path, map[string]any{"description": "  Line 1\nLine 2  "})
	m := jmap(t, raw)
	if st != 200 || m["description"] != "Line 1\nLine 2" || m["id"] != e.SectionA || m["course"] == nil || m["term"] == nil || m["staff"] == nil || m["meetings"] == nil {
		t.Fatalf("inst: %d %s", st, raw)
	}
	if st, raw = e.call(e.TA, "PUT", path, map[string]any{"description": "From TA"}); st != 200 || jmap(t, raw)["description"] != "From TA" {
		t.Fatalf("ta: %d %s", st, raw)
	}
	if st, raw = e.call(e.Inst, "PUT", path, map[string]any{"description": ""}); st != 200 || jmap(t, raw)["description"] != "" {
		t.Fatalf("clear: %d %s", st, raw)
	}
	if st, raw = e.call(e.Inst, "PUT", path, map[string]any{"description": strings.Repeat("é", 10000)}); st != 200 {
		t.Fatalf("10000 runes: %d %s", st, raw)
	}
	for name, body := range map[string]any{
		"10001":   map[string]any{"description": strings.Repeat("é", 10001)},
		"missing": map[string]any{},
		"null":    map[string]any{"description": nil},
		"nul":     map[string]any{"description": "a\x00b"},
	} {
		st, raw := e.call(e.Inst, "PUT", path, body)
		if _, f := errDetails(raw); st != 400 || !reflect.DeepEqual(f, []string{"description"}) {
			t.Errorf("%s: %d %s", name, st, raw)
		}
	}
	if st, raw = e.call(e.Inst, "PUT", path, map[string]any{"description": "x", "other": 1}); st != 400 {
		t.Errorf("unknown key: %d %s", st, raw)
	}
	// audit: one section.update per successful PUT, before/after {description}
	var n int
	var before, after string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select count(*) from audit_log where action = 'section.update' and entity_type = 'section' and entity_id = $1`, e.SectionA).Scan(&n)
	})
	if n != 4 {
		t.Fatalf("audit rows = %d", n)
	}
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select before::text, after::text from audit_log where action = 'section.update' and entity_id = $1 order by created_at, id limit 1`, e.SectionA).Scan(&before, &after)
	})
	var b, a map[string]string
	json.Unmarshal([]byte(before), &b)
	json.Unmarshal([]byte(after), &a)
	if !reflect.DeepEqual(b, map[string]string{"description": ""}) || !reflect.DeepEqual(a, map[string]string{"description": "Line 1\nLine 2"}) {
		t.Fatalf("audit %s -> %s", before, after)
	}
}

func TestDescriptionAuthz(t *testing.T) {
	e := newEnv(t)
	rows := []struct {
		name, method, path string
		body               any
		want               [8]int
	}{
		{"GET section", "GET", "/sections/" + e.SectionA, nil, [8]int{200, 200, 200, 200, 200, 200, 200, 403}},
		{"PUT description", "PUT", "/sections/" + e.SectionA + "/description", map[string]any{"description": "x"}, [8]int{200, 200, 403, 200, 200, 403, 403, 403}},
	}
	for _, row := range rows {
		for i, role := range roles {
			if st, raw := e.call(e.as(role), row.method, row.path, row.body); st != row.want[i] {
				t.Errorf("%s as %s: %d, want %d (%s)", row.name, role, st, row.want[i], raw)
			}
		}
	}
	zero := "00000000-0000-0000-0000-000000000000"
	for _, p := range []string{zero, "x"} {
		if st, _ := e.call(e.Admin, "PUT", "/sections/"+p+"/description", map[string]any{"description": "x"}); st != 404 {
			t.Errorf("unknown section %s: %d", p, st)
		}
	}
	if st, _ := e.call(e.Admin, "PUT", "/sections/"+fmt.Sprint(e.SectionB())+"/description", map[string]any{"description": "x"}); st != 404 {
		t.Errorf("B's section: %d", st)
	}
}
