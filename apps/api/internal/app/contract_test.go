package app

import (
	"bufio"
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

func pendingRoutes(t *testing.T) map[string]bool {
	t.Helper()
	f, err := os.Open("pending_routes.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	out := map[string]bool{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if l := strings.TrimSpace(sc.Text()); l != "" && !strings.HasPrefix(l, "#") {
			out[norm(l)] = true
		}
	}
	return out
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
	code := map[string]bool{}
	for _, r := range NewRouter(Deps{}).Routes() {
		code[norm(r.Method+" /api/v1"+strings.TrimPrefix(r.Pattern, "/api/v1"))] = true
	}
	pending := pendingRoutes(t)

	for _, k := range sorted(code) {
		if !spec[k] {
			t.Errorf("route in Go but not in openapi.json: %s", k)
		}
		if pending[k] {
			t.Errorf("route is registered but still in pending_routes.txt: %s", k)
		}
	}
	for _, k := range sorted(spec) {
		if !code[k] && !pending[k] {
			t.Errorf("route in openapi.json but not registered in Go (and not pending): %s", k)
		}
	}
	for _, k := range sorted(pending) {
		if !spec[k] {
			t.Errorf("pending_routes.txt line not in openapi.json: %s", k)
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
