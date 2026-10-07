package app

import (
	"encoding/json"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var param = regexp.MustCompile(`\{[^}]*\}`)

func norm(s string) string { return param.ReplaceAllString(s, "{}") }

func readJSON(t *testing.T) map[string]any {
	t.Helper()
	b, err := os.ReadFile("../../openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

// pending: routes in openapi.json that a later P7-L1 sub-lane registers. Empty when P7-L1c lands.
var pending = map[string]bool{
	"PUT /api/v1/universities/{}/grade-scale":                                       true, // L1a, removed as it lands
	"GET /api/v1/universities/{}/grade-scale":                                       true, // L1a
	"GET /api/v1/universities/{}/sections/{}/assessments":                           true, // L1a
	"POST /api/v1/universities/{}/sections/{}/assessments":                          true, // L1a
	"PATCH /api/v1/universities/{}/sections/{}/assessments/{}":                      true, // L1a
	"DELETE /api/v1/universities/{}/sections/{}/assessments/{}":                     true, // L1a
	"PUT /api/v1/universities/{}/sections/{}/assessments/{}/scores":                 true, // L1a
	"GET /api/v1/universities/{}/sections/{}/gradebook":                             true, // L1b
	"GET /api/v1/universities/{}/sections/{}/final-grades":                          true, // L1b
	"POST /api/v1/universities/{}/sections/{}/final-grades":                         true, // L1b
	"DELETE /api/v1/universities/{}/sections/{}/final-grades":                       true, // L1b
	"PATCH /api/v1/universities/{}/sections/{}/final-grades/{}":                     true, // L1b
	"GET /api/v1/universities/{}/me/grades":                                         true, // L1b
	"POST /api/v1/universities/{}/sections/{}/assessments/{}/scores/import/preview": true, // L1c
	"POST /api/v1/universities/{}/sections/{}/assessments/{}/scores/import":         true, // L1c
}

func TestContractMatchesRouter(t *testing.T) {
	doc := readJSON(t)
	servers := doc["servers"].([]any)
	if u := servers[0].(map[string]any)["url"]; u != "/api/v1" {
		t.Fatalf("servers[0].url = %v, want /api/v1", u)
	}
	spec := map[string]bool{}
	ids := map[string]string{}
	for path, item := range doc["paths"].(map[string]any) {
		for method, op := range item.(map[string]any) {
			if method == "parameters" {
				continue
			}
			key := norm(strings.ToUpper(method) + " /api/v1" + path)
			spec[key] = true
			id, _ := op.(map[string]any)["operationId"].(string)
			if id == "" {
				t.Errorf("%s has no operationId", key)
			} else if prev, dup := ids[id]; dup {
				t.Errorf("operationId %q used by %s and %s", id, prev, key)
			}
			ids[id] = key
		}
	}
	code := []string{}
	for _, r := range NewRouter(Deps{}).Routes() {
		code = append(code, r.Method+" "+r.Pattern)
	}
	for _, p := range diff(spec, code, pending) {
		t.Error(p)
	}
}

// diff returns every mismatch between the contract set, the routes as
// registered ("METHOD /pattern") and the pending list.
func diff(spec map[string]bool, routes []string, pending map[string]bool) []string {
	var out []string
	code := map[string]bool{}
	for _, r := range routes {
		code[norm(r)] = true
	}
	for _, k := range sorted(code) {
		if !spec[k] {
			out = append(out, "route in Go but not in openapi.json: "+k)
		}
		if pending[k] {
			out = append(out, "route is registered but also in the pending set: "+k)
		}
	}
	for _, k := range sorted(spec) {
		if !code[k] && !pending[k] {
			out = append(out, "route in openapi.json but not registered in Go (and not pending): "+k)
		}
	}
	for _, k := range sorted(pending) {
		if !spec[k] {
			out = append(out, "pending entry not in openapi.json: "+k)
		}
	}
	return out
}

func TestDiff(t *testing.T) {
	spec := map[string]bool{"GET /api/v1/a": true, "GET /api/v1/b/{}": true}
	set := func(v ...string) map[string]bool {
		m := map[string]bool{}
		for _, x := range v {
			m[x] = true
		}
		return m
	}
	cases := []struct {
		name    string
		routes  []string
		pending map[string]bool
		want    string // substring of the single expected problem, "" for none
	}{
		{"ok", []string{"GET /api/v1/a", "GET /api/v1/b/{id}"}, set(), ""},
		{"ok pending", []string{"GET /api/v1/a"}, set("GET /api/v1/b/{}"), ""},
		{"go only", []string{"GET /api/v1/a", "GET /api/v1/b/{id}", "GET /api/v1/c"}, set(), "in Go but not in openapi.json: GET /api/v1/c"},
		{"contract only", []string{"GET /api/v1/a"}, set(), "not registered in Go (and not pending): GET /api/v1/b/{}"},
		{"pending but registered", []string{"GET /api/v1/a", "GET /api/v1/b/{id}"}, set("GET /api/v1/a"), "also in the pending set: GET /api/v1/a"},
		{"pending not in contract", []string{"GET /api/v1/a", "GET /api/v1/b/{id}"}, set("GET /api/v1/zzz"), "not in openapi.json: GET /api/v1/zzz"},
		{"missing prefix", []string{"GET /api/v1/a", "GET /b/{id}"}, set("GET /api/v1/b/{}"), "in Go but not in openapi.json: GET /b/{}"},
	}
	for _, c := range cases {
		got := diff(spec, c.routes, c.pending)
		if c.want == "" && len(got) != 0 || c.want != "" && (len(got) != 1 || !strings.Contains(got[0], c.want)) {
			t.Errorf("%s: got %v, want %q", c.name, got, c.want)
		}
	}
}

func TestRefsResolve(t *testing.T) {
	doc := readJSON(t)
	var walk func(v any)
	walk = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			if ref, ok := x["$ref"].(string); ok {
				var cur any = doc
				for _, seg := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
					m, ok := cur.(map[string]any)
					if !ok || m[seg] == nil {
						t.Errorf("unresolved $ref %s", ref)
						return
					}
					cur = m[seg]
				}
			}
			for _, c := range x {
				walk(c)
			}
		case []any:
			for _, c := range x {
				walk(c)
			}
		}
	}
	walk(doc)
}

func sorted(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
