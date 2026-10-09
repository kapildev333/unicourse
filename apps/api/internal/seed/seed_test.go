package seed

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/blob"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/testdb"
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
			f["overrides"] = append(f["overrides"].([]any), map[string]any{"student": "david.mensah@<domain>", "reason": "x"})
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
		{"quiz window reversed", func(m map[string]any) { sec(m, 0, 0)["quizzes"].([]any)[0].(map[string]any)["closesIn"] = "-2h" }, "closesIn"},
		{"attempt by a non-student", func(m map[string]any) {
			sec(m, 0, 0)["quizzes"].([]any)[0].(map[string]any)["attempts"].([]any)[0].(map[string]any)["email"] = "ingrid.larsen@<domain>"
		}, "invited"},
		{"answer to no question", func(m map[string]any) {
			sec(m, 0, 0)["quizzes"].([]any)[0].(map[string]any)["attempts"].([]any)[0].(map[string]any)["answers"] = map[string]any{"9": map[string]any{"text": "x"}}
		}, "question index"},
		{"submission file missing", func(m map[string]any) {
			sec(m, 0, 0)["assignments"].([]any)[0].(map[string]any)["submissions"].([]any)[0].(map[string]any)["files"] = []any{"nope.pdf"}
		}, "nope.pdf"},
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

// count runs a one-number query in the university's tenant scope (forced row level security).
func count(t *testing.T, db *sql.DB, uni, q string, args ...any) (n int) {
	t.Helper()
	err := pg.WithTx(context.Background(), db, uni, func(tx *sql.Tx) error { return tx.QueryRow(q, args...).Scan(&n) })
	if err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	return n
}

func platformCount(t *testing.T, db *sql.DB, q string, args ...any) (n int) {
	t.Helper()
	err := pg.WithPlatformTx(context.Background(), db, func(tx *sql.Tx) error { return tx.QueryRow(q, args...).Scan(&n) })
	if err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	return n
}

func seedOpts(t *testing.T) Options {
	t.Helper()
	d, err := blob.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return Options{BaseURL: "http://localhost:8080", Files: d}
}

func TestRunSeedsDemo(t *testing.T) {
	_, db := testdb.OpenWithOwner(t)
	f, err := Load(Demo, "", 2031)
	if err != nil {
		t.Fatal(err)
	}
	o := seedOpts(t)
	rep, err := Run(context.Background(), db, f, o)
	if err != nil {
		t.Fatal(err)
	}
	u := rep.UniversityID
	checks := []struct {
		what string
		q    string
		want int
		plat bool
	}{
		{"university", `select count(*) from universities where slug = 'northfield'`, 1, true},
		{"departments", `select count(*) from departments`, 4, false},
		{"terms", `select count(*) from terms`, 2, false},
		{"courses", `select count(*) from courses`, 5, false},
		{"sections", `select count(*) from sections`, 6, false},
		{"meetings", `select count(*) from section_meetings`, 10, false},
		{"users", `select count(*) from users where email like '%@northfield.edu'`, 13, true},
		{"memberships", `select count(*) from memberships where status = 'active'`, 13, false},
		{"memberships total", `select count(*) from memberships`, 13, false},
		{"department_admins", `select count(*) from department_admins`, 1, false},
		{"invitation", `select count(*) from invitations where accepted_at is null and email = 'ingrid.larsen@northfield.edu' and display_name = 'Ingrid Larsen'`, 1, false},
		{"invitations", `select count(*) from invitations`, 1, false},
		{"pending staff", `select count(*) from pending_section_staff`, 1, false},
		{"pending enrollments", `select count(*) from pending_enrollments`, 0, false},
		{"instructors", `select count(*) from section_staff where role = 'instructor'`, 5, false},
		{"tas", `select count(*) from section_staff where role = 'ta'`, 1, false},
		{"enrollments", `select count(*) from enrollments`, 26, false},
		{"grade_scales", `select count(*) from grade_scales`, 12, false},
		{"assessments", `select count(*) from assessments where published`, 10, false},
		{"assessments total", `select count(*) from assessments`, 12, false},
		{"scores", `select count(*) from scores`, 50, false},
		{"quizzes", `select count(*) from assessments where kind = 'quiz' and name = 'Week 3 check' and not published and opens_at < now() and closes_at > now()`, 1, false},
		{"quiz questions", `select count(*) from questions`, 3, false},
		{"quiz attempts", `select count(*) from attempts where submitted_at is not null and not auto_submitted`, 2, false},
		{"quiz reviewed", `select count(*) from attempts a join users u on u.id = a.reviewed_by where u.display_name = 'Sofia Alvarez' and a.feedback <> '' and a.score = 2.5`, 1, false},
		{"assignments", `select count(*) from assessments where kind = 'assignment' and name = 'Essay 1' and max_points = 20 and not published and closes_at > now()`, 1, false},
		{"submissions", `select count(*) from submissions`, 2, false},
		{"submission files", `select count(*) from submission_files`, 1, false},
		{"graded submission", `select count(*) from submissions s join scores c on c.assessment_id = s.assessment_id and c.user_id = s.user_id where s.graded_at is not null and s.score = 17 and s.feedback <> '' and c.points = 17`, 1, false},
		{"final_grades", `select count(*) from final_grades`, 8, false},
		{"incomplete", `select count(*) from final_grades f join users u on u.id = f.user_id where f.status = 'incomplete' and f.override_reason <> '' and u.email = 'lucas.moreau@northfield.edu'`, 1, false},
		{"cs210 final grades", `select count(*) from final_grades f join sections s on s.id = f.section_id join courses c on c.id = s.course_id where c.code = 'CS210'`, 0, false},
		{"descriptions", `select count(*) from sections where description <> ''`, 6, false},
		{"announcements", `select count(*) from announcements`, 8, false},
		{"emailed", `select count(*) from announcements where emailed`, 0, false},
		{"weeks", `select count(*) from weeks`, 8, false},
		{"materials", `select count(*) from materials`, 3, false},
		{"outbox", `select count(*) from outbox_mails`, 0, true},
		{"sessions", `select count(*) from sessions s join users u on u.id = s.user_id where u.email like '%@northfield.edu'`, 0, true},
		{"audit enrollment", `select count(*) from audit_log where action = 'enrollment.create'`, 26, false},
		{"audit staff", `select count(*) from audit_log where action = 'staff.create'`, 6, false},
		{"audit finalize", `select count(*) from audit_log where action = 'final_grade.finalize'`, 1, false},
		{"audit invitation", `select count(*) from audit_log where action = 'invitation.create'`, 1, false},
	}
	for _, c := range checks {
		var got int
		if c.plat {
			got = platformCount(t, db, c.q)
		} else {
			got = count(t, db, u, c.q)
		}
		if got != c.want {
			t.Errorf("%s: got %d, want %d", c.what, got, c.want)
		}
	}
	if n := count(t, db, u, `select count(*) from announcements a join users x on x.id = a.author_user_id where x.display_name = 'Sofia Alvarez'`); n != 1 {
		t.Errorf("announcements by Sofia: %d", n)
	}
	for _, p := range f.People {
		if p.Invited {
			continue
		}
		if n := platformCount(t, db, `select count(*) from users where email = $1 and display_name = $2`, p.Email, p.DisplayName); n != 1 {
			t.Errorf("user %s: %d", p.Email, n)
		}
	}
	err = pg.WithTx(context.Background(), db, u, func(tx *sql.Tx) error {
		rows, err := tx.Query(`select id, file_name from materials`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id, name string
			if err := rows.Scan(&id, &name); err != nil {
				return err
			}
			want, err := files.ReadFile("files/" + name)
			if err != nil {
				return err
			}
			got, err := os.ReadFile(filepath.Join(o.Files.Root, u, id))
			if err != nil || !bytes.Equal(got, want) {
				t.Errorf("blob of %s: %v", name, err)
			}
		}
		return rows.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Personas) != 14 {
		t.Fatalf("personas: %d", len(rep.Personas))
	}
	by := map[string]Persona{}
	for _, p := range rep.Personas {
		by[p.Email] = p
	}
	if got := by["elena.marsh@northfield.edu"].Sees; got != "instructor of CS101, CS210, MATH201" {
		t.Errorf("elena: %q", got)
	}
	if got := by["ingrid.larsen@northfield.edu"]; got.Role != "invited" || got.Sees != "invitation pending, never signed in; TA on CS210 at first sign-in" {
		t.Errorf("ingrid: %+v", got)
	}
	if got := by["amara.okafor@northfield.edu"].Sees; got != "student in CS101, CS210, MATH201" {
		t.Errorf("amara: %q", got)
	}
}

func TestRunTwiceAndReset(t *testing.T) {
	_, db := testdb.OpenWithOwner(t)
	f, err := Load(Demo, "", 2031)
	if err != nil {
		t.Fatal(err)
	}
	o := seedOpts(t)
	first, err := Run(context.Background(), db, f, o)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Run(context.Background(), db, f, o); !errors.Is(err, ErrExists) {
		t.Fatalf("second run: %v", err)
	}
	if n := platformCount(t, db, `select count(*) from universities`); n != 1 {
		t.Fatalf("universities %d", n)
	}
	if n := count(t, db, first.UniversityID, `select count(*) from assessments`); n != 12 {
		t.Fatalf("assessments %d", n)
	}
	o.Reset = true
	again, err := Run(context.Background(), db, f, o)
	if err != nil {
		t.Fatal(err)
	}
	if again.UniversityID == first.UniversityID {
		t.Fatal("reset kept the old id")
	}
	u := again.UniversityID
	for q, want := range map[string]int{`select count(*) from sections`: 6, `select count(*) from scores`: 50, `select count(*) from materials`: 3} {
		if n := count(t, db, u, q); n != want {
			t.Errorf("%s: %d, want %d", q, n, want)
		}
	}
	if n := platformCount(t, db, `select count(*) from users where email like '%@northfield.edu'`); n != 13 {
		t.Errorf("users %d", n)
	}
	if err := o.Files.Walk(func(uni, id string, _ time.Time) error {
		if uni == first.UniversityID {
			t.Errorf("old blob %s/%s left", uni, id)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestResetLeavesOtherUniversityAlone(t *testing.T) {
	_, db := testdb.OpenWithOwner(t)
	o := seedOpts(t)
	run := func(slug string) Report {
		f, err := Load(Demo, slug, 2031)
		if err != nil {
			t.Fatal(err)
		}
		rep, err := Run(context.Background(), db, f, o)
		if err != nil {
			t.Fatal(err)
		}
		return rep
	}
	keep, gone := run("northfield"), run("acme")
	blobs := func(uni string) (n int) {
		if err := o.Files.Walk(func(u, _ string, _ time.Time) error {
			if u == uni {
				n++
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if blobs(keep.UniversityID) != 4 || blobs(gone.UniversityID) != 4 {
		t.Fatal("want 4 blobs per university before the reset")
	}
	f, err := Load(Demo, "acme", 2031)
	if err != nil {
		t.Fatal(err)
	}
	o.Reset = true
	if _, err := Run(context.Background(), db, f, o); err != nil {
		t.Fatal(err)
	}
	for q, want := range map[string]int{`select count(*) from sections`: 6, `select count(*) from scores`: 50, `select count(*) from materials`: 3, `select count(*) from enrollments`: 26} {
		if n := count(t, db, keep.UniversityID, q); n != want {
			t.Errorf("northfield %s: %d, want %d", q, n, want)
		}
	}
	if n := platformCount(t, db, `select count(*) from universities where slug = 'northfield'`); n != 1 {
		t.Errorf("northfield rows: %d", n)
	}
	if blobs(keep.UniversityID) != 4 {
		t.Errorf("northfield blobs: %d", blobs(keep.UniversityID))
	}
	if blobs(gone.UniversityID) != 0 {
		t.Errorf("old acme blobs left: %d", blobs(gone.UniversityID))
	}
	if _, err := os.Stat(filepath.Join(o.Files.Root, gone.UniversityID)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("old acme files directory left: %v", err)
	}
	if _, err := os.Stat(filepath.Join(o.Files.Root, keep.UniversityID)); err != nil {
		t.Errorf("northfield files directory: %v", err)
	}
}

func TestRunFailureHint(t *testing.T) {
	_, db := testdb.OpenWithOwner(t)
	f, err := Load(Demo, "", 2031)
	if err != nil {
		t.Fatal(err)
	}
	f.Courses[0].Sections[0].Assessments[0].Weight = "1000"
	_, err = Run(context.Background(), db, f, seedOpts(t))
	if err == nil {
		t.Fatal("want an error")
	}
	for _, w := range []string{`assessment "Midterm Exam" of CS101 001 as elena.marsh@northfield.edu`, "HTTP 400", "weight", "rerun with --reset"} {
		if !strings.Contains(err.Error(), w) {
			t.Errorf("error lacks %q: %v", w, err)
		}
	}
	if n := platformCount(t, db, `select count(*) from sessions s join users u on u.id = s.user_id where u.email like '%@northfield.edu'`); n != 0 {
		t.Errorf("sessions left: %d", n)
	}
}
