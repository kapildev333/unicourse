// Package course holds the catalog (courses) and timetable (sections) routes.
package course

import (
	"encoding/json"

	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

const base = "/api/v1/universities/{slug}"

// Register mounts the course and section routes, each through tenant.Wrap.
func Register(r *httpx.Router, d tenant.Deps) {
	r.Handle("GET", base+"/courses", tenant.Wrap(d, listCourses))
	r.Handle("POST", base+"/courses", tenant.Wrap(d, createCourse))
	r.Handle("PATCH", base+"/courses/{id}", tenant.Wrap(d, updateCourse))
	r.Handle("DELETE", base+"/courses/{id}", tenant.Wrap(d, deleteCourse))
	r.Handle("GET", base+"/sections", tenant.Wrap(d, listSections))
	r.Handle("POST", base+"/sections", tenant.Wrap(d, createSection))
	r.Handle("GET", base+"/sections/{id}", tenant.Wrap(d, getSection))
	r.Handle("PATCH", base+"/sections/{id}", tenant.Wrap(d, updateSection))
	r.Handle("DELETE", base+"/sections/{id}", tenant.Wrap(d, deleteSection))
	r.Handle("GET", base+"/sections/{id}/staff", tenant.Wrap(d, listStaff))
	r.Handle("POST", base+"/sections/{id}/staff", tenant.Wrap(d, createStaff(d)))
	r.Handle("DELETE", base+"/sections/{id}/staff/{userId}", tenant.Wrap(d, deleteStaff))
	r.Handle("DELETE", base+"/sections/{id}/pending-staff/{invitationId}", tenant.Wrap(d, deletePendingStaff))
	r.Handle("GET", base+"/sections/{id}/enrollments", tenant.Wrap(d, listEnrollments))
	r.Handle("DELETE", base+"/sections/{id}/enrollments/{userId}", tenant.Wrap(d, deleteEnrollment))
	for _, im := range []struct {
		path string
		mk   importer
	}{{"/courses/import", coursesImport}, {"/sections/import", sectionsImport}, {"/sections/{id}/enrollments/import", enrollmentsImport}} {
		r.Handle("POST", base+im.path+"/preview", tenant.Wrap(d, importHandler(d, false, im.mk)))
		r.Handle("POST", base+im.path, tenant.Wrap(d, importHandler(d, true, im.mk)))
	}
	r.Handle("GET", base+"/me/sections", tenant.Wrap(d, getMySections))
}

// plainMember is true for callers who may not use write routes (and see no pendingStaff in lists).
func plainMember(s authz.Scope) bool { return s.Role == authz.Member }

// scanSection scans one row of sectionCols (plus any extra trailing columns, e.g. my_role).
func scanSection(sc interface{ Scan(...any) error }, extra ...any) (Section, error) {
	var s Section
	var staff, pending, meetings []byte
	dest := []any{&s.ID, &s.CourseID, &s.TermID, &s.Number, &s.Description, &s.createdAt,
		&s.Course.ID, &s.Course.DepartmentID, &s.Course.Code, &s.Course.Title, &s.Course.Credits,
		&s.Term.ID, &s.Term.Name, &s.Term.StartsOn, &s.Term.EndsOn, &staff, &pending, &meetings}
	if err := sc.Scan(append(dest, extra...)...); err != nil {
		return s, err
	}
	s.Staff, s.PendingStaff, s.Meetings = []Staff{}, []PendingStaff{}, []Meeting{}
	if err := json.Unmarshal(staff, &s.Staff); err != nil {
		return s, err
	}
	if err := json.Unmarshal(pending, &s.PendingStaff); err != nil {
		return s, err
	}
	if err := json.Unmarshal(meetings, &s.Meetings); err != nil {
		return s, err
	}
	return s, nil
}

func invalidUUID(field string) httpx.FieldError {
	return httpx.FieldError{Field: field, Message: "must be a uuid"}
}
