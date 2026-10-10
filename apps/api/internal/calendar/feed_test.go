package calendar

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/pg"
)

// get fetches an absolute-path feed URL with no cookie.
func (e *env) get(path string) (*http.Response, string) {
	e.T.Helper()
	res, err := http.Get(e.Srv.URL + path)
	if err != nil {
		e.T.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res, string(b)
}

// section002 adds a term starting (campus) startDays from today for days+1 days and a section CS101-002 of it that
// stud@a.edu is in, with a 09:00-10:30 meeting on every weekday at Room 7.
func (e *env) section002(startDays, endDays int) {
	e.T.Helper()
	e.tx(func(tx *sql.Tx) error {
		var term, course, sec string
		if err := tx.QueryRow(`insert into terms (university_id, name, starts_on, ends_on)
			values ($1,'Feed term', (now() at time zone 'America/New_York')::date + $2::int, (now() at time zone 'America/New_York')::date + $3::int) returning id`,
			e.Fx.UniversityA, startDays, endDays).Scan(&term); err != nil {
			return err
		}
		if err := tx.QueryRow(`select course_id from sections where id = $1`, e.S1).Scan(&course); err != nil {
			return err
		}
		if err := tx.QueryRow(`insert into sections (university_id, course_id, term_id, number) values ($1,$2,$3,'002') returning id`, e.Fx.UniversityA, course, term).Scan(&sec); err != nil {
			return err
		}
		if _, err := tx.Exec(`insert into enrollments (university_id, section_id, user_id) values ($1,$2,$3)`, e.Fx.UniversityA, sec, e.UserID["stud@a.edu"]); err != nil {
			return err
		}
		for wd := 1; wd <= 7; wd++ {
			if _, err := tx.Exec(`insert into section_meetings (university_id, section_id, weekday, starts_at, ends_at, location) values ($1,$2,$3,'09:00','10:30','Room 7')`, e.Fx.UniversityA, sec, wd); err != nil {
				return err
			}
		}
		return nil
	})
}

func TestFeedServes(t *testing.T) {
	e := newEnv(t)
	longTitle := "Intro, Part 1; Basics " + strings.Repeat("of everything ", 8)
	e.tx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`update courses set title = $1 where code = 'CS101'`, longTitle)
		return err
	})
	e.section002(2, 12) // 11 days
	quiz := e.work(e.S1, "quiz", "Quiz 1", "-1 day", "2 days", "")
	e.work(e.S1, "assignment", "Essay", "-1 day", "3 days", "")
	turnedIn := e.work(e.S1, "assignment", "turned in", "-1 day", "3 days", "")
	e.submit("stud@a.edu", turnedIn)
	e.work(e.S2, "quiz", "not mine", "-1 day", "2 days", "")
	path, _ := e.mint("stud@a.edu")

	res, body := e.get(path)
	if res.StatusCode != 200 || res.Header.Get("Content-Type") != "text/calendar; charset=utf-8" || res.Header.Get("Cache-Control") != "no-store" ||
		res.Header.Get("Referrer-Policy") != "no-referrer" || !strings.Contains(res.Header.Get("Content-Disposition"), ".ics") {
		t.Fatalf("%d %v", res.StatusCode, res.Header)
	}
	if strings.Contains(strings.ReplaceAll(body, "\r\n", ""), "\n") || !strings.HasPrefix(body, "BEGIN:VCALENDAR\r\n") || !strings.HasSuffix(body, "END:VCALENDAR\r\n") {
		t.Fatalf("framing: %q", body)
	}
	for _, l := range strings.Split(strings.TrimSuffix(body, "\r\n"), "\r\n") {
		if len(l) > 75 {
			t.Fatalf("physical line of %d octets: %q", len(l), l)
		}
	}
	un := strings.ReplaceAll(body, "\r\n ", "")

	// meetings: one VEVENT per day of the term (bounds), at the UTC instant of 09:00 in New York (DST-correct)
	if n := strings.Count(un, "UID:meeting-"); n != 11 {
		t.Fatalf("%d meeting events, want 11", n)
	}
	ny, _ := time.LoadLocation("America/New_York")
	d := time.Now().In(ny).AddDate(0, 0, 5)
	start := time.Date(d.Year(), d.Month(), d.Day(), 9, 0, 0, 0, ny).UTC().Format("20060102T150405Z")
	end := time.Date(d.Year(), d.Month(), d.Day(), 10, 30, 0, 0, ny).UTC().Format("20060102T150405Z")
	if !strings.Contains(un, "DTSTART:"+start+"\r\n") || !strings.Contains(un, "DTEND:"+end+"\r\n") || !strings.Contains(un, "-"+d.Format("20060102")+"@unicourse") {
		t.Fatalf("no occurrence at %s in %s", start, un)
	}
	// summary escaped, location present
	if !strings.Contains(un, `SUMMARY:CS101-002 Intro\, Part 1\; Basics of everything`) || !strings.Contains(un, "LOCATION:Room 7\r\n") {
		t.Fatalf("summary/location: %s", un)
	}
	// deadlines
	if !strings.Contains(un, "SUMMARY:CS101-001 Quiz closes: Quiz 1\r\n") || !strings.Contains(un, "SUMMARY:CS101-001 Assignment due: Essay\r\n") ||
		!strings.Contains(un, "DESCRIPTION:http://app.test/a/sections/"+e.S1+"/quizzes/"+quiz+"\r\n") || !strings.Contains(un, "UID:assessment-"+quiz+"@unicourse\r\n") {
		t.Fatalf("deadlines: %s", un)
	}
	if strings.Contains(un, "turned in") || strings.Contains(un, "not mine") {
		t.Fatalf("submitted or foreign work leaked: %s", un)
	}
	if !strings.Contains(un, "X-WR-CALNAME:UniCourse: ") || !regexp.MustCompile(`DTSTAMP:\d{8}T\d{6}Z`).MatchString(un) {
		t.Fatalf("header: %s", un)
	}
}

func TestFeed404(t *testing.T) {
	e := newEnv(t)
	path, tok := e.mint("stud@a.edu")
	if res, _ := e.get(path); res.StatusCode != 200 {
		t.Fatalf("baseline %d", res.StatusCode)
	}
	other := strings.Repeat("A", 43)
	ok := func(name, p string) {
		t.Helper()
		res, body := e.get(p)
		if res.StatusCode != 404 || strings.Contains(res.Header.Get("Content-Type"), "calendar") || strings.Contains(body, "BEGIN:VCALENDAR") {
			t.Errorf("%s: %d %s", name, res.StatusCode, body)
		}
	}
	root := "/api/v1/universities/"
	ok("unknown token", root+"a/calendar-feeds/"+other+".ics")
	ok("malformed", root+"a/calendar-feeds/short.ics")
	ok("no .ics", root+"a/calendar-feeds/"+tok)
	ok("wrong slug", root+"b/calendar-feeds/"+tok+".ics")
	ok("unknown slug", root+"nope/calendar-feeds/"+tok+".ics")

	_, tok2 := e.mint("stud@a.edu") // rotate
	ok("rotated", root+"a/calendar-feeds/"+tok+".ics")
	if res, _ := e.get(root + "a/calendar-feeds/" + tok2 + ".ics"); res.StatusCode != 200 {
		t.Fatalf("new token %d", res.StatusCode)
	}
	e.tx(func(tx *sql.Tx) error { // membership no longer active
		_, err := tx.Exec(`update memberships set status = 'invited' where user_id = $1`, e.UserID["stud@a.edu"])
		return err
	})
	ok("inactive membership", root+"a/calendar-feeds/"+tok2+".ics")
	e.tx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`update memberships set status = 'active' where user_id = $1`, e.UserID["stud@a.edu"])
		return err
	})
	if st, _ := e.call(e.Stud, "DELETE", "/me/calendar-feed", nil); st != 204 {
		t.Fatalf("delete %d", st)
	}
	ok("revoked", root+"a/calendar-feeds/"+tok2+".ics")

	path3, tok3 := e.mint("stud@a.edu")
	e.tx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`delete from memberships where user_id = $1`, e.UserID["stud@a.edu"])
		return err
	})
	ok("membership deleted", path3)
	_ = tok3
}

func TestFeedTokenNotInRequestLog(t *testing.T) {
	e := newEnv(t)
	path, tok := e.mint("stud@a.edu")
	e.Logs.Reset()
	e.get(path)
	e.get("/api/v1/universities/a/calendar-feeds/" + strings.Repeat("B", 43) + ".ics")
	if strings.Contains(e.Logs.String(), tok) || !strings.Contains(e.Logs.String(), `"path":"/api/v1/universities/{slug}/calendar-feeds/{token}"`) {
		t.Fatalf("log: %s", e.Logs)
	}
}

// TestMeetingsDST: a 09:00 Monday class of a Europe/Berlin campus is 07:00Z before the October change and 08:00Z after,
// and the window is clipped to the term.
func TestMeetingsDST(t *testing.T) {
	e := newEnv(t)
	e.tx(func(tx *sql.Tx) error {
		var term, course, sec string
		if err := tx.QueryRow(`insert into terms (university_id, name, starts_on, ends_on) values ($1,'Berlin','2026-10-01','2026-11-30') returning id`, e.Fx.UniversityA).Scan(&term); err != nil {
			return err
		}
		if err := tx.QueryRow(`select course_id from sections where id = $1`, e.S1).Scan(&course); err != nil {
			return err
		}
		if err := tx.QueryRow(`insert into sections (university_id, course_id, term_id, number) values ($1,$2,$3,'002') returning id`, e.Fx.UniversityA, course, term).Scan(&sec); err != nil {
			return err
		}
		if _, err := tx.Exec(`insert into enrollments (university_id, section_id, user_id) values ($1,$2,$3)`, e.Fx.UniversityA, sec, e.UserID["stud@a.edu"]); err != nil {
			return err
		}
		_, err := tx.Exec(`insert into section_meetings (university_id, section_id, weekday, starts_at, ends_at) values ($1,$2,1,'09:00','10:30')`, e.Fx.UniversityA, sec)
		return err
	})
	var got []meeting
	if err := pg.WithTx(context.Background(), e.DB, e.Fx.UniversityA, func(tx *sql.Tx) (err error) {
		got, err = meetings(context.Background(), tx, e.UserID["stud@a.edu"], "Europe/Berlin", "2026-10-12")
		return
	}); err != nil {
		t.Fatal(err)
	}
	by := map[string]meeting{}
	for _, m := range got {
		by[m.Day] = m
	}
	if len(got) == 0 || got[0].Day != "20261005" || got[len(got)-1].Day != "20261130" || len(got) != 9 {
		t.Fatalf("days: %+v", got)
	}
	for day, want := range map[string]string{"20261019": "07:00", "20261026": "08:00", "20261130": "08:00"} {
		if m := by[day]; m.Start.UTC().Format("15:04") != want || m.End.Sub(m.Start) != 90*time.Minute || m.Code != "CS101" || m.Number != "002" {
			t.Errorf("%s: %+v want %sZ", day, m, want)
		}
	}
}
