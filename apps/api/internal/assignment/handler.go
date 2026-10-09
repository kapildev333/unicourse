// Package assignment holds the submissions of assignments: text, files, download and grading (Spec 11).
package assignment

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

const base = "/api/v1/universities/{slug}"

// Register mounts the submission routes, each through tenant.Wrap.
func Register(r *httpx.Router, d tenant.Deps) {
	const sub = base + "/sections/{id}/assessments/{assessmentId}/submissions"
	r.Handle("GET", sub, tenant.Wrap(d, listSubmissions))
	r.Handle("GET", sub+"/{userId}", tenant.Wrap(d, getSubmission))
	r.Handle("PUT", sub+"/me", tenant.Wrap(d, putSubmission))
	r.Handle("POST", sub+"/me/files", tenant.Wrap(d, uploadFile(d)))
	r.Handle("DELETE", sub+"/me/files/{fileId}", tenant.Wrap(d, deleteFile(d)))
	r.Handle("GET", sub+"/{userId}/files/{fileId}/download", tenant.Wrap(d, downloadFile(d)))
	r.Handle("PATCH", sub+"/{userId}", tenant.Wrap(d, gradeSubmission))
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

// work is the assignment of the path; wrong kind, other section or bad uuid -> 404 "assessment".
type work struct {
	ID, MaxPoints, Penalty      string // MaxPoints, Penalty numeric::text
	Published, Upcoming, Closed bool   // Upcoming: now() < opens_at; Closed: now() >= coalesce(late_until, closes_at)
	Late                        bool   // now() >= closes_at: a student write is late
}

// loadWork reads the assignment; lock is "", "for share" or "for update". Window state is the database's now() (D30).
func loadWork(r *http.Request, tx *sql.Tx, secID, id, lock string) (w work, err error) {
	if !tenant.IsUUID(id) {
		return w, httpx.NotFound("assessment")
	}
	err = tx.QueryRowContext(r.Context(), `select id, max_points::text, published, now() < opens_at, now() >= coalesce(late_until, closes_at),
		now() >= closes_at, late_penalty_percent::text from assessments where id = $1 and section_id = $2 and kind = 'assignment' `+lock, id, secID).
		Scan(&w.ID, &w.MaxPoints, &w.Published, &w.Upcoming, &w.Closed, &w.Late, &w.Penalty)
	if errors.Is(err, sql.ErrNoRows) {
		return w, httpx.NotFound("assessment")
	}
	return
}

// window is the 409 of writes outside opens_at..closes_at.
func (w work) window() error {
	switch {
	case w.Upcoming:
		return httpx.Conflict("not open yet")
	case w.Closed:
		return httpx.Conflict("closed")
	}
	return nil
}

func enrolled(ctx context.Context, tx *sql.Tx, secID, userID string) (ok bool, err error) {
	err = tx.QueryRowContext(ctx, `select exists(select 1 from enrollments where section_id = $1 and user_id = $2)`, secID, userID).Scan(&ok)
	return
}

// mustEnroll is the 403 of the student routes.
func mustEnroll(r *http.Request, tx *sql.Tx, s authz.Scope, secID string) error {
	return allow(enrolled(r.Context(), tx, secID, s.UserID))
}

// target resolves {userId}: "me" is the caller. Editors may name anyone (a non-uuid is 404);
// others must resolve to themselves and be enrolled, else 403. staff reports which projection to use.
func target(r *http.Request, tx *sql.Tx, s authz.Scope, secID string) (userID string, staff bool, err error) {
	userID = r.PathValue("userId")
	if userID == "me" {
		userID = s.UserID
	}
	if staff, err = authz.CanEditContent(r.Context(), tx, s, secID); err != nil {
		return
	}
	if staff {
		if !tenant.IsUUID(userID) {
			err = httpx.NotFound("submission")
		}
		return
	}
	if userID != s.UserID {
		return "", false, httpx.Forbidden()
	}
	err = mustEnroll(r, tx, s, secID)
	return
}

// editor is the 403 of the staff routes.
func editor(r *http.Request, tx *sql.Tx, s authz.Scope, secID string) error {
	return allow(authz.CanEditContent(r.Context(), tx, s, secID))
}
