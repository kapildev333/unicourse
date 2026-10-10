package calendar

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

type item struct {
	AssessmentID string `json:"assessmentId"`
	Kind, Name   string
	Section      struct{ ID, Number string }
	Course       struct{ Code, Title string }
	MyRole       string
	OpensAt      time.Time
	DueAt        time.Time
	LateUntil    *time.Time
	TimeLimit    *int `json:"timeLimitMinutes"`
	Started      bool
	Submitted    *int `json:"submittedCount"`
	Enrolled     *int `json:"enrolledCount"`
}

func (e *env) cookieOf(email string) *http.Cookie { return e.cookie(e.UserID[email]) }

func (e *env) upcoming(email string) []item {
	e.T.Helper()
	st, raw := e.call(e.cookieOf(email), "GET", "/me/upcoming", nil)
	var out struct{ Items []item }
	if st != 200 || json.Unmarshal(raw, &out) != nil || strings.Contains(string(raw), "null") {
		e.T.Fatalf("upcoming: %d %s", st, raw)
	}
	return out.Items
}

func names(items []item) string {
	var out []string
	for _, i := range items {
		out = append(out, i.Name)
	}
	return strings.Join(out, ",")
}

func TestUpcomingStudent(t *testing.T) {
	e := newEnv(t)
	e.work(e.S1, "quiz", "open quiz", "-1 day", "2 days", "")
	timed := e.work(e.S1, "quiz", "timed quiz", "-1 day", "2 days", "30")
	e.attempt("stud@a.edu", timed, "10 minutes", false)
	done := e.work(e.S1, "quiz", "done quiz", "-1 day", "2 days", "")
	e.attempt("stud@a.edu", done, "1 hour", true)
	e.work(e.S1, "assignment", "late window", "-3 days", "-1 hour", "1 day")
	e.work(e.S1, "assignment", "closed", "-3 days", "-1 day", "")
	turnedIn := e.work(e.S1, "assignment", "turned in", "-1 day", "1 day", "")
	e.submit("stud@a.edu", turnedIn)
	e.work(e.S1, "quiz", "opens soon", "3 days", "5 days", "")
	e.work(e.S1, "quiz", "opens far", "20 days", "21 days", "")
	e.work(e.S1, "quiz", "long open", "-1 day", "30 days", "")
	e.work(e.S2, "quiz", "other section", "-1 day", "2 days", "")
	e.manual(e.S1, "manual")

	items := e.upcoming("stud@a.edu")
	if got := names(items); got != "late window,timed quiz,open quiz,opens soon,long open" {
		t.Fatalf("items %s", got)
	}
	late, timedItem, open := items[0], items[1], items[2]
	if late.Kind != "assignment" || late.LateUntil == nil || late.MyRole != "student" || late.Started || late.Submitted != nil || late.Enrolled != nil {
		t.Fatalf("late: %+v", late)
	}
	if late.Course.Code != "CS101" || late.Course.Title != "Intro to CS" || late.Section.ID != e.S1 || late.Section.Number != "001" {
		t.Fatalf("refs: %+v", late)
	}
	if !timedItem.Started || timedItem.TimeLimit == nil || *timedItem.TimeLimit != 30 || time.Until(timedItem.DueAt) > 25*time.Minute || time.Until(timedItem.DueAt) < 15*time.Minute {
		t.Fatalf("timed: %+v", timedItem)
	}
	if open.Started || open.TimeLimit != nil || open.LateUntil != nil || time.Until(open.DueAt) < 47*time.Hour {
		t.Fatalf("open: %+v", open)
	}
	if got := names(e.upcoming("stud2@a.edu")); got != "late window,open quiz,timed quiz,done quiz,opens soon,long open" {
		t.Fatalf("stud2 (nothing started or submitted) %s", got)
	}
}

func TestUpcomingStaff(t *testing.T) {
	e := newEnv(t)
	q := e.work(e.S1, "quiz", "Quiz 1", "-1 day", "2 days", "30")
	a := e.work(e.S1, "assignment", "Essay", "-1 day", "3 days", "")
	e.work(e.S2, "quiz", "other section", "-1 day", "2 days", "")
	e.attempt("stud@a.edu", q, "5 minutes", true)
	e.attempt("stud2@a.edu", q, "5 minutes", false) // running, not submitted
	e.submit("stud@a.edu", a)
	e.submit("stud2@a.edu", a)
	e.submit("inst@a.edu", a) // staff rows never count
	for email, role := range map[string]string{"inst@a.edu": "instructor", "ta@a.edu": "ta"} {
		items := e.upcoming(email)
		if names(items) != "Quiz 1,Essay" {
			t.Fatalf("%s: %s", email, names(items))
		}
		if i := items[0]; i.MyRole != role || i.Started || i.Enrolled == nil || *i.Enrolled != 2 || i.Submitted == nil || *i.Submitted != 1 {
			t.Fatalf("%s quiz: %+v", email, i)
		}
		if i := items[1]; i.Submitted == nil || *i.Submitted != 2 || *i.Enrolled != 2 {
			t.Fatalf("%s essay: %+v", email, i)
		}
	}
	// a staff quiz's dueAt is closes_at, not shortened by a time limit
	if i := e.upcoming("inst@a.edu")[0]; time.Until(i.DueAt) < 47*time.Hour {
		t.Fatalf("staff due %v", i.DueAt)
	}
}

func TestUpcomingEmpty(t *testing.T) {
	e := newEnv(t)
	e.work(e.S1, "quiz", "Quiz 1", "-1 day", "2 days", "")
	for _, email := range []string{"admin@a.edu", "lone@a.edu"} { // an admin who is not staff, a member of no section
		st, raw := e.call(e.cookieOf(email), "GET", "/me/upcoming", nil)
		if st != 200 || !strings.Contains(string(raw), `"items":[]`) {
			t.Fatalf("%s: %d %s", email, st, raw)
		}
	}
	if st, _ := e.call(nil, "GET", "/me/upcoming", nil); st != 401 {
		t.Fatalf("anonymous: %d", st)
	}
}
