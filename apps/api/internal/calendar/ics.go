package calendar

import (
	"strings"
	"time"
	"unicode/utf8"
)

const icsTime = "20060102T150405Z"

// event is one VEVENT. Times are instants, written in UTC (no TZID, so no VTIMEZONE is needed).
type event struct {
	UID, Summary, Location, Description string
	Start, End                          time.Time
	Transparent                         bool // deadlines do not block time
}

// escape is RFC 5545 TEXT escaping: backslash, semicolon, comma and newline; a bare CR is dropped.
var escape = strings.NewReplacer(`\`, `\\`, ";", `\;`, ",", `\,`, "\r\n", `\n`, "\n", `\n`, "\r", "").Replace

// fold ends line with CRLF and splits it into physical lines of at most 75 octets (RFC 5545 3.1); each
// continuation starts with one space, which counts. It never cuts inside a UTF-8 sequence.
func fold(line string) string {
	var b strings.Builder
	limit := 75
	for len(line) > limit {
		cut := limit
		for cut > 0 && !utf8.RuneStart(line[cut]) {
			cut--
		}
		b.WriteString(line[:cut] + "\r\n ")
		line, limit = line[cut:], 74
	}
	b.WriteString(line + "\r\n")
	return b.String()
}

// render writes the whole calendar; every line, the last included, ends in CRLF.
func render(name string, stamp time.Time, evs []event) string {
	var b strings.Builder
	w := func(line string) { b.WriteString(fold(line)) }
	for _, l := range []string{"BEGIN:VCALENDAR", "VERSION:2.0", "PRODID:-//UniCourse//Calendar feed//EN", "CALSCALE:GREGORIAN", "METHOD:PUBLISH",
		"X-WR-CALNAME:UniCourse: " + escape(name), "REFRESH-INTERVAL;VALUE=DURATION:PT6H", "X-PUBLISHED-TTL:PT6H"} {
		w(l)
	}
	for _, e := range evs {
		w("BEGIN:VEVENT")
		w("UID:" + e.UID)
		w("DTSTAMP:" + stamp.UTC().Format(icsTime))
		w("DTSTART:" + e.Start.UTC().Format(icsTime))
		w("DTEND:" + e.End.UTC().Format(icsTime))
		w("SUMMARY:" + escape(e.Summary))
		if e.Location != "" {
			w("LOCATION:" + escape(e.Location))
		}
		if e.Description != "" {
			w("DESCRIPTION:" + escape(e.Description))
		}
		if e.Transparent {
			w("TRANSP:TRANSPARENT")
		}
		w("END:VEVENT")
	}
	w("END:VCALENDAR")
	return b.String()
}
