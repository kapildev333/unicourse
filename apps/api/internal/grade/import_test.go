package grade

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
)

type prow struct {
	Line    int
	Action  string
	Message string
	Data    map[string]string
}
type pview struct {
	Checksum string
	Rows     []prow
	Counts   map[string]int
}

// upload posts a real multipart body; checksum "" omits the field.
func (e *env) upload(c *http.Cookie, path, csv, checksum string) (int, []byte) {
	e.T.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	if checksum != "" {
		w.WriteField("checksum", checksum)
	}
	fw, _ := w.CreateFormFile("file", "x.csv")
	io.WriteString(fw, csv)
	w.Close()
	req, _ := http.NewRequest("POST", e.Srv.URL+root+path, &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if c != nil {
		req.AddCookie(c)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.T.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	return res.StatusCode, raw
}

func (e *env) preview(c *http.Cookie, path, csv string) pview {
	e.T.Helper()
	st, raw := e.upload(c, path+"/preview", csv, "")
	if st != 200 {
		e.T.Fatalf("preview %s: %d %s", path, st, raw)
	}
	var p pview
	if err := json.Unmarshal(raw, &p); err != nil {
		e.T.Fatal(err)
	}
	return p
}

func actions(p pview) string {
	var out []string
	for _, r := range p.Rows {
		out = append(out, fmt.Sprintf("%d:%s:%s", r.Line, r.Action, r.Message))
	}
	return strings.Join(out, "|")
}

func (e *env) importPath(aid string) string {
	return "/sections/" + e.SectionA + "/assessments/" + aid + "/scores/import"
}

func (e *env) points(aid, email string) (p string) {
	e.T.Helper()
	_, raw := e.raw(e.Admin, "GET", "/sections/"+e.SectionA+"/gradebook", nil)
	var g struct {
		Students []struct {
			Email  string
			Scores map[string]json.Number
		}
	}
	json.Unmarshal([]byte(raw), &g)
	for _, s := range g.Students {
		if s.Email == email {
			return string(s.Scores[aid])
		}
	}
	return ""
}

func TestScoresImport(t *testing.T) {
	e := newEnv(t)
	aid := e.assessment("Midterm", "100", "40")
	e.score(aid, "stud@a.edu", "80")
	path := e.importPath(aid)
	csv := "email,points\nstud@a.edu,90\nstud2@a.edu,75\n STUD@a.edu,1\nbad,1\nnobody@a.edu,1\nother@a.edu,1\n" +
		"stud2@a.edu,1\n"
	p := e.preview(e.Admin, path, csv)
	want := "2:update:80.00 → 90.00|3:add:|4:error:duplicate of line 2|5:error:invalid email|6:error:unknown email|7:error:not enrolled in this section|8:error:duplicate of line 3"
	if got := actions(p); got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
	if st, raw := e.upload(e.Admin, path, csv, p.Checksum); st != 400 {
		t.Fatalf("commit with errors: %d %s", st, raw)
	} else if _, f := errDetails(raw); strings.Join(f, ",") != "line:4,line:5,line:6,line:7,line:8" {
		t.Fatalf("details %v", f)
	}
	for _, c := range [][2]string{{"x", "points must be a number"}, {"-1", "points must be between 0 and 200.00"}, {"200.01", "points must be between 0 and 200.00"},
		{"1.234", "points must have at most 2 decimals"}, {"", "points is required"}} {
		p := e.preview(e.Admin, path, "email,points\nstud@a.edu,"+c[0]+"\n")
		if got := actions(p); got != "2:error:"+c[1] {
			t.Errorf("points %q: %s", c[0], got)
		}
	}
	if e.points(aid, "stud@a.edu") != "80.00" || e.points(aid, "stud2@a.edu") != "" {
		t.Fatal("rejected commit wrote")
	}
	clean := "email,points\nstud@a.edu,90\nstud2@a.edu,75\n"
	p = e.preview(e.Admin, path, clean)
	if st, raw := e.upload(e.Admin, path, clean, ""); st != 400 {
		t.Fatalf("missing checksum: %d %s", st, raw)
	}
	if st, raw := e.upload(e.Admin, path, clean+"\n", p.Checksum); st != 400 {
		t.Fatalf("changed file must fail: %d %s", st, raw)
	}
	st, raw := e.upload(e.Inst, path, clean, p.Checksum)
	if st != 200 {
		t.Fatalf("commit: %d %s", st, raw)
	}
	if !strings.Contains(string(raw), `"counts":{"add":1,"error":0,"exists":0,"update":1}`) {
		t.Fatalf("counts: %s", raw)
	}
	if e.points(aid, "stud@a.edu") != "90.00" || e.points(aid, "stud2@a.edu") != "75.00" {
		t.Fatal("commit did not write")
	}
	n, before, after := e.audit("score.set", aid)
	if n != 1 || !strings.Contains(before.String, "80.00") || !strings.Contains(after.String, "75.00") {
		t.Fatalf("audit: %d %s %s", n, before.String, after.String)
	}
	bSec, bAssessment := e.foreign() // another university's assessment, under its own section and under A's
	for _, path := range []string{e.importPath(bAssessment), "/sections/" + bSec + "/assessments/" + bAssessment + "/scores/import"} {
		if st, raw := e.upload(e.Admin, path+"/preview", clean, ""); st != 404 {
			t.Errorf("foreign assessment %s: %d %s", path, st, raw)
		}
	}
}

func TestScoresImportExists(t *testing.T) {
	e := newEnv(t)
	aid := e.assessment("Midterm", "100", "40")
	e.score(aid, "stud@a.edu", "80")
	path := e.importPath(aid)
	csv := "email,points\nstud@a.edu,80\n"
	p := e.preview(e.Admin, path, csv)
	if actions(p) != "2:exists:" {
		t.Fatal(actions(p))
	}
	if st, raw := e.upload(e.Admin, path, csv, p.Checksum); st != 200 {
		t.Fatalf("%d %s", st, raw)
	}
	if n, _, _ := e.audit("score.set", aid); n != 0 {
		t.Errorf("exists-only commit audited %d", n)
	}
}

func TestScoresImportAuthz(t *testing.T) {
	e := newEnv(t)
	aid := e.assessment("Midterm", "100", "40")
	want := map[string]int{"admin": 200, "dain": 200, "daout": 403, "inst": 200, "ta": 200, "stud": 403, "other": 403}
	csv := "email,points\nstud@a.edu,90\n"
	for _, role := range roles {
		st, raw := e.upload(e.as(role), e.importPath(aid)+"/preview", csv, "")
		if st != want[role] {
			t.Errorf("preview %s: %d %s", role, st, raw)
		}
		// the commit without a checksum is a 400 for everyone who passes the authz check
		wantCommit := 400
		if want[role] == 403 {
			wantCommit = 403
		}
		if st, raw := e.upload(e.as(role), e.importPath(aid), csv, ""); st != wantCommit {
			t.Errorf("commit %s: %d %s", role, st, raw)
		}
	}
}
