package course

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kapildev333/unicourse/apps/api/internal/audit"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

// Course is the contract resource; createdAt only feeds the list cursor.
type Course struct {
	ID           string      `json:"id"`
	DepartmentID string      `json:"departmentId"`
	Code         string      `json:"code"`
	Title        string      `json:"title"`
	Credits      json.Number `json:"credits"`
	createdAt    time.Time
}

var likeEsc = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// likeParam reads q: nil when blank after trim; else "%"+escaped+"%".
func likeParam(r *http.Request) (any, *httpx.FieldError) {
	v := strings.TrimSpace(r.URL.Query().Get("q"))
	switch {
	case v == "":
		return nil, nil
	case !utf8.ValidString(v):
		return nil, &httpx.FieldError{Field: "q", Message: "must be valid UTF-8"}
	case strings.ContainsRune(v, 0):
		return nil, &httpx.FieldError{Field: "q", Message: "must not contain NUL"}
	case utf8.RuneCountInString(v) > 100:
		return nil, &httpx.FieldError{Field: "q", Message: "must be at most 100 characters"}
	}
	return "%" + likeEsc.Replace(v) + "%", nil
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
	cr, fe := tenant.Decimal("credits", c.Credits, 1, "0", "20")
	if fe != nil {
		details = append(details, *fe)
	}
	c.Credits = json.Number(cr)
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
	err := tx.QueryRowContext(r.Context(), `select id, department_id, code, title, credits::text, created_at from courses where id = $1 for update`, id).
		Scan(&c.ID, &c.DepartmentID, &c.Code, &c.Title, &c.Credits, &c.createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return c, httpx.NotFound("course")
	}
	return c, err
}

func listCourses(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	p, _, afterID, err := tenant.Page(r)
	if err != nil {
		return 0, nil, err
	}
	var dept any
	var details []httpx.FieldError
	if v := r.URL.Query().Get("departmentId"); v != "" {
		if tenant.IsUUID(v) {
			dept = v
		} else {
			details = append(details, invalidUUID("departmentId"))
		}
	}
	q, fe := likeParam(r)
	if fe != nil {
		details = append(details, *fe)
	}
	if len(details) > 0 {
		return 0, nil, httpx.Validation(details...)
	}
	// ponytail: cursor names the last row; a deleted or re-keyed row ends or shifts the next page; encode the sort key in the cursor if that bites
	rows, err := tx.QueryContext(r.Context(), `select id, department_id, code, title, credits::text, created_at from courses
		where ($3::uuid is null or department_id = $3) and ($4::text is null or code ilike $4 or title ilike $4)
		and ($1::uuid is null or (code, id) > (select code, id from courses where id = $1))
		order by code, id limit $2`, afterID, p.Limit+1, dept, q)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	out := httpx.List[Course]{Items: []Course{}}
	var last Course
	for rows.Next() {
		var c Course
		if err := rows.Scan(&c.ID, &c.DepartmentID, &c.Code, &c.Title, &c.Credits, &c.createdAt); err != nil {
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
	var in struct {
		DepartmentID string       `json:"departmentId"`
		Code         string       `json:"code"`
		Title        string       `json:"title"`
		Credits      *json.Number `json:"credits"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	credits := json.Number("3.0")
	if in.Credits != nil {
		credits = *in.Credits
	}
	c, err := courseFields(Course{DepartmentID: in.DepartmentID, Code: in.Code, Title: in.Title, Credits: credits})
	if err != nil {
		return 0, nil, err
	}
	if !authz.CanManageDepartment(s, c.DepartmentID) {
		return 0, nil, httpx.Forbidden()
	}
	if err := tx.QueryRowContext(r.Context(), `insert into courses (university_id, department_id, code, title, credits) values ($1,$2,$3,$4,$5::numeric) returning id, credits::text, created_at`,
		s.UniversityID, c.DepartmentID, c.Code, c.Title, c.Credits).Scan(&c.ID, &c.Credits, &c.createdAt); err != nil {
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
		DepartmentID tenant.OptStr           `json:"departmentId"`
		Code         tenant.OptStr           `json:"code"`
		Title        tenant.OptStr           `json:"title"`
		Credits      tenant.Opt[json.Number] `json:"credits"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	nullErr := tenant.PatchNulls(tenant.Named("departmentId", in.DepartmentID), tenant.Named("code", in.Code), tenant.Named("title", in.Title))
	if in.Credits.Null {
		var d []httpx.FieldError
		if ve, ok := nullErr.(*httpx.Error); ok {
			d = ve.Details
		}
		nullErr = httpx.Validation(append(d, httpx.FieldError{Field: "credits", Message: "must be a number"})...)
	}
	if nullErr != nil {
		return 0, nil, nullErr
	}
	if !in.DepartmentID.Set && !in.Code.Set && !in.Title.Set && !in.Credits.Set {
		return 0, nil, tenant.NoFields()
	}
	after := before
	if in.DepartmentID.Set {
		after.DepartmentID = in.DepartmentID.V
	}
	if in.Code.Set {
		after.Code = in.Code.V
	}
	if in.Title.Set {
		after.Title = in.Title.V
	}
	if in.Credits.Set {
		after.Credits = in.Credits.V
	}
	if after, err = courseFields(after); err != nil {
		return 0, nil, err
	}
	if !authz.CanManageDepartment(s, after.DepartmentID) {
		return 0, nil, httpx.Forbidden()
	}
	if _, err := tx.ExecContext(r.Context(), `update courses set department_id = $2, code = $3, title = $4, credits = $5::numeric where id = $1`,
		after.ID, after.DepartmentID, after.Code, after.Title, after.Credits); err != nil {
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
