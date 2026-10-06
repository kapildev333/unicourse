package course

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/auth"
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

func TestRosterImport(t *testing.T) {
	e := newEnv(t)
	path := "/sections/" + e.SectionA + "/enrollments/import"
	csv := "\ufeffEmail,DisplayName\r\nother@a.edu,Oth\r\nstud@a.edu,\r\nnew1@a.edu,\"Doe, J\"\r\nbad,\r\nx@zzz.com,\r\n OTHER@a.edu,\r\n\r\n"
	p := e.preview(e.Admin, path, csv)
	want := "2:add:|3:exists:|4:invite:|5:error:invalid email|6:error:domain not allowed|7:error:duplicate of line 2"
	if got := actions(p); got != want {
		t.Fatalf("got %s", got)
	}
	if p.Counts["add"] != 1 || p.Counts["invite"] != 1 || p.Counts["error"] != 3 || p.Counts["exists"] != 1 {
		t.Fatalf("counts %v", p.Counts)
	}
	if e.count(`select count(*) from enrollments`) != 1 || e.Mails.count() != 0 {
		t.Fatal("preview wrote")
	}
	// commit with error rows: line details, nothing written
	st, raw := e.upload(e.Admin, path, csv, p.Checksum)
	if code, f := errDetails(raw); st != 400 || code != "validation_failed" || strings.Join(f, ",") != "line:5,line:6,line:7" {
		t.Fatalf("commit with errors: %d %s", st, raw)
	}
	clean := "email\nother@a.edu\nnew1@a.edu\nnew2@a.edu\nstud@a.edu\n"
	p = e.preview(e.Admin, path, clean)
	if st, raw := e.upload(e.Admin, path, clean, ""); st != 400 {
		t.Fatalf("missing checksum: %d %s", st, raw)
	} else if _, f := errDetails(raw); len(f) != 1 || f[0] != "checksum" {
		t.Fatalf("missing checksum: %s", raw)
	}
	if st, _ := e.upload(e.Admin, path, clean+"new3@a.edu\n", p.Checksum); st != 400 {
		t.Fatal("changed file accepted")
	}
	if e.count(`select count(*) from enrollments`) != 1 || e.count(`select count(*) from invitations where email like 'new%'`) != 0 {
		t.Fatal("failed commits wrote")
	}
	st, raw = e.upload(e.Admin, path, clean, p.Checksum)
	if st != 200 || !strings.Contains(string(raw), `"rows":[`) {
		t.Fatalf("commit: %d %s", st, raw)
	}
	if e.count(`select count(*) from enrollments where section_id = $1`, e.SectionA) != 2 ||
		e.count(`select count(*) from pending_enrollments`) != 2 ||
		e.count(`select count(*) from audit_log where action = 'enrollment.create'`) != 1 ||
		e.count(`select count(*) from audit_log where action = 'invitation.create'`) != 2 || e.Mails.count() != 2 {
		t.Fatal("commit state wrong")
	}
	// second run: everything exists
	if p := e.preview(e.Admin, path, clean); p.Counts["exists"] != 4 {
		t.Fatalf("rerun %v", p.Counts)
	}
	uid, err := auth.SignIn(context.Background(), e.DB, "new1@a.edu", "New One")
	if err != nil || e.count(`select count(*) from enrollments where section_id = $1 and user_id = $2`, e.SectionA, uid) != 1 {
		t.Fatalf("signin enrollment: %v", err)
	}
}

func TestImportUploadErrors(t *testing.T) {
	e := newEnv(t)
	path := "/sections/" + e.SectionA + "/enrollments/import"
	p := e.preview(e.Admin, path, "email\n")
	if len(p.Rows) != 0 || p.Counts["add"] != 0 || p.Checksum == "" {
		t.Fatalf("header only %+v", p)
	}
	if st, raw := e.upload(e.Admin, path, "email\n", p.Checksum); st != 200 || !strings.Contains(string(raw), `"rows":[]`) {
		t.Fatalf("header-only commit %d %s", st, raw)
	}
	for name, body := range map[string]string{"missing column": "name\nx\n", "oversize": "email\n" + strings.Repeat("a", 6<<20)} {
		st, raw := e.upload(e.Admin, path+"/preview", body, "")
		if _, f := errDetails(raw); st != 400 || len(f) == 0 || f[0] != "file" {
			t.Fatalf("%s: %d %s", name, st, raw)
		}
	}
	if st, raw := e.call(e.Admin, "POST", path+"/preview", map[string]string{"a": "b"}); st != 400 {
		t.Fatalf("non-multipart %d %s", st, raw)
	}
}

func TestImportAuthorisation(t *testing.T) {
	e := newEnv(t)
	for _, c := range []*http.Cookie{e.Inst, e.TA, e.Stud, e.Other} {
		for _, p := range []string{"/courses/import", "/sections/import", "/sections/" + e.SectionA + "/enrollments/import"} {
			for _, suf := range []string{"", "/preview"} {
				if st, raw := e.upload(c, p+suf, "x\n", ""); st != 403 {
					t.Fatalf("%s%s: %d %s", p, suf, st, raw)
				}
			}
		}
	}
	rp := "/sections/" + e.SectionA + "/enrollments/import/preview"
	if st, _ := e.upload(e.DaOut, rp, "email\n", ""); st != 403 {
		t.Fatalf("da-out roster %d", st)
	}
	if st, raw := e.upload(e.DaIn, rp, "email\n", ""); st != 200 {
		t.Fatalf("da-in roster %d %s", st, raw)
	}
	var secB string
	e.tx(e.Fx.UniversityB, func(tx *sql.Tx) error {
		return tx.QueryRow(`insert into sections (university_id, course_id, term_id, number) values ($1,$2,$3,'001') returning id`, e.Fx.UniversityB, e.CourseB, e.TermB).Scan(&secB)
	})
	if st, _ := e.upload(e.Admin, "/sections/"+secB+"/enrollments/import/preview", "email\n", ""); st != 404 {
		t.Fatalf("section of B: %d", st)
	}
}

func TestCoursesImport(t *testing.T) {
	e := newEnv(t)
	csv := "departmentCode,code,title\nCS,CS101,Dup of existing\nCS,CS102,Data Structures\nNOPE,X1,x\nMATH,M1,Calc\nCS,,t\n"
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`update departments set code = 'CS' where id = $1`, e.Fx.DeptA)
		return err
	})
	p := e.preview(e.DaIn, "/courses/import", csv)
	want := "2:exists:|3:add:|4:error:unknown department code NOPE|5:error:department MATH is not managed by you|6:error:code is required"
	if got := actions(p); got != want {
		t.Fatalf("got %s", got)
	}
	ok := "departmentCode,code,title\nCS,CS102,Data Structures\nMATH,M1,Calc\n"
	if st, _ := e.upload(e.DaIn, "/courses/import", ok, e.preview(e.DaIn, "/courses/import", ok).Checksum); st != 400 {
		t.Fatalf("dept admin outside: %d", st)
	}
	p = e.preview(e.Admin, "/courses/import", ok)
	if st, raw := e.upload(e.Admin, "/courses/import", ok, p.Checksum); st != 200 {
		t.Fatalf("%d %s", st, raw)
	}
	if e.count(`select count(*) from courses`) != 3 || e.count(`select count(*) from audit_log where action = 'course.create'`) != 2 {
		t.Fatal("courses not created")
	}
}

func TestSectionsImport(t *testing.T) {
	e := newEnv(t)
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`update terms set name = 'Fall 2026' where id = $1`, e.Fx.TermA)
		return err
	})
	csv := "courseCode,termName,number,instructorEmails,taEmails\n" +
		"CS101,Fall 2026,001,newbie@a.edu,\n" + // existing section, staff to invite
		"CS101,Fall 2026,002,other@a.edu; inst@a.edu,ta@a.edu\n" +
		"CS101,Fall 2026,003,,\n" +
		"CS101,Spring 2027,003,,\n" +
		"CS999,Fall 2026,004,,\n" +
		"CS101,Fall 2026,005,other@a.edu,other@a.edu\n"
	p := e.preview(e.Admin, "/sections/import", csv)
	want := "2:invite:|3:add:|4:add:|5:add:|6:error:unknown course code CS999|7:error:other@a.edu listed as both instructor and ta"
	if got := actions(p); got != want {
		t.Fatalf("got %s", got)
	}
	ok := strings.Join(strings.Split(csv, "\n")[:4], "\n") + "\n"
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`update terms set name = 'Spring 2027' where id = $1`, e.TermA2)
		return err
	})
	p = e.preview(e.Admin, "/sections/import", ok)
	if st, raw := e.upload(e.Admin, "/sections/import", ok, p.Checksum); st != 200 {
		t.Fatalf("%d %s", st, raw)
	}
	if e.count(`select count(*) from sections`) != 3 ||
		e.count(`select count(*) from section_staff`) != 5 || // inst, ta + section 002: other, inst, ta
		e.count(`select count(*) from pending_section_staff`) != 1 ||
		e.count(`select count(*) from audit_log where action = 'section.create'`) != 2 || e.Mails.count() != 1 {
		t.Fatalf("sections state: staff=%d", e.count(`select count(*) from section_staff`))
	}
	if p := e.preview(e.Admin, "/sections/import", ok); p.Counts["exists"] != 3 {
		t.Fatalf("rerun %s", actions(p))
	}
}

func TestRosterImportLarge(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	e := newEnv(t)
	var b strings.Builder
	b.WriteString("email\n")
	for i := 0; i < 5000; i++ {
		fmt.Fprintf(&b, "s%d@a.edu\n", i)
	}
	path := "/sections/" + e.SectionA + "/enrollments/import"
	start := time.Now()
	p := e.preview(e.Admin, path, b.String())
	if st, raw := e.upload(e.Admin, path, b.String(), p.Checksum); st != 200 {
		t.Fatalf("%d %.200s", st, raw)
	}
	if d := time.Since(start); d > 15*time.Second {
		t.Fatalf("took %v", d)
	}
	if e.count(`select count(*) from pending_enrollments`) != 5000 || e.Mails.count() != 5000 {
		t.Fatal("rows missing")
	}
}
