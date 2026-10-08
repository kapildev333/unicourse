package course

import (
	"context"
	"database/sql"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
)

const maxMeetings = 20

// MeetingInput is one weekly slot as written (request body, import cell, audit payload).
type MeetingInput struct {
	Weekday  int    `json:"weekday"`
	StartsAt string `json:"startsAt"`
	EndsAt   string `json:"endsAt"`
	Location string `json:"location"`
}

// Meeting is the contract resource.
type Meeting struct {
	ID string `json:"id"`
	MeetingInput
}

// meetingsJSON aggregates a section's meetings; s is the sections alias. Used by sectionCols and loadForWrite.
const meetingsJSON = `coalesce((select json_agg(json_build_object('id', m.id, 'weekday', m.weekday,
	'startsAt', to_char(m.starts_at, 'HH24:MI'), 'endsAt', to_char(m.ends_at, 'HH24:MI'), 'location', m.location)
	order by m.weekday, m.starts_at, m.ends_at, m.location)
	from section_meetings m where m.section_id = s.id), '[]')`

// hm reports whether s is exactly HH:MM (00:00..23:59).
func hm(s string) bool {
	if len(s) != 5 {
		return false
	}
	_, err := time.Parse("15:04", s)
	return err == nil
}

// checkMeeting trims location and returns errors with bare field names, in field order.
func checkMeeting(m MeetingInput) (MeetingInput, []httpx.FieldError) {
	var d []httpx.FieldError
	bad := func(f, msg string) { d = append(d, httpx.FieldError{Field: f, Message: msg}) }
	if m.Weekday < 1 || m.Weekday > 7 {
		bad("weekday", "must be between 1 and 7")
	}
	if !hm(m.StartsAt) {
		bad("startsAt", "must be a time HH:MM")
	}
	if !hm(m.EndsAt) {
		bad("endsAt", "must be a time HH:MM")
	} else if hm(m.StartsAt) && m.EndsAt <= m.StartsAt {
		bad("endsAt", "must be after startsAt")
	}
	m.Location = strings.TrimSpace(m.Location)
	if strings.ContainsRune(m.Location, 0) {
		bad("location", "must not contain NUL")
	} else if utf8.RuneCountInString(m.Location) > 100 {
		bad("location", "must be at most 100 characters")
	}
	return m, d
}

// validateMeetings checks a request list in the order the client sent it (error indexes refer to
// that order); only the returned rows are sorted. Never returns nil rows.
func validateMeetings(in []MeetingInput) ([]MeetingInput, []httpx.FieldError) {
	if len(in) > maxMeetings {
		return []MeetingInput{}, []httpx.FieldError{{Field: "meetings", Message: "add at most 20 meeting times"}}
	}
	out := make([]MeetingInput, 0, len(in))
	var d []httpx.FieldError
	for i, m := range in {
		m, errs := checkMeeting(m)
		for _, e := range errs {
			d = append(d, httpx.FieldError{Field: "meetings[" + strconv.Itoa(i) + "]." + e.Field, Message: e.Message})
		}
		out = append(out, m)
	}
	if len(d) == 0 {
		d = overlaps(out)
	}
	sortMeetings(out)
	return out, d
}

var dayNames = [...]string{"", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}

// overlaps reports the first pair (in sent order, 1-based as the editor shows them) whose [start, end) intersect on one day.
// Only called on rows that passed checkMeeting, so times compare as strings.
func overlaps(ms []MeetingInput) []httpx.FieldError {
	for i, a := range ms {
		for j := i + 1; j < len(ms); j++ {
			b := ms[j]
			if a.Weekday == b.Weekday && a.StartsAt < b.EndsAt && b.StartsAt < a.EndsAt {
				return []httpx.FieldError{{Field: "meetings", Message: "Meeting " + strconv.Itoa(i+1) + " and meeting " + strconv.Itoa(j+1) + " overlap on " + dayNames[a.Weekday] + "."}}
			}
		}
	}
	return nil
}

func sortMeetings(ms []MeetingInput) {
	slices.SortFunc(ms, func(a, b MeetingInput) int {
		if a.Weekday != b.Weekday {
			return a.Weekday - b.Weekday
		}
		return strings.Compare(a.StartsAt+"|"+a.EndsAt+"|"+a.Location, b.StartsAt+"|"+b.EndsAt+"|"+b.Location)
	})
}

// insertMeetings inserts ms[i] for section secs[i] in one statement; no-op when empty.
func insertMeetings(ctx context.Context, tx *sql.Tx, uni string, secs []string, ms []MeetingInput) error {
	if len(ms) == 0 {
		return nil
	}
	wd, a, b, l := make([]int32, len(ms)), make([]string, len(ms)), make([]string, len(ms)), make([]string, len(ms))
	for i, m := range ms {
		wd[i], a[i], b[i], l[i] = int32(m.Weekday), m.StartsAt, m.EndsAt, m.Location
	}
	_, err := tx.ExecContext(ctx, `insert into section_meetings (university_id, section_id, weekday, starts_at, ends_at, location)
		select $1, s, w, a::time, b::time, l from unnest($2::uuid[], $3::int[], $4::text[], $5::text[], $6::text[]) as x(s, w, a, b, l)`,
		uni, secs, wd, a, b, l)
	return err
}

// replaceMeetings deletes the section's meetings and inserts ms (caller holds the section row lock).
func replaceMeetings(ctx context.Context, tx *sql.Tx, uni, sectionID string, ms []MeetingInput) error {
	if _, err := tx.ExecContext(ctx, `delete from section_meetings where section_id = $1`, sectionID); err != nil {
		return err
	}
	secs := make([]string, len(ms))
	for i := range secs {
		secs[i] = sectionID
	}
	return insertMeetings(ctx, tx, uni, secs, ms)
}

var slotRe = regexp.MustCompile(`^([A-Za-z]+)\s+([0-9:]+)\s*-\s*([0-9:]+)\s*(?:@(.*))?$`)
var weekdayOf = map[string]int{"mon": 1, "tue": 2, "wed": 3, "thu": 4, "fri": 5, "sat": 6, "sun": 7}

// parseSlots parses an import cell ("Mon 10:00-11:15 @ Room 101; Wed ..."); "" or only separators -> nil, "".
// bad is the row message. Ceiling: a location cannot contain ";".
func parseSlots(cell string) (ms []MeetingInput, bad string) {
	var parts []string
	for _, p := range strings.Split(cell, ";") {
		if p = strings.TrimSpace(p); p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) > maxMeetings {
		return nil, "at most 20 meetings"
	}
	for _, p := range parts {
		g := slotRe.FindStringSubmatch(p)
		if g == nil {
			return nil, `meeting "` + p + `" must look like Mon 10:00-11:15 @ Room 101`
		}
		wd, ok := weekdayOf[strings.ToLower(g[1])]
		if !ok {
			return nil, `meeting "` + p + `": unknown weekday ` + g[1]
		}
		m, errs := checkMeeting(MeetingInput{Weekday: wd, StartsAt: g[2], EndsAt: g[3], Location: g[4]})
		if len(errs) > 0 {
			return nil, `meeting "` + p + `": ` + errs[0].Field + " " + errs[0].Message
		}
		ms = append(ms, m)
	}
	if o := overlaps(ms); o != nil {
		return nil, o[0].Message
	}
	sortMeetings(ms)
	return ms, ""
}
