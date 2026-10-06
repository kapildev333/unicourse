package httpx

import (
	"encoding/base64"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Page is keyset pagination over (created_at, id); see page_test.go for the query shape.
var uuidRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

type Page struct {
	Limit  int
	Cursor string
}

type List[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"nextCursor,omitempty"`
}

func ParsePage(r *http.Request) (Page, error) {
	p := Page{Limit: 50, Cursor: r.URL.Query().Get("cursor")}
	if s := r.URL.Query().Get("limit"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil {
			return Page{}, Validation(FieldError{"limit", "must be an integer"})
		}
		if n < 0 {
			return Page{}, Validation(FieldError{"limit", "must not be negative"})
		}
		if n >= 1 {
			p.Limit = min(n, 200)
		}
	}
	if p.Cursor != "" {
		if _, _, ok := p.After(); !ok {
			return Page{}, Validation(FieldError{"cursor", "invalid cursor"})
		}
	}
	return p, nil
}

// After decodes the cursor; ok is false when absent or corrupt.
func (p Page) After() (createdAt time.Time, id string, ok bool) {
	raw, err := base64.RawURLEncoding.DecodeString(p.Cursor)
	if err != nil {
		return time.Time{}, "", false
	}
	ts, id, found := strings.Cut(string(raw), "|")
	if !found || !uuidRe.MatchString(id) {
		return time.Time{}, "", false
	}
	createdAt, err = time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return time.Time{}, "", false
	}
	return createdAt, id, true
}

func NextCursor(createdAt time.Time, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(createdAt.UTC().Format(time.RFC3339Nano) + "|" + id))
}
