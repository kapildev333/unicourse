package grade

import (
	"context"
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

type Assessment struct {
	ID            string      `json:"id"`
	SectionID     string      `json:"sectionId"`
	Kind          string      `json:"kind"`
	Name          string      `json:"name"`
	Description   string      `json:"description"`
	MaxPoints     json.Number `json:"maxPoints"`
	Weight        json.Number `json:"weight"`
	DueOn         string      `json:"dueOn,omitempty"`
	OpensAt       *time.Time  `json:"opensAt,omitempty"`  // UTC
	ClosesAt      *time.Time  `json:"closesAt,omitempty"` // UTC
	Published     bool        `json:"published"`
	Position      int         `json:"position"`
	QuestionCount *int        `json:"questionCount,omitempty"` // quizzes only
}

const assessmentCols = `id, section_id, kind, name, description, max_points::text, weight::text,
  coalesce(to_char(due_on, 'YYYY-MM-DD'), ''), opens_at, closes_at, published, position,
  case when kind = 'quiz' then (select count(*)::int from questions q where q.assessment_id = assessments.id) end`

func scanAssessment(sc interface{ Scan(...any) error }) (Assessment, error) {
	var a Assessment
	var opens, closes sql.NullTime
	var count sql.NullInt32
	err := sc.Scan(&a.ID, &a.SectionID, &a.Kind, &a.Name, &a.Description, &a.MaxPoints, &a.Weight, &a.DueOn, &opens, &closes, &a.Published, &a.Position, &count)
	if opens.Valid {
		t := opens.Time.UTC()
		a.OpensAt = &t
	}
	if closes.Valid {
		t := closes.Time.UTC()
		a.ClosesAt = &t
	}
	if count.Valid {
		n := int(count.Int32)
		a.QuestionCount = &n
	}
	return a, err
}

// listSectionAssessments returns the section's assessments by position, created_at, id. studentView hides
// unpublished manual assessments; quizzes and assignments stay visible (their window gates the work, Spec 11 section 4).
func listSectionAssessments(ctx context.Context, tx *sql.Tx, sectionID string, studentView bool) ([]Assessment, error) {
	rows, err := tx.QueryContext(ctx, `select `+assessmentCols+` from assessments where section_id = $1 and (published or kind <> 'manual' or not $2)
		order by position, created_at, id`, sectionID, studentView)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Assessment{}
	for rows.Next() {
		a, err := scanAssessment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

type assessmentList struct {
	Items []Assessment `json:"items"`
}

func assessmentWriteErr(err error) error {
	if err = pg.MapError(err); errors.Is(err, pg.ErrConflict) {
		return httpx.Conflict("an assessment with this name already exists in this section")
	}
	return err
}

// fieldsIn is the merged (PATCH) or given (create) input of assessmentFields. SetMax and SetDue say whether the
// caller named maxPoints / dueOn, which quizzes and assignments do not accept.
type fieldsIn struct {
	Kind, Name, Description  string
	MaxPoints, Weight        json.Number
	DueOn, OpensAt, ClosesAt string // times RFC 3339, "" = none
	Position                 *int
	SetMax, SetDue           bool
}

// assessmentFields validates the merged or created values (all errors at once, body field order)
// and returns them canonical (OpensAt and ClosesAt stay strings; DueOn of a quiz or assignment is derived in SQL).
// position is nil when not given (create).
func assessmentFields(in fieldsIn) (fieldsIn, []httpx.FieldError) {
	var d []httpx.FieldError
	bad := func(f, m string) { d = append(d, httpx.FieldError{Field: f, Message: m}) }
	out := in
	manual := in.Kind == "manual"
	known := manual || in.Kind == "quiz" || in.Kind == "assignment"
	if !known {
		bad("kind", "must be manual, quiz or assignment")
	}
	var fe *httpx.FieldError
	if out.Name, fe = tenant.TrimLen("name", in.Name, 100); fe != nil {
		d = append(d, *fe)
	}
	if out.Description, fe = descriptionText(in.Description); fe != nil {
		d = append(d, *fe)
	}
	if !known { // the kind-dependent fields cannot be judged
	} else if in.Kind == "quiz" {
		if in.SetMax {
			bad("maxPoints", "derived from the questions")
		}
	} else if mp, fe := tenant.Decimal("maxPoints", in.MaxPoints, 2, "0.01", "10000"); fe != nil {
		d = append(d, *fe)
	} else {
		out.MaxPoints = json.Number(mp)
	}
	w, fe := tenant.Decimal("weight", in.Weight, 2, "0", "100")
	if fe != nil {
		d = append(d, *fe)
	}
	out.Weight = json.Number(w)
	switch {
	case !known:
	case !manual && in.SetDue:
		bad("dueOn", "derived from closesAt")
	case manual && in.DueOn != "":
		if _, err := time.Parse("2006-01-02", in.DueOn); err != nil || len(in.DueOn) != 10 {
			bad("dueOn", "must be a date YYYY-MM-DD")
		}
	}
	var opens, closes time.Time
	var opensOK, closesOK bool
	for _, f := range []struct {
		name, v string
		t       *time.Time
		ok      *bool
	}{{"opensAt", in.OpensAt, &opens, &opensOK}, {"closesAt", in.ClosesAt, &closes, &closesOK}} {
		switch {
		case !known:
		case manual && f.v != "":
			bad(f.name, "only for quizzes and assignments")
		case manual:
		case f.v == "":
			bad(f.name, "is required")
		default:
			if t, err := time.Parse(time.RFC3339, f.v); err != nil {
				bad(f.name, "must be an RFC 3339 time")
			} else {
				*f.t, *f.ok = t, true
			}
		}
	}
	if opensOK && closesOK && !closes.After(opens) {
		bad("closesAt", "must be after opensAt")
	}
	if in.Position != nil {
		if *in.Position < 0 || *in.Position > 10000 {
			bad("position", "must be between 0 and 10000")
		}
	}
	return out, d
}

// descriptionText trims, allows line breaks, rejects NUL and more than 10000 characters (content.optText's rules).
func descriptionText(v string) (string, *httpx.FieldError) {
	v = strings.TrimSpace(v)
	if strings.ContainsRune(v, 0) {
		return v, &httpx.FieldError{Field: "description", Message: "must not contain NUL"}
	}
	if utf8.RuneCountInString(v) > 10000 {
		return v, &httpx.FieldError{Field: "description", Message: "must be at most 10000 characters"}
	}
	return v, nil
}

// editor: the role predicate of the kind (design P18-L1 section 0.1): manual keeps Spec 4's teach, the other kinds take TAs too.
func editor(r *http.Request, tx *sql.Tx, s authz.Scope, secID, kind string) error {
	if kind == "manual" {
		return teach(r, tx, s, secID)
	}
	return allow(authz.CanEditContent(r.Context(), tx, s, secID))
}

func rfc3339(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339Nano)
}

func listAssessments(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	id := r.PathValue("id")
	if err := sectionExists(r, tx, id); err != nil {
		return 0, nil, err
	}
	studentView := false
	staff, err := authz.CanReadRoster(r.Context(), tx, s, id) // grade
	if err != nil {
		return 0, nil, err
	}
	if !staff {
		if err := allow(authz.CanReadSection(r.Context(), tx, s, id)); err != nil { // enrolled student
			return 0, nil, err
		}
		studentView = true
	}
	items, err := listSectionAssessments(r.Context(), tx, id, studentView)
	return http.StatusOK, assessmentList{items}, err
}

func createAssessment(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	secID := r.PathValue("id")
	if err := sectionExists(r, tx, secID); err != nil {
		return 0, nil, err
	}
	if err := allow(authz.CanEditContent(r.Context(), tx, s, secID)); err != nil {
		return 0, nil, err
	}
	var in struct {
		Kind        string      `json:"kind"`
		Name        string      `json:"name"`
		Description string      `json:"description"`
		MaxPoints   json.Number `json:"maxPoints"`
		Weight      json.Number `json:"weight"`
		DueOn       string      `json:"dueOn"`
		OpensAt     string      `json:"opensAt"`
		ClosesAt    string      `json:"closesAt"`
		Published   bool        `json:"published"`
		Position    *int        `json:"position"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	if in.Kind == "" {
		in.Kind = "manual"
	}
	if in.Kind == "manual" { // an invalid kind skips the check and is a 400
		if err := teach(r, tx, s, secID); err != nil {
			return 0, nil, err
		}
	}
	f := fieldsIn{Kind: in.Kind, Name: in.Name, Description: in.Description, MaxPoints: in.MaxPoints, Weight: in.Weight, DueOn: in.DueOn,
		OpensAt: in.OpensAt, ClosesAt: in.ClosesAt, Position: in.Position, SetMax: in.MaxPoints != "", SetDue: in.DueOn != ""}
	if in.Kind == "quiz" {
		f.MaxPoints = "1" // Σ question points once questions exist
	}
	v, d := assessmentFields(f)
	if len(d) > 0 {
		return 0, nil, httpx.Validation(d...)
	}
	a, err := scanAssessment(tx.QueryRowContext(r.Context(), `insert into assessments (university_id, section_id, kind, name, description, max_points, weight, opens_at, closes_at, due_on, published, position)
		values ($1, $2, $3, $4, $5, $6::numeric, $7::numeric, nullif($8, '')::timestamptz, nullif($9, '')::timestamptz,
		        case when $3 = 'manual' then nullif($10, '')::date
		             else (nullif($9, '')::timestamptz at time zone (select timezone from universities where id = $1))::date end,
		        $11, coalesce($12::int, (select coalesce(max(position) + 1, 0) from assessments where section_id = $2)))
		returning `+assessmentCols, s.UniversityID, secID, v.Kind, v.Name, v.Description, v.MaxPoints, v.Weight, v.OpensAt, v.ClosesAt, v.DueOn, in.Published, in.Position))
	if err != nil {
		return 0, nil, assessmentWriteErr(err)
	}
	if err := audit.Write(r.Context(), tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "assessment.create", EntityType: "assessment", EntityID: a.ID, After: a}); err != nil {
		return 0, nil, err
	}
	return http.StatusCreated, a, nil
}

func updateAssessment(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	secID := r.PathValue("id")
	if err := sectionExists(r, tx, secID); err != nil {
		return 0, nil, err
	}
	before, err := loadAssessment(r, tx, secID, r.PathValue("assessmentId"), true)
	if err != nil {
		return 0, nil, err
	}
	if err := editor(r, tx, s, secID, before.Kind); err != nil {
		return 0, nil, err
	}
	var in struct {
		Name        tenant.Opt[string]      `json:"name"`
		Description tenant.Opt[string]      `json:"description"`
		MaxPoints   tenant.Opt[json.Number] `json:"maxPoints"`
		Weight      tenant.Opt[json.Number] `json:"weight"`
		DueOn       tenant.Opt[string]      `json:"dueOn"`
		OpensAt     tenant.Opt[string]      `json:"opensAt"`
		ClosesAt    tenant.Opt[string]      `json:"closesAt"`
		Published   tenant.Opt[bool]        `json:"published"`
		Position    tenant.Opt[int]         `json:"position"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	var d []httpx.FieldError
	for _, f := range []struct {
		name string
		null bool
	}{{"name", in.Name.Null}, {"description", in.Description.Null}, {"maxPoints", in.MaxPoints.Null}, {"weight", in.Weight.Null},
		{"opensAt", in.OpensAt.Null}, {"closesAt", in.ClosesAt.Null}, {"published", in.Published.Null}, {"position", in.Position.Null}} {
		if f.null {
			d = append(d, httpx.FieldError{Field: f.name, Message: "must not be null"})
		}
	}
	if len(d) > 0 {
		return 0, nil, httpx.Validation(d...)
	}
	if !in.Name.Set && !in.Description.Set && !in.MaxPoints.Set && !in.Weight.Set && !in.DueOn.Set && !in.OpensAt.Set && !in.ClosesAt.Set && !in.Published.Set && !in.Position.Set {
		return 0, nil, tenant.NoFields()
	}
	after := before
	f := fieldsIn{Kind: before.Kind, Name: before.Name, Description: before.Description, MaxPoints: before.MaxPoints, Weight: before.Weight,
		DueOn: before.DueOn, OpensAt: rfc3339(before.OpensAt), ClosesAt: rfc3339(before.ClosesAt), Position: &before.Position,
		SetMax: in.MaxPoints.Set, SetDue: in.DueOn.Set}
	emptyDue := false // "" is a 400 (null clears), collected with the other field errors
	if in.Name.Set {
		f.Name = in.Name.V
	}
	if in.Description.Set {
		f.Description = in.Description.V
	}
	if in.MaxPoints.Set {
		f.MaxPoints = in.MaxPoints.V
	}
	if in.Weight.Set {
		f.Weight = in.Weight.V
	}
	if in.DueOn.Set {
		f.DueOn = in.DueOn.V // null: V is "" and clears it
		emptyDue = !in.DueOn.Null && in.DueOn.V == ""
	}
	if in.OpensAt.Set {
		f.OpensAt = in.OpensAt.V
	}
	if in.ClosesAt.Set {
		f.ClosesAt = in.ClosesAt.V
	}
	if in.Published.Set {
		after.Published = in.Published.V
	}
	if in.Position.Set {
		pos := in.Position.V
		f.Position, after.Position = &pos, pos
	}
	// ponytail: a changed maxPoints does not re-check existing scores against 2 x max; validate on PATCH if it matters
	v, d := assessmentFields(f)
	if emptyDue && before.Kind == "manual" {
		d = append(d, httpx.FieldError{Field: "dueOn", Message: "must be a date YYYY-MM-DD"})
	}
	if len(d) > 0 {
		return 0, nil, httpx.Validation(d...)
	}
	after, err = scanAssessment(tx.QueryRowContext(r.Context(), `update assessments set name = $2, description = $3,
		max_points = case when kind = 'quiz' then max_points else $4::numeric end, weight = $5::numeric,
		opens_at = nullif($6, '')::timestamptz, closes_at = nullif($7, '')::timestamptz,
		due_on = case when kind = 'manual' then nullif($8, '')::date
		              else (nullif($7, '')::timestamptz at time zone (select u.timezone from universities u where u.id = assessments.university_id))::date end,
		published = $9, position = $10 where id = $1 returning `+assessmentCols,
		after.ID, v.Name, v.Description, v.MaxPoints, v.Weight, v.OpensAt, v.ClosesAt, v.DueOn, after.Published, after.Position))
	if err != nil {
		return 0, nil, assessmentWriteErr(err)
	}
	if err := audit.Write(r.Context(), tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "assessment.update", EntityType: "assessment", EntityID: after.ID, Before: before, After: after}); err != nil {
		return 0, nil, err
	}
	return http.StatusOK, after, nil
}

func deleteAssessment(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	secID := r.PathValue("id")
	if err := sectionExists(r, tx, secID); err != nil {
		return 0, nil, err
	}
	before, err := loadAssessment(r, tx, secID, r.PathValue("assessmentId"), true)
	if err != nil {
		return 0, nil, err
	}
	if err := editor(r, tx, s, secID, before.Kind); err != nil {
		return 0, nil, err
	}
	if _, err := tx.ExecContext(r.Context(), `delete from assessments where id = $1`, before.ID); err != nil {
		return 0, nil, err
	}
	if err := audit.Write(r.Context(), tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "assessment.delete", EntityType: "assessment", EntityID: before.ID, Before: before}); err != nil {
		return 0, nil, err
	}
	return http.StatusNoContent, nil, nil
}
