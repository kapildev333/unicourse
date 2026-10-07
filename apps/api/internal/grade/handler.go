// Package grade holds the grade scale, assessments, scores, gradebook, final grades and GPA.
package grade

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

const base = "/api/v1/universities/{slug}"

// Register mounts the grading routes, each through tenant.Wrap.
func Register(r *httpx.Router, d tenant.Deps) {
	r.Handle("GET", base+"/grade-scale", tenant.Wrap(d, getScale))
	r.Handle("PUT", base+"/grade-scale", tenant.Wrap(d, putScale))
	r.Handle("GET", base+"/sections/{id}/assessments", tenant.Wrap(d, listAssessments))
	r.Handle("POST", base+"/sections/{id}/assessments", tenant.Wrap(d, createAssessment))
	r.Handle("PATCH", base+"/sections/{id}/assessments/{assessmentId}", tenant.Wrap(d, updateAssessment))
	r.Handle("DELETE", base+"/sections/{id}/assessments/{assessmentId}", tenant.Wrap(d, deleteAssessment))
	r.Handle("PUT", base+"/sections/{id}/assessments/{assessmentId}/scores", tenant.Wrap(d, putScores))
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

// loadAssessment reads one assessment of the section; unknown id, bad uuid or another section's id -> 404.
func loadAssessment(r *http.Request, tx *sql.Tx, sectionID, id string, forUpdate bool) (Assessment, error) {
	if !tenant.IsUUID(id) {
		return Assessment{}, httpx.NotFound("assessment")
	}
	q := `select ` + assessmentCols + ` from assessments where id = $1 and section_id = $2`
	if forUpdate {
		q += ` for update`
	}
	a, err := scanAssessment(tx.QueryRowContext(r.Context(), q, id, sectionID))
	if errors.Is(err, sql.ErrNoRows) {
		return a, httpx.NotFound("assessment")
	}
	return a, err
}

// teach: admin, dept_admin of the section's department, instructor of the section (spec 4 section 4).
func teach(r *http.Request, tx *sql.Tx, s authz.Scope, sectionID string) error {
	return allow(authz.CanManageTAs(r.Context(), tx, s, sectionID))
}

// grade: teach plus TA of the section.
func grade(r *http.Request, tx *sql.Tx, s authz.Scope, sectionID string) error {
	return allow(authz.CanReadRoster(r.Context(), tx, s, sectionID))
}
