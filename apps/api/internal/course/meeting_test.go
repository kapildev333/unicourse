package course

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestValidateMeetings(t *testing.T) {
	ok := MeetingInput{Weekday: 1, StartsAt: "10:00", EndsAt: "11:00"}
	with := func(f func(*MeetingInput)) MeetingInput { m := ok; f(&m); return m }
	rows := func(n int) []MeetingInput {
		out := make([]MeetingInput, n)
		for i := range out {
			out[i] = ok
		}
		return out
	}
	for _, c := range []struct {
		name   string
		in     []MeetingInput
		fields []string
	}{
		{"valid", []MeetingInput{ok}, nil},
		{"weekday 0", []MeetingInput{with(func(m *MeetingInput) { m.Weekday = 0 })}, []string{"meetings[0].weekday"}},
		{"weekday 8", []MeetingInput{with(func(m *MeetingInput) { m.Weekday = 8 })}, []string{"meetings[0].weekday"}},
		{"9:00", []MeetingInput{with(func(m *MeetingInput) { m.StartsAt = "9:00" })}, []string{"meetings[0].startsAt"}},
		{"24:00", []MeetingInput{with(func(m *MeetingInput) { m.StartsAt = "24:00" })}, []string{"meetings[0].startsAt"}},
		{"seconds", []MeetingInput{with(func(m *MeetingInput) { m.StartsAt = "10:00:00" })}, []string{"meetings[0].startsAt"}},
		{"empty", []MeetingInput{with(func(m *MeetingInput) { m.StartsAt = "" })}, []string{"meetings[0].startsAt"}},
		{"equal", []MeetingInput{ok, with(func(m *MeetingInput) { m.EndsAt = "10:00" })}, []string{"meetings[1].endsAt"}},
		{"before", []MeetingInput{ok, with(func(m *MeetingInput) { m.EndsAt = "09:00" })}, []string{"meetings[1].endsAt"}},
		{"location 101", []MeetingInput{with(func(m *MeetingInput) { m.Location = strings.Repeat("é", 101) })}, []string{"meetings[0].location"}},
		{"location 100", []MeetingInput{with(func(m *MeetingInput) { m.Location = strings.Repeat("é", 100) })}, nil},
		{"NUL", []MeetingInput{with(func(m *MeetingInput) { m.Location = "a\x00" })}, []string{"meetings[0].location"}},
		{"21 rows", rows(21), []string{"meetings"}},
		{"20 rows", rows(20), nil},
	} {
		_, d := validateMeetings(c.in)
		var got []string
		for _, f := range d {
			got = append(got, f.Field)
		}
		if !reflect.DeepEqual(got, c.fields) {
			t.Errorf("%s: %v, want %v", c.name, got, c.fields)
		}
	}
	_, d := validateMeetings([]MeetingInput{ok, with(func(m *MeetingInput) { m.EndsAt = "10:00" })})
	if d[0].Message != "must be after startsAt" {
		t.Errorf("message %q", d[0].Message)
	}
	out, _ := validateMeetings([]MeetingInput{{Weekday: 3, StartsAt: "09:00", EndsAt: "10:00", Location: "  R1 "}, with(func(m *MeetingInput) { m.Weekday = 1 })})
	if out[0].Weekday != 1 || out[1].Location != "R1" {
		t.Errorf("trim/sort: %+v", out)
	}
	if out, _ := validateMeetings(nil); out == nil || len(out) != 0 {
		t.Errorf("nil input: %#v", out)
	}
	// P5-L2: error indexes follow the order sent, not the sorted order
	_, d = validateMeetings([]MeetingInput{{Weekday: 5, StartsAt: "09:00", EndsAt: "10:00"}, {Weekday: 1, StartsAt: "11:00", EndsAt: "10:00"}})
	if len(d) != 1 || d[0].Field != "meetings[1].endsAt" {
		t.Errorf("input-order index: %+v", d)
	}
}

func rowsOf(t *testing.T, raw []byte) []Meeting {
	t.Helper()
	var s Section
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("%v %s", err, raw)
	}
	return s.Meetings
}

func meetingCount(e *env, id string) (n int) {
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select count(*) from section_meetings where section_id = $1`, id).Scan(&n)
	})
	return
}

func TestSectionMeetingsRoundTrip(t *testing.T) {
	e := newEnv(t)
	st, raw := e.call(e.Admin, "POST", "/sections", map[string]any{"courseId": e.CourseA, "termId": e.TermA2, "number": "M1", "meetings": []map[string]any{
		{"weekday": 3, "startsAt": "10:00", "endsAt": "11:15", "location": " Room 1 "},
		{"weekday": 1, "startsAt": "09:00", "endsAt": "10:00"},
	}})
	if st != 201 {
		t.Fatalf("create: %d %s", st, raw)
	}
	ms := rowsOf(t, raw)
	if len(ms) != 2 || ms[0].Weekday != 1 || ms[0].Location != "" || ms[1].Weekday != 3 || ms[1].Location != "Room 1" || len(ms[0].ID) != 36 {
		t.Fatalf("create meetings: %s", raw)
	}
	id := jmap(t, raw)["id"].(string)
	same := func(raw []byte, what string) {
		t.Helper()
		if got := rowsOf(t, raw); !reflect.DeepEqual(got, ms) {
			t.Fatalf("%s meetings %+v, want %+v", what, got, ms)
		}
	}
	_, raw = e.call(e.Admin, "GET", "/sections/"+id, nil)
	same(raw, "get")
	_, raw = e.call(e.Admin, "GET", "/sections?termId="+e.TermA2, nil)
	var l struct{ Items []Section }
	json.Unmarshal(raw, &l)
	if len(l.Items) != 1 || !reflect.DeepEqual(l.Items[0].Meetings, ms) {
		t.Fatalf("list: %s", raw)
	}
	// /me/sections as Inst: put Inst on the new section
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`insert into section_staff (university_id, section_id, user_id, role) values ($1,$2,$3,'instructor')`, e.Fx.UniversityA, id, e.UserID["inst@a.edu"])
		return err
	})
	_, raw = e.call(e.Inst, "GET", "/me/sections", nil)
	var me struct {
		Terms []struct{ Sections []Section }
	}
	json.Unmarshal(raw, &me)
	found := false
	for _, tm := range me.Terms {
		for _, it := range tm.Sections {
			if it.ID == id {
				found = reflect.DeepEqual(it.Meetings, ms)
			}
		}
	}
	if !found {
		t.Fatalf("me/sections: %s", raw)
	}
	// replace
	st, raw = e.call(e.Admin, "PATCH", "/sections/"+id, map[string]any{"meetings": []map[string]any{{"weekday": 5, "startsAt": "13:00", "endsAt": "14:00", "location": "X"}}})
	got := rowsOf(t, raw)
	if st != 200 || len(got) != 1 || got[0].Weekday != 5 || got[0].Location != "X" || got[0].ID == ms[0].ID || got[0].ID == ms[1].ID {
		t.Fatalf("replace: %d %s", st, raw)
	}
	ms = got
	_, raw = e.call(e.Admin, "PATCH", "/sections/"+id, map[string]any{"number": "M2"})
	same(raw, "number patch")
	st, raw = e.call(e.Admin, "PATCH", "/sections/"+id, map[string]any{"meetings": []any{}})
	if st != 200 || !strings.Contains(string(raw), `"meetings":[]`) {
		t.Fatalf("clear: %d %s", st, raw)
	}
	st, raw = e.call(e.Admin, "PATCH", "/sections/"+id, map[string]any{"meetings": nil})
	if c, f := errDetails(raw); st != 400 || c != "validation_failed" || !reflect.DeepEqual(f, []string{""}) {
		t.Fatalf("null: %d %s", st, raw)
	}
	_, raw = e.call(e.Admin, "GET", "/sections/"+e.SectionA, nil)
	if !strings.Contains(string(raw), `"meetings":[]`) {
		t.Fatalf("empty serialisation: %s", raw)
	}
}

func TestSectionMeetingsValidation(t *testing.T) {
	e := newEnv(t)
	good := map[string]any{"weekday": 1, "startsAt": "10:00", "endsAt": "11:00"}
	bad := map[string]any{"weekday": 1, "startsAt": "10:00", "endsAt": "09:00"}
	st, raw := e.call(e.Admin, "POST", "/sections", map[string]any{"courseId": e.CourseA, "termId": e.TermA2, "number": "", "meetings": []any{good, bad}})
	if _, f := errDetails(raw); st != 400 || !reflect.DeepEqual(f, []string{"number", "meetings[1].endsAt"}) {
		t.Fatalf("both: %d %s", st, raw)
	}
	// out-of-order rows: the index is the one the client sent (P5-L2)
	late := map[string]any{"weekday": 5, "startsAt": "10:00", "endsAt": "11:00"}
	early := map[string]any{"weekday": 1, "startsAt": "11:00", "endsAt": "10:00"}
	st, raw = e.call(e.Admin, "POST", "/sections", map[string]any{"courseId": e.CourseA, "termId": e.TermA2, "number": "Z", "meetings": []any{late, early}})
	if _, f := errDetails(raw); st != 400 || !reflect.DeepEqual(f, []string{"meetings[1].endsAt"}) {
		t.Fatalf("input order: %d %s", st, raw)
	}
	st, raw = e.call(e.Admin, "POST", "/sections", map[string]any{"courseId": e.CourseA, "termId": e.TermA2, "number": "Z", "meetings": []any{map[string]any{"weekday": 8, "startsAt": "10:00", "endsAt": "11:00"}}})
	if _, f := errDetails(raw); st != 400 || !reflect.DeepEqual(f, []string{"meetings[0].weekday"}) {
		t.Fatalf("weekday: %d %s", st, raw)
	}
	many := make([]any, 21)
	for i := range many {
		many[i] = good
	}
	st, raw = e.call(e.Admin, "POST", "/sections", map[string]any{"courseId": e.CourseA, "termId": e.TermA2, "number": "Z", "meetings": many})
	if _, f := errDetails(raw); st != 400 || !reflect.DeepEqual(f, []string{"meetings"}) {
		t.Fatalf("21 rows: %d %s", st, raw)
	}
	st, raw = e.call(e.Admin, "POST", "/sections", map[string]any{"courseId": e.CourseA, "termId": e.TermA2, "number": "Z", "meetings": []any{map[string]any{"day": 1, "weekday": 1, "startsAt": "10:00", "endsAt": "11:00"}}})
	if c, _ := errDetails(raw); st != 400 || c != "validation_failed" {
		t.Fatalf("unknown key: %d %s", st, raw)
	}
	st, raw = e.call(e.Admin, "POST", "/sections", map[string]any{"courseId": e.CourseA, "termId": e.TermA2, "number": "Z", "meetings": []any{map[string]any{"weekday": "Mon"}}})
	if c, _ := errDetails(raw); st != 400 || c != "validation_failed" {
		t.Fatalf("type error: %d %s", st, raw)
	}
	// a failing PATCH leaves the stored meetings alone
	e.call(e.Admin, "PATCH", "/sections/"+e.SectionA, map[string]any{"meetings": []any{good}})
	st, raw = e.call(e.Admin, "PATCH", "/sections/"+e.SectionA, map[string]any{"meetings": []any{bad}})
	if st != 400 || meetingCount(e, e.SectionA) != 1 {
		t.Fatalf("patch bad: %d %s (rows %d)", st, raw, meetingCount(e, e.SectionA))
	}
}

func TestSectionMeetingsAudit(t *testing.T) {
	e := newEnv(t)
	m := []map[string]any{{"weekday": 1, "startsAt": "09:00", "endsAt": "10:00"}, {"weekday": 2, "startsAt": "09:00", "endsAt": "10:00", "location": "R"}}
	_, raw := e.call(e.Admin, "POST", "/sections", map[string]any{"courseId": e.CourseA, "termId": e.TermA2, "number": "A1", "meetings": m})
	id := jmap(t, raw)["id"].(string)
	_, _, after := e.audit("section.create", id)
	var a struct{ Meetings []map[string]any }
	json.Unmarshal([]byte(after.String), &a)
	if len(a.Meetings) != 2 || a.Meetings[0]["id"] != nil {
		t.Fatalf("create audit: %s", after.String)
	}
	e.call(e.Admin, "PATCH", "/sections/"+id, map[string]any{"meetings": m[:1]})
	_, before, after := e.audit("section.update", id)
	var b struct{ Meetings []map[string]any }
	json.Unmarshal([]byte(before.String), &b)
	json.Unmarshal([]byte(after.String), &a)
	if len(b.Meetings) != 2 || len(a.Meetings) != 1 {
		t.Fatalf("update audit: %s -> %s", before.String, after.String)
	}
	e.call(e.Admin, "DELETE", "/sections/"+id, nil)
	_, before, _ = e.audit("section.delete", id)
	json.Unmarshal([]byte(before.String), &b)
	if len(b.Meetings) != 1 {
		t.Fatalf("delete audit: %s", before.String)
	}
}

func TestSectionMeetingsCascadeAndAuthz(t *testing.T) {
	e := newEnv(t)
	good := []map[string]any{{"weekday": 1, "startsAt": "10:00", "endsAt": "11:00"}}
	_, raw := e.call(e.Admin, "POST", "/sections", map[string]any{"courseId": e.CourseA, "termId": e.TermA2, "number": "C1", "meetings": good})
	id := jmap(t, raw)["id"].(string)
	for who, ck := range map[string]*http.Cookie{"daout": e.DaOut, "inst": e.Inst, "stud": e.Stud} {
		if st, raw := e.call(ck, "PATCH", "/sections/"+id, map[string]any{"meetings": []any{}}); st != 403 {
			t.Errorf("%s: %d %s", who, st, raw)
		}
	}
	if meetingCount(e, id) != 1 {
		t.Fatal("rows changed")
	}
	if st, _ := e.call(e.Admin, "DELETE", "/sections/"+id, nil); st != 204 || meetingCount(e, id) != 0 {
		t.Fatalf("cascade: %d rows", meetingCount(e, id))
	}
}

func TestParseSlots(t *testing.T) {
	long := strings.Repeat("x", 101)
	cases := []struct {
		cell string
		want []MeetingInput
		bad  string
	}{
		{"Mon 10:00-11:15 @ Room 101; wed 10:00-11:15", []MeetingInput{{1, "10:00", "11:15", "Room 101"}, {3, "10:00", "11:15", ""}}, ""},
		{"MON 10:00 - 11:15", []MeetingInput{{1, "10:00", "11:15", ""}}, ""},
		{"Mon 10:00-11:15 @", []MeetingInput{{1, "10:00", "11:15", ""}}, ""},
		{"Mon 10:00-11:00; ", []MeetingInput{{1, "10:00", "11:00", ""}}, ""},
		{"Mon 11:00-12:00; Mon 10:00-11:00", []MeetingInput{{1, "10:00", "11:00", ""}, {1, "11:00", "12:00", ""}}, ""},
		{"", nil, ""},
		{" ; ;", nil, ""},
		{"Mon 10:00-10:00", nil, `meeting "Mon 10:00-10:00": endsAt must be after startsAt`},
		{"Xyz 10:00-11:00", nil, `meeting "Xyz 10:00-11:00": unknown weekday Xyz`},
		{"Monday 10:00-11:00", nil, `meeting "Monday 10:00-11:00": unknown weekday Monday`},
		{"Mon 9:00-10:00", nil, `meeting "Mon 9:00-10:00": startsAt must be a time HH:MM`},
		{"Mon 10:00 11:00", nil, `meeting "Mon 10:00 11:00" must look like Mon 10:00-11:15 @ Room 101`},
		{"Mon 10:00-11:15 Room", nil, `meeting "Mon 10:00-11:15 Room" must look like Mon 10:00-11:15 @ Room 101`},
		{"Mon 10:00-11:00 @ " + long, nil, `meeting "Mon 10:00-11:00 @ ` + long + `": location must be at most 100 characters`},
		{strings.TrimSuffix(strings.Repeat("Mon 10:00-11:00;", 21), ";"), nil, "at most 20 meetings"},
	}
	for _, c := range cases {
		got, bad := parseSlots(c.cell)
		if bad != c.bad || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%q: got %v %q, want %v %q", c.cell, got, bad, c.want, c.bad)
		}
	}
}
