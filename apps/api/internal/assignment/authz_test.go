package assignment

import (
	"strings"
	"testing"
)

// TestAssignmentAuthz is the submission rows of the design 7 role matrix: roles are admin, dain, daout, inst, ta, stud, other, badm.
func TestAssignmentAuthz(t *testing.T) {
	e := newEnv(t)
	a := e.open()
	stud := e.UserID["stud@a.edu"]
	e.put(e.Stud, a, "work")
	_, raw := e.upload(e.Stud, a, "a.txt", strings.NewReader("x"))
	file := fileIDs(t, raw)[0]
	userPath := e.subPath(a, "/"+stud)

	check := func(name string, want [8]int, do func(role string) int) {
		t.Helper()
		for i, role := range roles {
			if got := do(role); got != want[i] {
				t.Errorf("%s as %s: %d, want %d", name, role, got, want[i])
			}
		}
	}
	call := func(method, path string, body any) func(string) int {
		return func(role string) int { st, _ := e.call(e.as(role), method, path, body); return st }
	}
	// admin dain daout inst ta stud other badm
	check("GET list", [8]int{200, 200, 403, 200, 200, 403, 403, 403}, call("GET", e.subPath(a, ""), nil))
	check("GET {userId}", [8]int{200, 200, 403, 200, 200, 200, 403, 403}, call("GET", userPath, nil))
	check("GET me", [8]int{404, 404, 403, 404, 404, 200, 403, 403}, call("GET", e.subPath(a, "/me"), nil))
	check("PATCH {userId}", [8]int{200, 200, 403, 200, 200, 403, 403, 403}, call("PATCH", userPath, map[string]any{"score": 10}))
	check("PUT me", [8]int{403, 403, 403, 403, 403, 200, 403, 403}, call("PUT", e.subPath(a, "/me"), map[string]any{"body": "again"}))
	check("POST file", [8]int{403, 403, 403, 403, 403, 201, 403, 403}, func(role string) int {
		st, _ := e.upload(e.as(role), a, "b.txt", strings.NewReader("y"))
		return st
	})
	check("download", [8]int{200, 200, 403, 200, 200, 200, 403, 403}, func(role string) int {
		p := userPath
		if role == "stud" {
			p = e.subPath(a, "/me")
		}
		res, _ := e.get(e.as(role), p+"/files/"+file+"/download", nil)
		return res.StatusCode
	})
	check("DELETE me file", [8]int{403, 403, 403, 403, 403, 204, 403, 403}, func(role string) int {
		f := file
		if role == "stud" {
			_, raw := e.call(e.Stud, "GET", e.subPath(a, "/me"), nil)
			f = jmap(t, raw)["files"].([]any)[1].(map[string]any)["id"].(string)
		}
		st, _ := e.call(e.as(role), "DELETE", e.subPath(a, "/me/files/"+f), nil)
		return st
	})

	// stud2 (enrolled) on stud's rows: 403 everywhere; on its own rows it works
	for name, do := range map[string]func() int{
		"GET":      func() int { st, _ := e.call(e.Stud2, "GET", userPath, nil); return st },
		"PATCH":    func() int { st, _ := e.call(e.Stud2, "PATCH", userPath, map[string]any{"score": 1}); return st },
		"download": func() int { res, _ := e.get(e.Stud2, userPath+"/files/"+file+"/download", nil); return res.StatusCode },
		"list":     func() int { st, _ := e.call(e.Stud2, "GET", e.subPath(a, ""), nil); return st },
	} {
		if got := do(); got != 403 {
			t.Errorf("stud2 %s: %d", name, got)
		}
	}
	// a student reading its own row by id equals "me"; unknown ids and sections are 404 for everyone allowed
	if st, _ := e.call(e.Stud, "GET", e.subPath(a, "/"+stud), nil); st != 200 {
		t.Errorf("own id: %d", st)
	}
	for name, p := range map[string]string{
		"bad section": "/sections/x/assessments/" + a + "/submissions",
		"other kind":  "/sections/" + e.SectionA + "/assessments/" + e.assessmentManual() + "/submissions",
		"bad id":      "/sections/" + e.SectionA + "/assessments/x/submissions",
		"foreign":     "/sections/" + e.SectionA2 + "/assessments/" + a + "/submissions",
	} {
		for _, role := range []string{"admin", "stud", "other"} {
			if st, _ := e.call(e.as(role), "GET", p, nil); st != 404 {
				t.Errorf("%s as %s: %d", name, role, st)
			}
		}
	}
	// editors: a non-uuid user is 404, an unknown uuid 404
	for _, u := range []string{"x", "00000000-0000-0000-0000-000000000000"} {
		if st, _ := e.call(e.Inst, "GET", e.subPath(a, "/"+u), nil); st != 404 {
			t.Errorf("user %s: %d", u, st)
		}
	}
}
