package httpx

import (
	"errors"
	"net/http/httptest"
	"testing"
	"time"
)

const testID = "0190a1b2-c3d4-4e5f-8a6b-7c8d9e0f1a2b"

func parse(t *testing.T, q string) (Page, error) {
	t.Helper()
	return ParsePage(httptest.NewRequest("GET", "/x?"+q, nil))
}

func TestParsePageLimit(t *testing.T) {
	for q, want := range map[string]int{"": 50, "limit=0": 50, "limit=500": 200, "limit=7": 7} {
		p, err := parse(t, q)
		if err != nil || p.Limit != want {
			t.Errorf("%q: %v %v", q, p, err)
		}
	}
	assertField(t, "limit=abc", "limit")
	assertField(t, "limit=-5", "limit")
}

func assertField(t *testing.T, q, field string) {
	t.Helper()
	_, err := parse(t, q)
	var e *Error
	if !errors.As(err, &e) || e.Code != "validation_failed" || e.Details[0].Field != field {
		t.Errorf("%q: %#v", q, err)
	}
}

// Consumers must query:
//
//	WHERE (created_at, id) > ($1, $2) ORDER BY created_at, id LIMIT $3+1
//
// A keyset cursor (not an offset) means deleting the row the cursor points at
// neither skips nor repeats items.
func TestCursorRoundTrip(t *testing.T) {
	at := time.Date(2026, 10, 5, 12, 0, 0, 123456789, time.UTC)
	p, err := parse(t, "cursor="+NextCursor(at, testID))
	if err != nil {
		t.Fatal(err)
	}
	gotAt, gotID, ok := p.After()
	if !ok || !gotAt.Equal(at) || gotID != testID {
		t.Fatal(gotAt, gotID, ok)
	}
	p, _ = parse(t, "")
	if _, _, ok := p.After(); ok {
		t.Fatal("empty cursor must not be ok")
	}
}

func TestCorruptCursor(t *testing.T) {
	assertField(t, "cursor=!!!", "cursor")
	assertField(t, "cursor=bm9waXBl", "cursor") // "nopipe"
}

func TestCursorRejectsBadID(t *testing.T) {
	at := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	assertField(t, "cursor="+NextCursor(at, "not-a-uuid"), "cursor")
	assertField(t, "cursor="+NextCursor(at, testID+"|x"), "cursor")
	assertField(t, "cursor="+NextCursor(at, "x|"+testID), "cursor")
}
