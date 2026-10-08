// Package quiz holds the questions, attempts, grading and review of quizzes (Spec 11).
package quiz

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

const base = "/api/v1/universities/{slug}"

// Register mounts the quiz routes, each through tenant.Wrap. Sub-lanes 4b and 4c append their lines after the last one.
func Register(r *httpx.Router, d tenant.Deps) {
	r.Handle("GET", base+"/sections/{id}/assessments/{assessmentId}/questions", tenant.Wrap(d, listQuestions))
	r.Handle("PUT", base+"/sections/{id}/assessments/{assessmentId}/questions", tenant.Wrap(d, putQuestions))
	r.Handle("POST", base+"/sections/{id}/assessments/{assessmentId}/attempts", tenant.Wrap(d, startAttempt))
	r.Handle("GET", base+"/sections/{id}/assessments/{assessmentId}/attempts/{userId}", tenant.Wrap(d, getAttempt))
	r.Handle("PUT", base+"/sections/{id}/assessments/{assessmentId}/attempts/me", tenant.Wrap(d, putAttempt))
}

// sectionExists is the 404 of the check order (RLS hides other universities' sections).
func sectionExists(r *http.Request, tx *sql.Tx, id string) error {
	if !tenant.IsUUID(id) {
		return httpx.NotFound("section")
	}
	var one int
	err := tx.QueryRowContext(r.Context(), `select 1 from sections where id = $1`, id).Scan(&one)
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

// work is the quiz of the path; wrong kind, other section or bad uuid is a 404 "assessment".
type work struct {
	ID, SectionID, MaxPoints    string // MaxPoints numeric::text
	Published, Upcoming, Closed bool   // Upcoming: now() < opens_at; Closed: now() >= closes_at (database clock, D30)
}

// loadWork reads the quiz; lock is "", "for update" or "for share" (a constant, never user input).
func loadWork(r *http.Request, tx *sql.Tx, secID, id, kind, lock string) (work, error) {
	var w work
	if !tenant.IsUUID(id) {
		return w, httpx.NotFound("assessment")
	}
	err := tx.QueryRowContext(r.Context(), `select id, section_id, max_points::text, published, now() < opens_at, now() >= closes_at
		from assessments where id = $1 and section_id = $2 and kind = $3 `+lock, id, secID, kind).
		Scan(&w.ID, &w.SectionID, &w.MaxPoints, &w.Published, &w.Upcoming, &w.Closed)
	if errors.Is(err, sql.ErrNoRows) {
		return w, httpx.NotFound("assessment")
	}
	return w, err
}

// enrolled reports whether the user is a student of the section.
func enrolled(r *http.Request, tx *sql.Tx, secID, userID string) (bool, error) {
	var ok bool
	err := tx.QueryRowContext(r.Context(), `select exists(select 1 from enrollments where section_id = $1 and user_id = $2)`, secID, userID).Scan(&ok)
	return ok, err
}

// editor: the content editors of the section (admin, dept_admin, instructor, TA).
func editor(r *http.Request, tx *sql.Tx, s authz.Scope, secID string) (bool, error) {
	return authz.CanEditContent(r.Context(), tx, s, secID)
}

// target resolves {userId}: "me" is the caller. Editors may name anyone (a non-uuid is a 404 "attempt");
// others must resolve to themselves and be enrolled, else 403. staff says which projection to use.
func target(r *http.Request, tx *sql.Tx, s authz.Scope, secID string) (userID string, staff bool, err error) {
	userID = r.PathValue("userId")
	if userID == "me" {
		userID = s.UserID
	}
	if staff, err = editor(r, tx, s, secID); err != nil {
		return "", false, err
	}
	if staff {
		if !tenant.IsUUID(userID) {
			return "", true, httpx.NotFound("attempt")
		}
		return userID, true, nil
	}
	if userID != s.UserID {
		return "", false, httpx.Forbidden()
	}
	ok, err := enrolled(r, tx, secID, userID)
	return userID, false, allow(ok, err)
}
