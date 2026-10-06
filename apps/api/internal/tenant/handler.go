package tenant

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
)

var uuidRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func IsUUID(s string) bool { return uuidRe.MatchString(s) }

// ScanIDs decodes a json array of ids (to_json / json_agg); the result is never nil.
func ScanIDs(b []byte) ([]string, error) {
	ids := []string{}
	if err := json.Unmarshal(b, &ids); err != nil {
		return nil, err
	}
	if ids == nil {
		ids = []string{}
	}
	return ids, nil
}

// Page parses limit/cursor. Without a cursor after and afterID are both nil, which the keyset
// clause `($1::timestamptz is null or (created_at, id) > ($1, $2::uuid))` treats as "first page";
// fetch p.Limit+1 rows. after is `any` (not time.Time) so "no cursor" is a real SQL NULL.
func Page(r *http.Request) (p httpx.Page, after any, afterID any, err error) {
	if p, err = httpx.ParsePage(r); err != nil {
		return p, nil, nil, err
	}
	if p.Cursor != "" {
		t, id, _ := p.After()
		return p, t, id, nil
	}
	return p, nil, nil, nil
}

// TrimLen trims v and checks it is non-empty and at most max runes.
func TrimLen(field, v string, max int) (string, *httpx.FieldError) {
	v = strings.TrimSpace(v)
	if v == "" {
		return v, &httpx.FieldError{Field: field, Message: "is required"}
	}
	if strings.ContainsRune(v, 0) { // postgres rejects NUL in text (22021)
		return v, &httpx.FieldError{Field: field, Message: "must not contain NUL"}
	}
	if utf8.RuneCountInString(v) > max {
		return v, &httpx.FieldError{Field: field, Message: "must be at most " + strconv.Itoa(max) + " characters"}
	}
	return v, nil
}

func NoFields() error {
	return httpx.Validation(httpx.FieldError{Field: "", Message: "no fields to update"})
}
