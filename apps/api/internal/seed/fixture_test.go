package seed

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kapildev333/unicourse/apps/api/internal/testdb"
)

const miniFixture = `{
  "university": {"name": "Mini University", "slug": "mini", "timezone": "UTC", "locale": "en-US"},
  "departments": [{"name": "Computer Science", "code": "CS"}],
  "terms": [{"name": "Fall <year>", "startsOn": "<year>-09-01", "endsOn": "<year>-12-20"}],
  "people": [{"email": "dean@<domain>", "displayName": "Dean Mini", "role": "admin"}],
  "courses": [{"departmentCode": "CS", "code": "CS1", "title": "Basics", "sections": [{"term": "Fall <year>", "number": "001",
    "weeks": [{"title": "Week 1", "notes": "n", "materials": [{"file": "hello.txt", "title": "Hello"}]}]}]}]
}`

func writeFixture(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "mini.json")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hi"), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadFileRunsAMiniFixture(t *testing.T) {
	f, err := LoadFile(writeFixture(t, miniFixture), "", 2031)
	if err != nil {
		t.Fatal(err)
	}
	_, db := testdb.OpenWithOwner(t)
	rep, err := Run(context.Background(), db, f, seedOpts(t))
	if err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, rep.UniversityID, `select count(*) from materials where file_name = 'hello.txt'`); n != 1 {
		t.Fatalf("materials from the fixture's own directory: %d", n)
	}
}

func TestLoadFileErrors(t *testing.T) {
	noName := strings.Replace(miniFixture, `"name": "Mini University", `, "", 1)
	if _, err := LoadFile(writeFixture(t, noName), "", 2031); err == nil || !strings.Contains(err.Error(), "university.name") || !strings.Contains(err.Error(), "mini.json") {
		t.Fatalf("missing key: %v", err)
	}
	noFile := strings.Replace(miniFixture, "hello.txt", "gone.txt", 1)
	if _, err := LoadFile(writeFixture(t, noFile), "", 2031); err == nil || !strings.Contains(err.Error(), "gone.txt") {
		t.Fatalf("missing material file: %v", err)
	}
	if _, err := LoadFile(filepath.Join(t.TempDir(), "nope.json"), "", 2031); err == nil {
		t.Fatal("missing fixture file loaded")
	}
}

// The schema is checked without a validator library, so the ceiling is structural: every key Load
// requires is in the matching `required` list, and every key of demo.json is declared under `properties`.
// Types, enums and patterns in the schema are not enforced here.
func TestSchemaMatchesLoad(t *testing.T) {
	raw, err := os.ReadFile("../../../../docs/seed/fixture.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}
	defs, _ := schema["$defs"].(map[string]any)
	// resolve follows $ref into $defs and unwraps array items.
	var resolve func(n map[string]any) map[string]any
	resolve = func(n map[string]any) map[string]any {
		if ref, ok := n["$ref"].(string); ok {
			return resolve(defs[strings.TrimPrefix(ref, "#/$defs/")].(map[string]any))
		}
		if items, ok := n["items"].(map[string]any); ok {
			return resolve(items)
		}
		return n
	}
	need := func(path string, keys ...string) {
		n := schema
		for _, p := range strings.Split(path, ".") {
			if p == "" {
				continue
			}
			n = resolve(n["properties"].(map[string]any)[p].(map[string]any))
		}
		req, _ := n["required"].([]any)
		for _, k := range keys {
			found := false
			for _, r := range req {
				found = found || r == k
			}
			if !found {
				t.Errorf("schema %q: %q is required by Load but not in required %v", path, k, req)
			}
		}
	}
	need("", "university")
	need("university", "name", "slug", "timezone", "locale")
	need("people", "email", "displayName", "role")
	need("courses", "departmentCode", "code")
	need("courses.sections", "term", "number")
	need("courses.sections.staff", "email", "role")
	need("courses.sections.weeks.materials", "file")

	var walk func(path string, node map[string]any, data any)
	walk = func(path string, node map[string]any, data any) {
		node = resolve(node)
		switch d := data.(type) {
		case []any:
			for _, v := range d {
				walk(path, node, v)
			}
		case map[string]any:
			props, _ := node["properties"].(map[string]any)
			for k, v := range d {
				p, ok := props[k].(map[string]any)
				if !ok {
					t.Errorf("demo.json key %s.%s is not declared in the schema", path, k)
					continue
				}
				walk(path+"."+k, p, v)
			}
		}
	}
	var demo any
	if err := json.Unmarshal(Demo, &demo); err != nil {
		t.Fatal(err)
	}
	walk("$", schema, demo)
}
