package grade

import (
	"context"
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

type Assessment struct {
	ID        string      `json:"id"`
	SectionID string      `json:"sectionId"`
	Name      string      `json:"name"`
	MaxPoints json.Number `json:"maxPoints"`
	Weight    json.Number `json:"weight"`
	DueOn     string      `json:"dueOn,omitempty"`
	Published bool        `json:"published"`
	Position  int         `json:"position"`
}

const assessmentCols = `id, section_id, name, max_points::text, weight::text, coalesce(to_char(due_on, 'YYYY-MM-DD'), ''), published, position`

func scanAssessment(sc interface{ Scan(...any) error }) (Assessment, error) {
	var a Assessment
	err := sc.Scan(&a.ID, &a.SectionID, &a.Name, &a.MaxPoints, &a.Weight, &a.DueOn, &a.Published, &a.Position)
	return a, err
}

// listSectionAssessments returns the section's assessments by position, created_at, id; publishedOnly filters.
func listSectionAssessments(ctx context.Context, tx *sql.Tx, sectionID string, publishedOnly bool) ([]Assessment, error) {
	rows, err := tx.QueryContext(ctx, `select `+assessmentCols+` from assessments where section_id = $1 and (published or not $2)
		order by position, created_at, id`, sectionID, publishedOnly)
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

// assessmentFields validates the merged or created values (all errors at once, body field order)
// and returns them canonical. position is nil when not given (create).
func assessmentFields(name string, maxPoints, weight json.Number, dueOn string, position *int) (Assessment, []httpx.FieldError) {
	var a Assessment
	var d []httpx.FieldError
	var fe *httpx.FieldError
	if a.Name, fe = tenant.TrimLen("name", name, 100); fe != nil {
		d = append(d, *fe)
	}
	mp, fe := tenant.Decimal("maxPoints", maxPoints, 2, "0.01", "10000")
	if fe != nil {
		d = append(d, *fe)
	}
	w, fe := tenant.Decimal("weight", weight, 2, "0", "100")
	if fe != nil {
		d = append(d, *fe)
	}
	if dueOn != "" {
		if _, err := time.Parse("2006-01-02", dueOn); err != nil || len(dueOn) != 10 {
			d = append(d, httpx.FieldError{Field: "dueOn", Message: "must be a date YYYY-MM-DD"})
		}
	}
	if position != nil {
		if *position < 0 || *position > 10000 {
			d = append(d, httpx.FieldError{Field: "position", Message: "must be between 0 and 10000"})
		} else {
			a.Position = *position
		}
	}
	a.MaxPoints, a.Weight, a.DueOn = json.Number(mp), json.Number(w), dueOn
	return a, d
}

func listAssessments(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	id := r.PathValue("id")
	if err := sectionExists(r, tx, id); err != nil {
		return 0, nil, err
	}
	publishedOnly := false
	staff, err := authz.CanReadRoster(r.Context(), tx, s, id) // grade
	if err != nil {
		return 0, nil, err
	}
	if !staff {
		if err := allow(authz.CanReadSection(r.Context(), tx, s, id)); err != nil { // enrolled student
			return 0, nil, err
		}
		publishedOnly = true
	}
	items, err := listSectionAssessments(r.Context(), tx, id, publishedOnly)
	return http.StatusOK, assessmentList{items}, err
}

func createAssessment(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	secID := r.PathValue("id")
	if err := sectionExists(r, tx, secID); err != nil {
		return 0, nil, err
	}
	if err := teach(r, tx, s, secID); err != nil {
		return 0, nil, err
	}
	var in struct {
		Name      string      `json:"name"`
		MaxPoints json.Number `json:"maxPoints"`
		Weight    json.Number `json:"weight"`
		DueOn     string      `json:"dueOn"`
		Published bool        `json:"published"`
		Position  *int        `json:"position"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	v, d := assessmentFields(in.Name, in.MaxPoints, in.Weight, in.DueOn, in.Position)
	if len(d) > 0 {
		return 0, nil, httpx.Validation(d...)
	}
	a, err := scanAssessment(tx.QueryRowContext(r.Context(), `insert into assessments (university_id, section_id, name, max_points, weight, due_on, published, position)
		values ($1, $2, $3, $4::numeric, $5::numeric, nullif($6, '')::date, $7,
		        coalesce($8::int, (select coalesce(max(position) + 1, 0) from assessments where section_id = $2)))
		returning `+assessmentCols, s.UniversityID, secID, v.Name, v.MaxPoints, v.Weight, v.DueOn, in.Published, in.Position))
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
	if err := teach(r, tx, s, secID); err != nil {
		return 0, nil, err
	}
	var in struct {
		Name      tenant.Opt[string]      `json:"name"`
		MaxPoints tenant.Opt[json.Number] `json:"maxPoints"`
		Weight    tenant.Opt[json.Number] `json:"weight"`
		DueOn     tenant.Opt[string]      `json:"dueOn"`
		Published tenant.Opt[bool]        `json:"published"`
		Position  tenant.Opt[int]         `json:"position"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	var d []httpx.FieldError
	for _, f := range []struct {
		name string
		null bool
	}{{"name", in.Name.Null}, {"maxPoints", in.MaxPoints.Null}, {"weight", in.Weight.Null}, {"published", in.Published.Null}, {"position", in.Position.Null}} {
		if f.null {
			d = append(d, httpx.FieldError{Field: f.name, Message: "must not be null"})
		}
	}
	if len(d) > 0 {
		return 0, nil, httpx.Validation(d...)
	}
	if !in.Name.Set && !in.MaxPoints.Set && !in.Weight.Set && !in.DueOn.Set && !in.Published.Set && !in.Position.Set {
		return 0, nil, tenant.NoFields()
	}
	after := before
	emptyDue := false // "" is a 400 (null clears), collected with the other field errors
	if in.Name.Set {
		after.Name = in.Name.V
	}
	if in.MaxPoints.Set {
		after.MaxPoints = in.MaxPoints.V
	}
	if in.Weight.Set {
		after.Weight = in.Weight.V
	}
	if in.DueOn.Set {
		after.DueOn = in.DueOn.V // null: V is "" and clears it
		emptyDue = !in.DueOn.Null && in.DueOn.V == ""
	}
	if in.Published.Set {
		after.Published = in.Published.V
	}
	if in.Position.Set {
		after.Position = in.Position.V
	}
	// ponytail: a changed maxPoints does not re-check existing scores against 2 x max; validate on PATCH if it matters
	v, d := assessmentFields(after.Name, after.MaxPoints, after.Weight, after.DueOn, &after.Position)
	if emptyDue {
		d = append(d, httpx.FieldError{Field: "dueOn", Message: "must be a date YYYY-MM-DD"})
	}
	if len(d) > 0 {
		return 0, nil, httpx.Validation(d...)
	}
	after.Name, after.MaxPoints, after.Weight = v.Name, v.MaxPoints, v.Weight
	after, err = scanAssessment(tx.QueryRowContext(r.Context(), `update assessments set name = $2, max_points = $3::numeric, weight = $4::numeric,
		due_on = nullif($5, '')::date, published = $6, position = $7 where id = $1 returning `+assessmentCols,
		after.ID, after.Name, after.MaxPoints, after.Weight, after.DueOn, after.Published, after.Position))
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
	if err := teach(r, tx, s, secID); err != nil {
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
