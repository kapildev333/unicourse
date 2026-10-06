package course

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/audit"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

// Course is the contract resource; createdAt only feeds the list cursor.
type Course struct {
	ID           string `json:"id"`
	DepartmentID string `json:"departmentId"`
	Code         string `json:"code"`
	Title        string `json:"title"`
	createdAt    time.Time
}

// courseFields validates departmentId, code and title (all errors at once, body field order).
func courseFields(c Course) (Course, error) {
	var details []httpx.FieldError
	if c.DepartmentID == "" {
		details = append(details, httpx.FieldError{Field: "departmentId", Message: "is required"})
	} else if !tenant.IsUUID(c.DepartmentID) {
		details = append(details, invalidUUID("departmentId"))
	}
	var fe *httpx.FieldError
	if c.Code, fe = tenant.TrimLen("code", c.Code, 20); fe != nil {
		details = append(details, *fe)
	}
	if c.Title, fe = tenant.TrimLen("title", c.Title, 200); fe != nil {
		details = append(details, *fe)
	}
	if len(details) > 0 {
		return c, httpx.Validation(details...)
	}
	return c, nil
}

func courseWriteErr(err error) error {
	switch err = pg.MapError(err); {
	case errors.Is(err, pg.ErrConflict):
		return httpx.Conflict("course code already exists")
	case errors.Is(err, pg.ErrNotFound):
		return httpx.NotFound("department")
	}
	return err
}

func loadCourse(r *http.Request, tx *sql.Tx, id string) (Course, error) {
	var c Course
	if !tenant.IsUUID(id) {
		return c, httpx.NotFound("course")
	}
	err := tx.QueryRowContext(r.Context(), `select id, department_id, code, title, created_at from courses where id = $1 for update`, id).
		Scan(&c.ID, &c.DepartmentID, &c.Code, &c.Title, &c.createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return c, httpx.NotFound("course")
	}
	return c, err
}

func listCourses(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	if plainMember(s) {
		return 0, nil, httpx.Forbidden()
	}
	p, after, afterID, err := tenant.Page(r)
	if err != nil {
		return 0, nil, err
	}
	var dept any
	if v := r.URL.Query().Get("departmentId"); v != "" {
		if !tenant.IsUUID(v) {
			return 0, nil, httpx.Validation(invalidUUID("departmentId"))
		}
		dept = v
	}
	rows, err := tx.QueryContext(r.Context(), `select id, department_id, code, title, created_at from courses
		where ($4::uuid is null or department_id = $4)
		and ($1::timestamptz is null or (created_at, id) > ($1, $2::uuid))
		order by created_at, id limit $3`, after, afterID, p.Limit+1, dept)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	out := httpx.List[Course]{Items: []Course{}}
	var last Course
	for rows.Next() {
		var c Course
		if err := rows.Scan(&c.ID, &c.DepartmentID, &c.Code, &c.Title, &c.createdAt); err != nil {
			return 0, nil, err
		}
		if len(out.Items) == p.Limit {
			out.NextCursor = httpx.NextCursor(last.createdAt.UTC(), last.ID)
			break
		}
		out.Items = append(out.Items, c)
		last = c
	}
	if err := rows.Err(); err != nil {
		return 0, nil, err
	}
	return http.StatusOK, out, nil
}

func createCourse(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	if plainMember(s) {
		return 0, nil, httpx.Forbidden()
	}
	var in struct{ DepartmentID, Code, Title string }
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	c, err := courseFields(Course{DepartmentID: in.DepartmentID, Code: in.Code, Title: in.Title})
	if err != nil {
		return 0, nil, err
	}
	if !authz.CanManageDepartment(s, c.DepartmentID) {
		return 0, nil, httpx.Forbidden()
	}
	if err := tx.QueryRowContext(r.Context(), `insert into courses (university_id, department_id, code, title) values ($1,$2,$3,$4) returning id, created_at`,
		s.UniversityID, c.DepartmentID, c.Code, c.Title).Scan(&c.ID, &c.createdAt); err != nil {
		return 0, nil, courseWriteErr(err)
	}
	if err := audit.Write(r.Context(), tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "course.create", EntityType: "course", EntityID: c.ID, After: c}); err != nil {
		return 0, nil, err
	}
	return http.StatusCreated, c, nil
}

func updateCourse(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	before, err := loadCourse(r, tx, r.PathValue("id"))
	if err != nil {
		return 0, nil, err
	}
	if !authz.CanManageDepartment(s, before.DepartmentID) {
		return 0, nil, httpx.Forbidden()
	}
	var in struct {
		DepartmentID, Code, Title *string
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	if in.DepartmentID == nil && in.Code == nil && in.Title == nil {
		return 0, nil, tenant.NoFields()
	}
	after := before
	if in.DepartmentID != nil {
		after.DepartmentID = *in.DepartmentID
	}
	if in.Code != nil {
		after.Code = *in.Code
	}
	if in.Title != nil {
		after.Title = *in.Title
	}
	if after, err = courseFields(after); err != nil {
		return 0, nil, err
	}
	if !authz.CanManageDepartment(s, after.DepartmentID) {
		return 0, nil, httpx.Forbidden()
	}
	if _, err := tx.ExecContext(r.Context(), `update courses set department_id = $2, code = $3, title = $4 where id = $1`,
		after.ID, after.DepartmentID, after.Code, after.Title); err != nil {
		return 0, nil, courseWriteErr(err)
	}
	if err := audit.Write(r.Context(), tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "course.update", EntityType: "course", EntityID: after.ID, Before: before, After: after}); err != nil {
		return 0, nil, err
	}
	return http.StatusOK, after, nil
}

func deleteCourse(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	before, err := loadCourse(r, tx, r.PathValue("id"))
	if err != nil {
		return 0, nil, err
	}
	if !authz.CanManageDepartment(s, before.DepartmentID) {
		return 0, nil, httpx.Forbidden()
	}
	if _, err := tx.ExecContext(r.Context(), `delete from courses where id = $1`, before.ID); err != nil {
		if err = pg.MapDeleteError(err); errors.Is(err, pg.ErrConflict) {
			return 0, nil, httpx.Conflict("course still has sections")
		}
		return 0, nil, err
	}
	if err := audit.Write(r.Context(), tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "course.delete", EntityType: "course", EntityID: before.ID, Before: before}); err != nil {
		return 0, nil, err
	}
	return http.StatusNoContent, nil, nil
}
