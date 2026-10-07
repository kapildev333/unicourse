package tenant

import (
	"database/sql"
	"encoding/json"
	"math/big"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kapildev333/unicourse/apps/api/internal/audit"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
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

var decimalRe = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?$`)

// Decimal validates n (a JSON number or a CSV cell) as a decimal with at most places decimals in
// [min, max] (decimal strings) and returns it formatted with exactly places decimals ("3" -> "3.0"
// for places 1). Exact (math/big); exponents and fractions are rejected.
func Decimal(field string, n json.Number, places int, min, max string) (string, *httpx.FieldError) {
	bad := func(m string) (string, *httpx.FieldError) {
		return "", &httpx.FieldError{Field: field, Message: m}
	}
	v := strings.TrimSpace(string(n))
	if v == "" {
		return bad("is required")
	}
	if !decimalRe.MatchString(v) {
		return bad("must be a number")
	}
	r, _ := new(big.Rat).SetString(v)
	scaled := new(big.Rat).Mul(r, new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(places)), nil)))
	if !scaled.IsInt() {
		if places == 1 {
			return bad("must have at most 1 decimal")
		}
		return bad("must have at most " + strconv.Itoa(places) + " decimals")
	}
	lo, _ := new(big.Rat).SetString(min)
	hi, _ := new(big.Rat).SetString(max)
	if r.Cmp(lo) < 0 || r.Cmp(hi) > 0 {
		return bad("must be between " + min + " and " + max)
	}
	return r.FloatString(places), nil
}

// Opt is a PATCH field of any JSON type that tells absent from explicit null (D27); OptStr predates it.
type Opt[T any] struct {
	Set, Null bool
	V         T
}

func (o *Opt[T]) UnmarshalJSON(b []byte) error {
	o.Set = true
	if string(b) == "null" {
		o.Null = true
		return nil
	}
	return json.Unmarshal(b, &o.V)
}

func NoFields() error {
	return httpx.Validation(httpx.FieldError{Field: "", Message: "no fields to update"})
}

const entityTypes = "department, term, membership, invitation, course, section, staff, enrollment, assessment, final_grade, grade_scale, announcement, week, material"

// listAuditLog serves GET /audit-log (admin only); every filter error is reported at once.
func listAuditLog(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	if !authz.IsAdmin(s) {
		return 0, nil, httpx.Forbidden()
	}
	p, err := httpx.ParsePage(r)
	if err != nil {
		return 0, nil, err
	}
	q := r.URL.Query()
	var f audit.Filter
	var d []httpx.FieldError
	bad := func(field, msg string) { d = append(d, httpx.FieldError{Field: field, Message: msg}) }
	if v := q.Get("entityType"); v != "" {
		if !strings.Contains(", "+entityTypes+",", ", "+v+",") {
			bad("entityType", "must be one of "+entityTypes)
		}
		f.EntityType = v
	}
	for _, c := range []struct {
		name string
		dst  *string
	}{{"entityId", &f.EntityID}, {"actorUserId", &f.ActorUserID}} {
		if v := q.Get(c.name); v != "" {
			if !IsUUID(v) {
				bad(c.name, "must be a uuid")
			}
			*c.dst = v
		}
	}
	for _, c := range []struct {
		name string
		dst  *time.Time
	}{{"from", &f.From}, {"to", &f.To}} {
		if v := q.Get(c.name); v != "" {
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				bad(c.name, "must be an RFC 3339 date-time")
			}
			*c.dst = t
		}
	}
	if len(d) > 0 {
		return 0, nil, httpx.Validation(d...)
	}
	out, err := audit.List(r.Context(), tx, f, p)
	if err != nil {
		return 0, nil, err
	}
	return http.StatusOK, out, nil
}
