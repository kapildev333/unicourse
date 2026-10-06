package tenant

import (
	"database/sql"
	"errors"
	"net/http"
	"regexp"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/audit"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
)

// Department is the contract resource; createdAt only feeds the list cursor.
type Department struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	createdAt time.Time
}

var deptCodeRe = regexp.MustCompile(`^[A-Z0-9]{2,10}$`)

// deptFields validates name and code (all errors at once, body field order).
func deptFields(name, code string) (string, error) {
	var details []httpx.FieldError
	name, fe := trimLen("name", name, 100)
	if fe != nil {
		details = append(details, *fe)
	}
	if code == "" {
		details = append(details, httpx.FieldError{Field: "code", Message: "is required"})
	} else if !deptCodeRe.MatchString(code) {
		details = append(details, httpx.FieldError{Field: "code", Message: "must be 2 to 10 uppercase letters or digits"})
	}
	if len(details) > 0 {
		return "", httpx.Validation(details...)
	}
	return name, nil
}

func deptWriteErr(err error) error {
	if err = pg.MapError(err); errors.Is(err, pg.ErrConflict) {
		return httpx.Conflict("department code already exists")
	}
	return err
}

func listDepartments(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	p, after, afterID, err := page(r)
	if err != nil {
		return 0, nil, err
	}
	rows, err := tx.QueryContext(r.Context(), `select id, name, code, created_at from departments
		where ($1::timestamptz is null or (created_at, id) > ($1, $2::uuid))
		order by created_at, id limit $3`, after, afterID, p.Limit+1)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	out := httpx.List[Department]{Items: []Department{}}
	var last Department
	for rows.Next() {
		var d Department
		if err := rows.Scan(&d.ID, &d.Name, &d.Code, &d.createdAt); err != nil {
			return 0, nil, err
		}
		if len(out.Items) == p.Limit {
			out.NextCursor = httpx.NextCursor(last.createdAt.UTC(), last.ID)
			break
		}
		out.Items = append(out.Items, d)
		last = d
	}
	if err := rows.Err(); err != nil {
		return 0, nil, err
	}
	return http.StatusOK, out, nil
}

func createDepartment(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	if !authz.IsAdmin(s) {
		return 0, nil, httpx.Forbidden()
	}
	var in struct{ Name, Code string }
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	name, err := deptFields(in.Name, in.Code)
	if err != nil {
		return 0, nil, err
	}
	d := Department{Name: name, Code: in.Code}
	if err := tx.QueryRowContext(r.Context(), `insert into departments (university_id, name, code) values ($1,$2,$3) returning id, created_at`,
		s.UniversityID, d.Name, d.Code).Scan(&d.ID, &d.createdAt); err != nil {
		return 0, nil, deptWriteErr(err)
	}
	if err := audit.Write(r.Context(), tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "department.create", EntityType: "department", EntityID: d.ID, After: d}); err != nil {
		return 0, nil, err
	}
	return http.StatusCreated, d, nil
}

func updateDepartment(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	if !authz.IsAdmin(s) {
		return 0, nil, httpx.Forbidden()
	}
	id := r.PathValue("id")
	if !isUUID(id) {
		return 0, nil, httpx.NotFound("department")
	}
	var in struct{ Name, Code *string }
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	if in.Name == nil && in.Code == nil {
		return 0, nil, noFields()
	}
	var before Department
	err := tx.QueryRowContext(r.Context(), `select id, name, code, created_at from departments where id = $1 for update`, id).
		Scan(&before.ID, &before.Name, &before.Code, &before.createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil, httpx.NotFound("department")
	} else if err != nil {
		return 0, nil, err
	}
	after := before
	if in.Name != nil {
		after.Name = *in.Name
	}
	if in.Code != nil {
		after.Code = *in.Code
	}
	name, err := deptFields(after.Name, after.Code)
	if err != nil {
		return 0, nil, err
	}
	after.Name = name
	if _, err := tx.ExecContext(r.Context(), `update departments set name = $2, code = $3 where id = $1`, id, after.Name, after.Code); err != nil {
		return 0, nil, deptWriteErr(err)
	}
	if err := audit.Write(r.Context(), tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "department.update", EntityType: "department", EntityID: id, Before: before, After: after}); err != nil {
		return 0, nil, err
	}
	return http.StatusOK, after, nil
}

func deleteDepartment(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	if !authz.IsAdmin(s) {
		return 0, nil, httpx.Forbidden()
	}
	id := r.PathValue("id")
	if !isUUID(id) {
		return 0, nil, httpx.NotFound("department")
	}
	var before Department
	err := tx.QueryRowContext(r.Context(), `delete from departments where id = $1 returning id, name, code`, id).Scan(&before.ID, &before.Name, &before.Code)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil, httpx.NotFound("department")
	}
	if err != nil {
		if err = pg.MapDeleteError(err); errors.Is(err, pg.ErrConflict) {
			return 0, nil, httpx.Conflict("department is still in use")
		}
		return 0, nil, err
	}
	if _, err := tx.ExecContext(r.Context(), `update invitations set department_ids = array_remove(department_ids, $1::uuid)
		where accepted_at is null and $1::uuid = any(department_ids)`, id); err != nil {
		return 0, nil, err
	}
	if err := audit.Write(r.Context(), tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "department.delete", EntityType: "department", EntityID: id, Before: before}); err != nil {
		return 0, nil, err
	}
	return http.StatusNoContent, nil, nil
}
