package seed

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestLoadDemo(t *testing.T) {
	f, err := Load(Demo, "", 2031)
	if err != nil {
		t.Fatal(err)
	}
	if f.Terms[0].Name != "Academic Year 2031" || f.Terms[1].Name != "Spring 2032" || f.Terms[0].StartsOn != "2031-01-01" {
		t.Fatalf("terms: %+v", f.Terms)
	}
	for _, p := range f.People {
		if !strings.HasSuffix(p.Email, "@northfield.edu") {
			t.Fatalf("email %q", p.Email)
		}
	}
	g, err := Load(Demo, "acme", 2031)
	if err != nil {
		t.Fatal(err)
	}
	if g.University.Slug != "acme" {
		t.Fatalf("slug %q", g.University.Slug)
	}
	for _, p := range g.People {
		if !strings.HasSuffix(p.Email, "@acme.edu") {
			t.Fatalf("email %q", p.Email)
		}
	}
}

// mutate applies fn to the decoded demo.json and returns the re-encoded bytes.
func mutate(t *testing.T, fn func(m map[string]any)) []byte {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(Demo, &m); err != nil {
		t.Fatal(err)
	}
	fn(m)
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false) // keep the <domain> placeholders literal
	if err := enc.Encode(m); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func at(m map[string]any, key string, i int) map[string]any {
	return m[key].([]any)[i].(map[string]any)
}

func TestLoadRejects(t *testing.T) {
	sec := func(m map[string]any, c, s int) map[string]any {
		return at(at(m, "courses", c), "sections", s)
	}
	cases := []struct {
		name string
		fn   func(m map[string]any)
		want string
	}{
		{"unknown top-level", func(m map[string]any) { m["extra"] = 1 }, "unknown field"},
		{"unknown nested", func(m map[string]any) { sec(m, 0, 0)["extra"] = 1 }, "unknown field"},
		{"bad department", func(m map[string]any) { at(m, "courses", 0)["departmentCode"] = "XX" }, "departmentCode"},
		{"bad term", func(m map[string]any) { sec(m, 0, 0)["term"] = "Nope" }, "term"},
		{"roster not in people", func(m map[string]any) {
			s := sec(m, 0, 0)
			s["roster"] = append(s["roster"].([]any), "ghost@<domain>")
		}, "roster"},
		{"score outside roster", func(m map[string]any) {
			a := sec(m, 1, 0)["assessments"].([]any)[0].(map[string]any)
			a["scores"] = append(a["scores"].([]any), map[string]any{"student": "jonas.berg@<domain>", "points": 1})
		}, "scores"},
		{"override outside roster", func(m map[string]any) {
			f := sec(m, 0, 0)["finalize"].(map[string]any)
			f["overrides"] = append(f["overrides"].([]any), map[string]any{"student": "ingrid.larsen@<domain>", "reason": "x"})
		}, "overrides"},
		{"duplicate email", func(m map[string]any) { at(m, "people", 3)["email"] = at(m, "people", 2)["email"] }, "email"},
		{"second admin", func(m map[string]any) { at(m, "people", 2)["role"] = "admin" }, "admin"},
		{"dept_admin without departments", func(m map[string]any) { delete(at(m, "people", 1), "departments") }, "departments"},
		{"invited in roster", func(m map[string]any) {
			s := sec(m, 0, 0)
			s["roster"] = append(s["roster"].([]any), "ingrid.larsen@<domain>")
		}, "invited"},
		{"missing material file", func(m map[string]any) {
			sec(m, 0, 0)["weeks"].([]any)[0].(map[string]any)["materials"].([]any)[0].(map[string]any)["file"] = "nope.pdf"
		}, "file"},
		{"no instructor", func(m map[string]any) {
			sec(m, 0, 0)["staff"] = []any{map[string]any{"email": "sofia.alvarez@<domain>", "role": "ta"}}
		}, "instructor"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := Load(mutate(t, c.fn), "", 2031)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want error containing %q, got %v", c.want, err)
			}
		})
	}
	if _, err := Load(append(append([]byte{}, Demo...), []byte("{}")...), "", 2031); err == nil || !strings.Contains(err.Error(), "demo.json") {
		t.Fatalf("trailing value: %v", err)
	}
}
