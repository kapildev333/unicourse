package course

import (
	"database/sql"
	"encoding/json"
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
	InvitationID string `json:"invitationId"`
	Email        string `json:"email"`
	Role         string `json:"role"`
}

// Section always embeds its course, term and (pending) staff; createdAt only feeds the list cursor.
type Section struct {
	ID           string         `json:"id"`
	CourseID     string         `json:"courseId"`
	TermID       string         `json:"termId"`
	Number       string         `json:"number"`
	Description  string         `json:"description"`
	Course       Course         `json:"course"`
	Term         tenant.Term    `json:"term"`
	Staff        []Staff        `json:"staff"`
	PendingStaff []PendingStaff `json:"pendingStaff"`
	Meetings     []Meeting      `json:"meetings"`
	createdAt    time.Time
}

type sectionAudit struct {
	ID       string         `json:"id"`
	CourseID string         `json:"courseId"`
	TermID   string         `json:"termId"`
	Number   string         `json:"number"`
	Meetings []MeetingInput `json:"meetings"`
}

// sectionCols and sectionFrom are the one select used wherever a Section is read (staff and
// pending staff aggregated per row by correlated subqueries on indexed section_id: no N+1).
const sectionCols = `s.id, s.course_id, s.term_id, s.number, s.description, s.created_at,
	c.id, c.department_id, c.code, c.title, c.credits::text,
	t.id, t.name, to_char(t.starts_on, 'YYYY-MM-DD'), to_char(t.ends_on, 'YYYY-MM-DD'),
	coalesce((select json_agg(json_build_object('userId', u.id, 'email', u.email, 'displayName', u.display_name, 'role', ss.role)
		order by ss.role <> 'instructor', u.email)
		from section_staff ss join users u on u.id = ss.user_id where ss.section_id = s.id), '[]'),
	coalesce((select json_agg(json_build_object('invitationId', i.id, 'email', i.email, 'role', p.role) order by p.role <> 'instructor', i.email)
		from pending_section_staff p join invitations i on i.id = p.invitation_id
		where p.section_id = s.id and i.accepted_at is null and i.expires_at > now()), '[]'),
	` + meetingsJSON

const sectionFrom = ` from sections s join courses c on c.id = s.course_id join terms t on t.id = s.term_id`

func sectionFields(courseID, termID, number string, create bool) (string, []httpx.FieldError) {
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
	return number, details
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

// ReadSection is readSection for package content (PUT description answers with the Section).
func ReadSection(r *http.Request, tx *sql.Tx, id string) (Section, error) {
	return readSection(r, tx, id)
}

// loadForWrite locks the section and returns its audit record and course department.
func loadForWrite(r *http.Request, tx *sql.Tx, id string) (a sectionAudit, dept string, err error) {
	if !tenant.IsUUID(id) {
		return a, "", httpx.NotFound("section")
	}
	var meetings []byte
	err = tx.QueryRowContext(r.Context(), `select s.id, s.course_id, s.term_id, s.number, c.department_id, `+meetingsJSON+`
		from sections s join courses c on c.id = s.course_id where s.id = $1 for update of s`, id).
		Scan(&a.ID, &a.CourseID, &a.TermID, &a.Number, &dept, &meetings)
	if errors.Is(err, sql.ErrNoRows) {
		return a, "", httpx.NotFound("section")
	} else if err != nil {
		return a, "", err
	}
	a.Meetings = []MeetingInput{}
	err = json.Unmarshal(meetings, &a.Meetings)
	return
}

func listSections(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	p, after, afterID, err := tenant.Page(r)
	if err != nil {
		return 0, nil, err
	}
	var term, course, dept any
	var details []httpx.FieldError
	for _, f := range []struct {
		name string
		dst  *any
	}{{"termId", &term}, {"courseId", &course}, {"departmentId", &dept}} {
		if v := r.URL.Query().Get(f.name); v != "" {
			if tenant.IsUUID(v) {
				*f.dst = v
			} else {
				details = append(details, invalidUUID(f.name))
			}
		}
	}
	q, fe := likeParam(r)
	if fe != nil {
		details = append(details, *fe)
	}
	if len(details) > 0 {
		return 0, nil, httpx.Validation(details...)
	}
	rows, err := tx.QueryContext(r.Context(), `select `+sectionCols+sectionFrom+`
		where ($4::uuid is null or s.term_id = $4) and ($5::uuid is null or s.course_id = $5)
		and ($6::uuid is null or c.department_id = $6) and ($7::text is null or c.code ilike $7 or c.title ilike $7)
		and ($1::timestamptz is null or (s.created_at, s.id) > ($1, $2::uuid))
		order by s.created_at, s.id limit $3`, after, afterID, p.Limit+1, term, course, dept, q)
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
		if plainMember(s) {
			sec.PendingStaff = []PendingStaff{} // D41: pending invitees are for managers
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
	// design §14: any admin or dept_admin of the university may read one section with pendingStaff
	if s.Role != authz.Member {
		return http.StatusOK, sec, nil
	}
	// D41: every member reads the section; only its staff see pending invitees
	if ok, err := authz.CanReadRoster(r.Context(), tx, s, sec.ID); err != nil {
		return 0, nil, err
	} else if !ok {
		sec.PendingStaff = []PendingStaff{}
	}
	return http.StatusOK, sec, nil
}

func createSection(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	if plainMember(s) {
		return 0, nil, httpx.Forbidden()
	}
	var in struct {
		CourseID string         `json:"courseId"`
		TermID   string         `json:"termId"`
		Number   string         `json:"number"`
		Meetings []MeetingInput `json:"meetings"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	number, d := sectionFields(in.CourseID, in.TermID, in.Number, true)
	ms, md := validateMeetings(in.Meetings)
	if d = append(d, md...); len(d) > 0 {
		return 0, nil, httpx.Validation(d...)
	}
	var dept string
	err := tx.QueryRowContext(r.Context(), `select department_id from courses where id = $1`, in.CourseID).Scan(&dept)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil, httpx.NotFound("course")
	} else if err != nil {
		return 0, nil, err
	}
	if !authz.CanManageDepartment(s, dept) {
		return 0, nil, httpx.Forbidden()
	}
	a := sectionAudit{CourseID: in.CourseID, TermID: in.TermID, Number: number, Meetings: ms}
	if err := tx.QueryRowContext(r.Context(), `insert into sections (university_id, course_id, term_id, number) values ($1,$2,$3,$4) returning id`,
		s.UniversityID, a.CourseID, a.TermID, a.Number).Scan(&a.ID); err != nil {
		return 0, nil, sectionWriteErr(err)
	}
	if err := replaceMeetings(r.Context(), tx, s.UniversityID, a.ID, ms); err != nil {
		return 0, nil, err
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
	var in struct {
		Number   tenant.OptStr   `json:"number"`
		Meetings *[]MeetingInput `json:"meetings"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	if err := tenant.PatchNulls(tenant.Named("number", in.Number)); err != nil {
		return 0, nil, err
	}
	if !in.Number.Set && in.Meetings == nil {
		return 0, nil, tenant.NoFields()
	}
	after := before
	var d, md []httpx.FieldError
	if in.Number.Set {
		after.Number, d = sectionFields("", "", in.Number.V, false)
	}
	if in.Meetings != nil {
		after.Meetings, md = validateMeetings(*in.Meetings)
	}
	if d = append(d, md...); len(d) > 0 {
		return 0, nil, httpx.Validation(d...)
	}
	if in.Number.Set {
		if _, err := tx.ExecContext(r.Context(), `update sections set number = $2 where id = $1`, after.ID, after.Number); err != nil {
			return 0, nil, sectionWriteErr(err)
		}
	}
	if in.Meetings != nil {
		if err := replaceMeetings(r.Context(), tx, s.UniversityID, after.ID, after.Meetings); err != nil {
			return 0, nil, err
		}
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
		if err = pg.MapDeleteError(err); errors.Is(err, pg.ErrConflict) {
			return 0, nil, httpx.Conflict("section has final grades")
		}
		return 0, nil, err
	}
	if err := audit.Write(r.Context(), tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "section.delete", EntityType: "section", EntityID: before.ID, Before: before}); err != nil {
		return 0, nil, err
	}
	return http.StatusNoContent, nil, nil
}
