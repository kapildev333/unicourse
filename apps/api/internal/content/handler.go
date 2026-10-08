// Package content holds a section's description, announcements, weeks and materials (spec 6).
package content

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

const base = "/api/v1/universities/{slug}"

// Register mounts the content routes, each through tenant.Wrap.
func Register(r *httpx.Router, d tenant.Deps) {
	r.Handle("GET", "/api/v1/upload-policy", uploadPolicy(d))
	r.Handle("PUT", base+"/sections/{id}/description", tenant.Wrap(d, putDescription))
	r.Handle("GET", base+"/sections/{id}/announcements", tenant.Wrap(d, listAnnouncements))
	r.Handle("POST", base+"/sections/{id}/announcements", tenant.Wrap(d, createAnnouncement(d.BaseURL)))
	r.Handle("PATCH", base+"/sections/{id}/announcements/{announcementId}", tenant.Wrap(d, updateAnnouncement))
	r.Handle("DELETE", base+"/sections/{id}/announcements/{announcementId}", tenant.Wrap(d, deleteAnnouncement))
	r.Handle("GET", base+"/sections/{id}/weeks", tenant.Wrap(d, listWeeks))
	r.Handle("POST", base+"/sections/{id}/weeks", tenant.Wrap(d, createWeek))
	r.Handle("PATCH", base+"/sections/{id}/weeks/{weekId}", tenant.Wrap(d, updateWeek))
	r.Handle("DELETE", base+"/sections/{id}/weeks/{weekId}", tenant.Wrap(d, deleteWeek(d.Files)))
	r.Handle("POST", base+"/sections/{id}/weeks/{weekId}/materials", tenant.Wrap(d, uploadMaterial(d)))
	r.Handle("PATCH", base+"/sections/{id}/materials/{materialId}", tenant.Wrap(d, updateMaterial))
	r.Handle("DELETE", base+"/sections/{id}/materials/{materialId}", tenant.Wrap(d, deleteMaterial(d.Files)))
	r.Handle("GET", base+"/sections/{id}/materials/{materialId}/download", tenant.Wrap(d, downloadMaterial(d.Files)))
}

// sectionExists is the 404 of the check order; lock takes the per-section write lock (design decision 8).
func sectionExists(r *http.Request, tx *sql.Tx, id string, lock bool) error {
	if !tenant.IsUUID(id) {
		return httpx.NotFound("section")
	}
	q := `select 1 from sections where id = $1`
	if lock {
		q += ` for no key update`
	}
	var one int
	err := tx.QueryRowContext(r.Context(), q, id).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return httpx.NotFound("section")
	}
	return err
}

// allow turns a predicate result into the 403 of the check order.
func allow(ok bool, err error) error {
	if err != nil {
		return err
	}
	if !ok {
		return httpx.Forbidden()
	}
	return nil
}

// line validates a one-line title: tenant.TrimLen, then no control characters.
func line(field, v string, max int) (string, *httpx.FieldError) {
	v, fe := tenant.TrimLen(field, v, max)
	if fe != nil {
		return v, fe
	}
	if strings.IndexFunc(v, unicode.IsControl) >= 0 {
		return v, &httpx.FieldError{Field: field, Message: "must not contain control characters"}
	}
	return v, nil
}

// optText validates multi-line text that may be empty: trimmed, inner line breaks kept, no NUL.
func optText(field, v string, max int) (string, *httpx.FieldError) {
	v = strings.TrimSpace(v)
	if strings.ContainsRune(v, 0) {
		return v, &httpx.FieldError{Field: field, Message: "must not contain NUL"}
	}
	if utf8.RuneCountInString(v) > max {
		return v, &httpx.FieldError{Field: field, Message: "must be at most " + strconv.Itoa(max) + " characters"}
	}
	return v, nil
}

type person struct {
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
}
