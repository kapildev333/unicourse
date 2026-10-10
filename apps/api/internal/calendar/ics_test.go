package calendar

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestEscape(t *testing.T) {
	for in, want := range map[string]string{
		"plain":                 "plain",
		`a,b;c\d`:               `a\,b\;c\\d`,
		"line1\nline2":          `line1\nline2`,
		"crlf\r\nbreak":         `crlf\nbreak`,
		"Intro, Part 1; Basics": `Intro\, Part 1\; Basics`,
	} {
		if got := escape(in); got != want {
			t.Errorf("escape(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFold(t *testing.T) {
	short := "SUMMARY:short"
	if got := fold(short); got != short+"\r\n" {
		t.Fatalf("short: %q", got)
	}
	// 80 ASCII octets: 75, then a continuation space plus the last 5
	long := strings.Repeat("a", 80)
	if got, want := fold(long), strings.Repeat("a", 75)+"\r\n "+strings.Repeat("a", 5)+"\r\n"; got != want {
		t.Fatalf("80 octets: %q", got)
	}
	// a two-octet rune that would straddle the cut moves whole to the next line
	multi := strings.Repeat("a", 74) + "é" + "tail"
	got := fold(multi)
	if want := strings.Repeat("a", 74) + "\r\n é" + "tail\r\n"; got != want {
		t.Fatalf("multibyte: %q want %q", got, want)
	}
	// every physical line is at most 75 octets, valid UTF-8, and unfolding restores the input
	mixed := strings.Repeat("日本語é,x", 60)
	got = fold(mixed)
	for _, l := range strings.Split(strings.TrimSuffix(got, "\r\n"), "\r\n") {
		if len(l) > 75 || !utf8.ValidString(l) {
			t.Fatalf("line of %d octets: %q", len(l), l)
		}
	}
	if un := strings.ReplaceAll(strings.TrimSuffix(got, "\r\n"), "\r\n ", ""); un != mixed {
		t.Fatalf("unfold differs")
	}
}

func TestCRLF(t *testing.T) {
	at := time.Date(2026, 10, 26, 8, 0, 0, 0, time.UTC)
	out := render("Fall\nSemester", at, []event{
		{UID: "a@unicourse", Summary: "multi\nline", Location: "Room 1", Start: at, End: at.Add(time.Hour)},
		{UID: "b@unicourse", Summary: "deadline", Description: "http://x", Start: at, End: at, Transparent: true},
	})
	if strings.Contains(strings.ReplaceAll(out, "\r\n", ""), "\n") || strings.Contains(strings.ReplaceAll(out, "\r\n", ""), "\r") || !strings.HasSuffix(out, "END:VCALENDAR\r\n") {
		t.Fatalf("bare line break: %q", out)
	}
	for _, want := range []string{"BEGIN:VCALENDAR\r\nVERSION:2.0\r\n", "METHOD:PUBLISH\r\n", "X-WR-CALNAME:UniCourse: Fall\\nSemester\r\n", "DTSTART:20261026T080000Z\r\n",
		"DTEND:20261026T090000Z\r\n", "DTSTAMP:20261026T080000Z\r\n", "LOCATION:Room 1\r\n", "SUMMARY:multi\\nline\r\n", "TRANSP:TRANSPARENT\r\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %q", want, out)
		}
	}
	if strings.Count(out, "LOCATION:") != 1 { // omitted when empty
		t.Errorf("LOCATION must be omitted when empty: %q", out)
	}
}
