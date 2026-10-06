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

type Staff struct {
	UserID      string `json:"userId"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Role        string `json:"role"`
}

type PendingStaff struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

// Section always embeds its course, term and (pending) staff; createdAt only feeds the list cursor.
type Section struct {
	ID           string         `json:"id"`
	CourseID     string         `json:"courseId"`
	TermID       string         `json:"termId"`
	Number       string         `json:"number"`
	Course       Course         `json:"course"`
	Term         tenant.Term    `json:"term"`
	Staff        []Staff        `json:"staff"`
	PendingStaff []PendingStaff `json:"pendingStaff"`
	createdAt    time.Time
}

type sectionAudit struct {
	ID       string `json:"id"`
	CourseID string `json:"courseId"`
	TermID   string `json:"termId"`
	Number   string `json:"number"`
}

// sectionCols and sectionFrom are the one select used wherever a Section is read (staff and
// pending staff aggregated per row by correlated subqueries on indexed section_id: no N+1).
const sectionCols = `s.id, s.course_id, s.term_id, s.number, s.created_at,
	c.id, c.department_id, c.code, c.title,
	t.id, t.name, to_char(t.starts_on, 'YYYY-MM-DD'), to_char(t.ends_on, 'YYYY-MM-DD'),
	coalesce((select json_agg(json_build_object('userId', u.id, 'email', u.email, 'displayName', u.display_name, 'role', ss.role)
		order by ss.role <> 'instructor', u.email)
		from section_staff ss join users u on u.id = ss.user_id where ss.section_id = s.id), '[]'),
	coalesce((select json_agg(json_build_object('email', i.email, 'role', p.role) order by p.role <> 'instructor', i.email)
		from pending_section_staff p join invitations i on i.id = p.invitation_id
		where p.section_id = s.id and i.accepted_at is null and i.expires_at > now()), '[]')`

const sectionFrom = ` from sections s join courses c on c.id = s.course_id join terms t on t.id = s.term_id`

func sectionFields(courseID, termID, number string, create bool) (string, error) {
	var details []httpx.FieldError
	if create {
		for _, f := range [][2]string{{"courseId", courseID}, {"termId", termID}} {
			if f[1] == "" {
				details = append(details, httpx.FieldError{Field: f[0], Message: "is required"})
			} else if !tenant.IsUUID(f[1]) {
				details = append(details, invalidUUID(f[0]))
			}
		}
	}
	number, fe := tenant.TrimLen("number", number, 20)
	if fe != nil {
		details = append(details, *fe)
	}
	if len(details) > 0 {
		return "", httpx.Validation(details...)
	}
	return number, nil
}

func sectionWriteErr(err error) error {
	switch err = pg.MapError(err); {
	case errors.Is(err, pg.ErrConflict):
		return httpx.Conflict("section number already exists for this course and term")
	case errors.Is(err, pg.ErrNotFound):
		return httpx.NotFound("term")
	}
	return err
}

func readSection(r *http.Request, tx *sql.Tx, id string) (Section, error) {
	if !tenant.IsUUID(id) {
		return Section{}, httpx.NotFound("section")
	}
	sec, err := scanSection(tx.QueryRowContext(r.Context(), `select `+sectionCols+sectionFrom+` where s.id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return sec, httpx.NotFound("section")
	}
	return sec, err
}

// loadForWrite locks the section and returns its audit record and course department.
func loadForWrite(r *http.Request, tx *sql.Tx, id string) (a sectionAudit, dept string, err error) {
	if !tenant.IsUUID(id) {
		return a, "", httpx.NotFound("section")
	}
	err = tx.QueryRowContext(r.Context(), `select s.id, s.course_id, s.term_id, s.number, c.department_id
		from sections s join courses c on c.id = s.course_id where s.id = $1 for update of s`, id).
		Scan(&a.ID, &a.CourseID, &a.TermID, &a.Number, &dept)
	if errors.Is(err, sql.ErrNoRows) {
		err = httpx.NotFound("section")
	}
	return
}

func listSections(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	if plainMember(s) {
		return 0, nil, httpx.Forbidden()
	}
	p, after, afterID, err := tenant.Page(r)
	if err != nil {
		return 0, nil, err
	}
	var term, course any
	var details []httpx.FieldError
	for _, f := range []struct {
		name string
		dst  *any
	}{{"termId", &term}, {"courseId", &course}} {
		if v := r.URL.Query().Get(f.name); v != "" {
			if tenant.IsUUID(v) {
				*f.dst = v
			} else {
				details = append(details, invalidUUID(f.name))
			}
		}
	}
	if len(details) > 0 {
		return 0, nil, httpx.Validation(details...)
	}
	rows, err := tx.QueryContext(r.Context(), `select `+sectionCols+sectionFrom+`
		where ($4::uuid is null or s.term_id = $4) and ($5::uuid is null or s.course_id = $5)
		and ($1::timestamptz is null or (s.created_at, s.id) > ($1, $2::uuid))
		order by s.created_at, s.id limit $3`, after, afterID, p.Limit+1, term, course)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	out := httpx.List[Section]{Items: []Section{}}
	var last Section
	for rows.Next() {
		sec, err := scanSection(rows)
		if err != nil {
			return 0, nil, err
		}
		if len(out.Items) == p.Limit {
			out.NextCursor = httpx.NextCursor(last.createdAt.UTC(), last.ID)
			break
		}
		out.Items = append(out.Items, sec)
		last = sec
	}
	if err := rows.Err(); err != nil {
		return 0, nil, err
	}
	return http.StatusOK, out, nil
}

func getSection(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	sec, err := readSection(r, tx, r.PathValue("id"))
	if err != nil {
		return 0, nil, err
	}
	if ok, err := authz.CanReadSection(r.Context(), tx, s, sec.ID); err != nil {
		return 0, nil, err
	} else if !ok {
		return 0, nil, httpx.Forbidden()
	}
	return http.StatusOK, sec, nil
}

func createSection(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	if plainMember(s) {
		return 0, nil, httpx.Forbidden()
	}
	var in struct{ CourseID, TermID, Number string }
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	number, err := sectionFields(in.CourseID, in.TermID, in.Number, true)
	if err != nil {
		return 0, nil, err
	}
	var dept string
	err = tx.QueryRowContext(r.Context(), `select department_id from courses where id = $1`, in.CourseID).Scan(&dept)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil, httpx.NotFound("course")
	} else if err != nil {
		return 0, nil, err
	}
	if !authz.CanManageDepartment(s, dept) {
		return 0, nil, httpx.Forbidden()
	}
	a := sectionAudit{CourseID: in.CourseID, TermID: in.TermID, Number: number}
	if err := tx.QueryRowContext(r.Context(), `insert into sections (university_id, course_id, term_id, number) values ($1,$2,$3,$4) returning id`,
		s.UniversityID, a.CourseID, a.TermID, a.Number).Scan(&a.ID); err != nil {
		return 0, nil, sectionWriteErr(err)
	}
	if err := audit.Write(r.Context(), tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "section.create", EntityType: "section", EntityID: a.ID, After: a}); err != nil {
		return 0, nil, err
	}
	sec, err := readSection(r, tx, a.ID)
	return http.StatusCreated, sec, err
}

func updateSection(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	before, dept, err := loadForWrite(r, tx, r.PathValue("id"))
	if err != nil {
		return 0, nil, err
	}
	if !authz.CanManageDepartment(s, dept) {
		return 0, nil, httpx.Forbidden()
	}
	var in struct{ Number *string }
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	if in.Number == nil {
		return 0, nil, tenant.NoFields()
	}
	after := before
	if after.Number, err = sectionFields("", "", *in.Number, false); err != nil {
		return 0, nil, err
	}
	if _, err := tx.ExecContext(r.Context(), `update sections set number = $2 where id = $1`, after.ID, after.Number); err != nil {
		return 0, nil, sectionWriteErr(err)
	}
	if err := audit.Write(r.Context(), tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "section.update", EntityType: "section", EntityID: after.ID, Before: before, After: after}); err != nil {
		return 0, nil, err
	}
	sec, err := readSection(r, tx, after.ID)
	return http.StatusOK, sec, err
}

func deleteSection(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	before, dept, err := loadForWrite(r, tx, r.PathValue("id"))
	if err != nil {
		return 0, nil, err
	}
	if !authz.CanManageDepartment(s, dept) {
		return 0, nil, httpx.Forbidden()
	}
	if _, err := tx.ExecContext(r.Context(), `delete from sections where id = $1`, before.ID); err != nil {
		return 0, nil, err
	}
	if err := audit.Write(r.Context(), tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "section.delete", EntityType: "section", EntityID: before.ID, Before: before}); err != nil {
		return 0, nil, err
	}
	return http.StatusNoContent, nil, nil
}
