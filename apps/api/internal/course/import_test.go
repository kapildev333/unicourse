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
	want := "2:invite:section exists; adds staff|3:add:|4:add:|5:add:|6:error:unknown course code CS999|7:error:other@a.edu listed as both instructor and ta"
	if got := actions(p); got != want {
		t.Fatalf("got %s", got)
	}
	ok := strings.Join(strings.Split(csv, "\n")[:4], "\n") + "\n"
	// a dept_admin of another department cannot import sections of CS101
	pd := e.preview(e.DaOut, "/sections/import", ok)
	if got := actions(pd); !strings.Contains(got, "3:error:course CS101 is not in a department you manage") {
		t.Fatalf("daout preview %s", got)
	}
	if st, _ := e.upload(e.DaOut, "/sections/import", ok, pd.Checksum); st != 400 || e.count(`select count(*) from sections`) != 1 ||
		e.count(`select count(*) from pending_section_staff`) != 0 {
		t.Fatalf("daout commit wrote or accepted: %d", st)
	}
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
	// one staff.create audit row per section_staff row created (3; the fixture's 2 have none)
	if n := e.count(`select count(*) from audit_log where action = 'staff.create'`); n != 3 {
		t.Fatalf("staff.create rows = %d", n)
	}
	if n := e.count(`select count(*) from audit_log a join section_staff ss on ss.id = a.entity_id join users u on u.id = ss.user_id
		where a.action = 'staff.create' and a.entity_type = 'staff'
		and a.after = jsonb_build_object('sectionId', ss.section_id, 'userId', ss.user_id, 'email', u.email, 'role', ss.role)`); n != 3 {
		t.Fatalf("staff audit rows matching section_staff = %d", n)
	}
}

// An existing section is never changed, but listed staff missing from it are added (D31);
// listed people already on it keep their role and nobody is removed.
func TestSectionsImportExistingSection(t *testing.T) {
	e := newEnv(t)
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`update terms set name = 'Fall 2026' where id = $1`, e.Fx.TermA)
		return err
	})
	hdr := "courseCode,termName,number,instructorEmails,taEmails\n"
	add, inv, same := hdr+"CS101,Fall 2026,001,,other@a.edu\n", hdr+"CS101,Fall 2026,001,newbie@a.edu,\n", hdr+"CS101,Fall 2026,001,ta@a.edu,inst@a.edu\n"
	for csv, want := range map[string]string{add: "2:add:section exists; adds staff", inv: "2:invite:section exists; adds staff", same: "2:exists:"} {
		if got := actions(e.preview(e.Admin, "/sections/import", csv)); got != want {
			t.Fatalf("got %s want %s", got, want)
		}
	}
	for _, csv := range []string{add, inv, same} {
		if st, raw := e.upload(e.Admin, "/sections/import", csv, e.preview(e.Admin, "/sections/import", csv).Checksum); st != 200 {
			t.Fatalf("%d %s", st, raw)
		}
	}
	if e.count(`select count(*) from sections`) != 1 || e.count(`select count(*) from section_staff`) != 3 ||
		e.count(`select count(*) from pending_section_staff`) != 1 {
		t.Fatal("additive commit state wrong")
	}
	for _, csv := range []string{add, inv, same} {
		if got := actions(e.preview(e.Admin, "/sections/import", csv)); got != "2:exists:" {
			t.Fatalf("rerun %s", got)
		}
	}
	// never re-roled, never removed
	if e.count(`select count(*) from section_staff ss join users u on u.id = ss.user_id
		where ss.section_id = $1 and ((u.email = 'inst@a.edu' and ss.role = 'instructor') or (u.email = 'ta@a.edu' and ss.role = 'ta'))`, e.SectionA) != 2 {
		t.Fatal("staff re-roled or removed")
	}
}

func TestSectionsImportMeetings(t *testing.T) {
	e := newEnv(t)
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`update terms set name = 'Fall 2026' where id = $1`, e.Fx.TermA)
		return err
	})
	csv := "courseCode,termName,number,meetings\n" +
		"CS101,Fall 2026,002,\"Wed 10:00-11:15; Mon 09:00-10:00 @ Room 101\"\n" +
		"CS101,Fall 2026,003,Mon 11:00-10:00\n"
	p := e.preview(e.Admin, "/sections/import", csv)
	if got, want := actions(p), `2:add:|3:error:meeting "Mon 11:00-10:00": endsAt must be after startsAt`; got != want {
		t.Fatalf("got %s", got)
	}
	ok := strings.Join(strings.Split(csv, "\n")[:2], "\n") + "\n"
	p = e.preview(e.Admin, "/sections/import", ok)
	if st, raw := e.upload(e.Admin, "/sections/import", ok, p.Checksum); st != 200 {
		t.Fatalf("%d %s", st, raw)
	}
	var id string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select id from sections where number = '002'`).Scan(&id)
	})
	st, raw := e.call(e.Admin, "GET", "/sections/"+id, nil)
	var s Section
	if json.Unmarshal(raw, &s); st != 200 || len(s.Meetings) != 2 || s.Meetings[0].Weekday != 1 || s.Meetings[0].Location != "Room 101" || s.Meetings[1].StartsAt != "10:00" {
		t.Fatalf("meetings %d %s", st, raw)
	}
	if n, _, after := e.audit("section.create", id); n != 1 || !strings.Contains(after.String, `"Room 101"`) {
		t.Fatalf("audit %d %s", n, after.String)
	}
	// a file without the column behaves as before
	if got := actions(e.preview(e.Admin, "/sections/import", "courseCode,termName,number\nCS101,Fall 2026,009\n")); got != "2:add:" {
		t.Fatalf("no column: %s", got)
	}
}

func TestSectionsImportMeetingsAdditive(t *testing.T) {
	e := newEnv(t)
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`update terms set name = 'Fall 2026' where id = $1`, e.Fx.TermA)
		return err
	})
	hdr := "courseCode,termName,number,instructorEmails,meetings\n"
	slots := hdr + "CS101,Fall 2026,001,,Tue 10:00-11:00\n"
	if got, want := actions(e.preview(e.Admin, "/sections/import", slots)), "2:add:section exists; adds meetings"; got != want {
		t.Fatalf("got %s", got)
	}
	if st, raw := e.upload(e.Admin, "/sections/import", slots, e.preview(e.Admin, "/sections/import", slots).Checksum); st != 200 {
		t.Fatalf("%d %s", st, raw)
	}
	if e.count(`select count(*) from section_meetings where section_id = $1`, e.SectionA) != 1 ||
		e.count(`select count(*) from audit_log where action = 'section.update' and entity_id = $1`, e.SectionA) != 1 {
		t.Fatal("meetings or audit not written")
	}
	// hand edit, then a different slot: kept
	if st, raw := e.call(e.Admin, "PATCH", "/sections/"+e.SectionA, map[string]any{"meetings": []map[string]any{{"weekday": 5, "startsAt": "08:00", "endsAt": "09:00"}}}); st != 200 {
		t.Fatalf("%d %s", st, raw)
	}
	other := hdr + "CS101,Fall 2026,001,,Mon 10:00-11:00\n"
	if got, want := actions(e.preview(e.Admin, "/sections/import", other)), "2:exists:section exists; keeps meetings"; got != want {
		t.Fatalf("got %s", got)
	}
	if st, raw := e.upload(e.Admin, "/sections/import", other, e.preview(e.Admin, "/sections/import", other).Checksum); st != 200 {
		t.Fatalf("%d %s", st, raw)
	}
	if e.count(`select count(*) from section_meetings where section_id = $1 and weekday = 5 and starts_at = '08:00'`, e.SectionA) != 1 ||
		e.count(`select count(*) from section_meetings where section_id = $1`, e.SectionA) != 1 {
		t.Fatal("hand-set meetings changed")
	}
	both := hdr + "CS101,Fall 2026,001,other@a.edu,Mon 10:00-11:00\n"
	if got, want := actions(e.preview(e.Admin, "/sections/import", both)), "2:add:section exists; adds staff; keeps meetings"; got != want {
		t.Fatalf("got %s", got)
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
