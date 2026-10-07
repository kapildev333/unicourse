package grade

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/audit"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

type FinalGrade struct {
	UserID         string      `json:"userId"`
	Email          string      `json:"email"`
	DisplayName    string      `json:"displayName"`
	Status         string      `json:"status"`
	Percent        json.Number `json:"percent,omitempty"`
	Letter         string      `json:"letter,omitempty"`
	Points         json.Number `json:"points,omitempty"`
	Credits        json.Number `json:"credits"`
	OverrideReason string      `json:"overrideReason,omitempty"`
	FinalizedBy    string      `json:"finalizedBy"`
	FinalizedAt    time.Time   `json:"finalizedAt"`
	id             string      // row id, the override audit entity
}

type finalList struct {
	Items []FinalGrade `json:"items"`
}

// gradeAudit is the audit payload element for a final grade.
type gradeAudit struct {
	UserID         string      `json:"userId"`
	Status         string      `json:"status"`
	Percent        json.Number `json:"percent,omitempty"`
	Letter         string      `json:"letter,omitempty"`
	Points         json.Number `json:"points,omitempty"`
	OverrideReason string      `json:"overrideReason,omitempty"`
}

type override struct {
	UserID string             `json:"userId"`
	Status tenant.Opt[string] `json:"status"`
	Letter tenant.Opt[string] `json:"letter"`
	Reason string             `json:"reason"`
}

// readFinal lists the section's final grades (also of students no longer enrolled), ordered by email.
func readFinal(ctx context.Context, tx *sql.Tx, sectionID string) ([]FinalGrade, error) {
	rows, err := tx.QueryContext(ctx, `select fg.id, u.id, u.email, u.display_name, fg.status, fg.percent::text, fg.letter, fg.points::text,
		fg.credits::text, fg.override_reason, fg.finalized_by, fg.finalized_at
		from final_grades fg join users u on u.id = fg.user_id where fg.section_id = $1 order by u.email`, sectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FinalGrade{}
	for rows.Next() {
		var f FinalGrade
		var pct, letter, pts, reason sql.NullString
		if err := rows.Scan(&f.id, &f.UserID, &f.Email, &f.DisplayName, &f.Status, &pct, &letter, &pts, &f.Credits, &reason, &f.FinalizedBy, &f.FinalizedAt); err != nil {
			return nil, err
		}
		f.Percent, f.Letter, f.Points, f.OverrideReason = json.Number(pct.String), letter.String, json.Number(pts.String), reason.String
		f.FinalizedAt = f.FinalizedAt.UTC()
		out = append(out, f)
	}
	return out, rows.Err()
}

func listFinal(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	secID := r.PathValue("id")
	if err := sectionExists(r, tx, secID); err != nil {
		return 0, nil, err
	}
	if err := grade(r, tx, s, secID); err != nil {
		return 0, nil, err
	}
	items, err := readFinal(r.Context(), tx, secID)
	return http.StatusOK, finalList{items}, err
}

// grade computes one student's final row: percent over every assessment (missing = 0), letter from the unrounded percent.
func (m matrix) grade(st GradebookStudent) (gradeAudit, error) {
	p, ok := percentOf(m.finalParts(st))
	if !ok {
		return gradeAudit{}, httpx.Conflict("the assessments' weights sum to 0")
	}
	row, ok := letterFor(m.scale, p)
	if !ok {
		return gradeAudit{}, httpx.Conflict("the grade scale has no row for " + string(round2(p)) + " percent")
	}
	return gradeAudit{UserID: st.UserID, Status: "graded", Percent: round2(p), Letter: row.Letter, Points: row.Points}, nil
}

// validateOverride checks one override; prefix is "overrides[<i>]." or "" (PATCH); cur is the row's current status.
func validateOverride(prefix string, o override, enrolled map[string]bool, scale []ScaleRow, cur string) []httpx.FieldError {
	var d []httpx.FieldError
	bad := func(f, m string) { d = append(d, httpx.FieldError{Field: prefix + f, Message: m}) }
	switch {
	case !tenant.IsUUID(o.UserID):
		bad("userId", "must be a uuid")
	case !enrolled[o.UserID]:
		bad("userId", "is not enrolled in this section")
	}
	status := cur
	switch {
	case o.Status.Null:
		bad("status", "must not be null")
	case o.Status.Set && o.Status.V != "graded" && o.Status.V != "incomplete" && o.Status.V != "withdrawn":
		bad("status", "must be one of graded, incomplete, withdrawn")
	case o.Status.Set:
		status = o.Status.V
	}
	switch {
	case o.Letter.Null:
		bad("letter", "must not be null")
	case o.Letter.Set && status != "graded":
		bad("letter", "must be omitted unless status is graded")
	case o.Letter.Set:
		found := false
		for _, r := range scale {
			found = found || r.Letter == o.Letter.V
		}
		if !found {
			bad("letter", "is not on the grade scale")
		}
	}
	if _, fe := tenant.TrimLen(prefix+"reason", o.Reason, 500); fe != nil {
		d = append(d, *fe)
	}
	if prefix != "" && !o.Status.Set && !o.Letter.Set {
		d = append(d, httpx.FieldError{Field: strings.TrimSuffix(prefix, "."), Message: "must set status or letter"})
	}
	return d
}

// applyOverride returns g changed by o; a graded result takes its points from the scale row of its letter.
func applyOverride(g gradeAudit, o override, m matrix, st GradebookStudent) (gradeAudit, error) {
	g.OverrideReason = strings.TrimSpace(o.Reason)
	if o.Status.Set && o.Status.V != "graded" {
		g.Status, g.Percent, g.Letter, g.Points = o.Status.V, "", "", ""
		return g, nil
	}
	wasGraded := g.Status == "graded"
	g.Status = "graded"
	if !wasGraded { // back to graded: recompute from the scores, the letter from the unrounded percent
		p, ok := percentOf(m.finalParts(st))
		if !ok {
			return g, httpx.Conflict("the assessments' weights sum to 0")
		}
		row, ok := letterFor(m.scale, p)
		if !ok {
			return g, httpx.Conflict("the grade scale has no row for " + string(round2(p)) + " percent")
		}
		g.Percent, g.Letter = round2(p), row.Letter
	}
	if o.Letter.Set {
		g.Letter = o.Letter.V
	}
	for _, r := range m.scale {
		if r.Letter == g.Letter {
			g.Points = r.Points
		}
	}
	return g, nil
}

func finalize(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	ctx := r.Context()
	secID := r.PathValue("id")
	if err := sectionExists(r, tx, secID); err != nil {
		return 0, nil, err
	}
	if err := teach(r, tx, s, secID); err != nil {
		return 0, nil, err
	}
	var in struct {
		Overrides []override `json:"overrides"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	var finalized bool
	if err := tx.QueryRowContext(ctx, `select exists(select 1 from final_grades where section_id = $1)`, secID).Scan(&finalized); err != nil {
		return 0, nil, err
	}
	if finalized {
		return 0, nil, httpx.Conflict("section is already finalized; reopen first")
	}
	m, err := loadMatrix(ctx, tx, secID)
	if err != nil {
		return 0, nil, err
	}
	var d []httpx.FieldError
	sum := new(big.Rat)
	for _, a := range m.assessments {
		sum.Add(sum, rat(a.Weight))
	}
	if sum.Cmp(big.NewRat(100, 1)) != 0 {
		d = append(d, httpx.FieldError{Field: "weights", Message: "must sum to 100.00 (is " + sum.FloatString(2) + ")"})
	}
	enrolled := map[string]bool{}
	byUser := map[string]GradebookStudent{}
	for _, st := range m.students {
		enrolled[st.UserID], byUser[st.UserID] = true, st
	}
	ovr := map[string]override{}
	if len(in.Overrides) > 1000 {
		d = append(d, httpx.FieldError{Field: "overrides", Message: "must have at most 1000 entries"})
	} else {
		for i, o := range in.Overrides {
			p := "overrides[" + strconv.Itoa(i) + "]."
			d = append(d, validateOverride(p, o, enrolled, m.scale, "graded")...)
			if _, dup := ovr[o.UserID]; dup {
				d = append(d, httpx.FieldError{Field: p + "userId", Message: "duplicate userId"})
			}
			ovr[o.UserID] = o
		}
	}
	if len(d) > 0 {
		return 0, nil, httpx.Validation(d...)
	}
	published := []string{}
	rows, err := tx.QueryContext(ctx, `update assessments set published = true where section_id = $1 and not published returning id::text`, secID)
	if err != nil {
		return 0, nil, err
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, nil, err
		}
		published = append(published, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, nil, err
	}
	var credits string
	if err := tx.QueryRowContext(ctx, `select c.credits::text from sections s join courses c on c.id = s.course_id where s.id = $1`, secID).Scan(&credits); err != nil {
		return 0, nil, err
	}
	n := len(m.students)
	grades := make([]gradeAudit, 0, n)
	ids, sts := make([]string, 0, n), make([]string, 0, n)
	pcts, lets, pts, rsns := make([]*string, 0, n), make([]*string, 0, n), make([]*string, 0, n), make([]*string, 0, n)
	opt := func(v string) *string {
		if v == "" {
			return nil
		}
		return &v
	}
	for _, st := range m.students {
		g, err := m.grade(st)
		if err != nil {
			return 0, nil, err
		}
		if o, ok := ovr[st.UserID]; ok {
			if g, err = applyOverride(g, o, m, st); err != nil {
				return 0, nil, err
			}
		}
		grades = append(grades, g)
		ids, sts = append(ids, g.UserID), append(sts, g.Status)
		pcts, lets, pts, rsns = append(pcts, opt(string(g.Percent))), append(lets, opt(g.Letter)), append(pts, opt(string(g.Points))), append(rsns, opt(g.OverrideReason))
	}
	if n > 0 {
		if _, err := tx.ExecContext(ctx, `insert into final_grades (university_id, section_id, user_id, status, percent, letter, points, override_reason, credits, finalized_by)
			select $1, $2, u, st, p::numeric, l, pt::numeric, r, $3::numeric, $4
			from unnest($5::uuid[], $6::text[], $7::text[], $8::text[], $9::text[], $10::text[]) as x(u, st, p, l, pt, r)`,
			s.UniversityID, secID, credits, s.UserID, ids, sts, pcts, lets, pts, rsns); err != nil {
			if err = pg.MapError(err); errors.Is(err, pg.ErrConflict) {
				return 0, nil, httpx.Conflict("section is already finalized; reopen first")
			}
			return 0, nil, err
		}
	}
	if err := audit.Write(ctx, tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "final_grade.finalize", EntityType: "section", EntityID: secID,
		After: map[string]any{"sectionId": secID, "publishedAssessmentIds": published, "grades": grades}}); err != nil {
		return 0, nil, err
	}
	items, err := readFinal(ctx, tx, secID)
	return http.StatusCreated, finalList{items}, err
}

func overrideFinal(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	ctx := r.Context()
	secID, userID := r.PathValue("id"), r.PathValue("userId")
	if err := sectionExists(r, tx, secID); err != nil {
		return 0, nil, err
	}
	if err := teach(r, tx, s, secID); err != nil {
		return 0, nil, err
	}
	if !tenant.IsUUID(userID) {
		return 0, nil, httpx.NotFound("final grade")
	}
	var rowID string
	before := gradeAudit{UserID: userID}
	var pct, letter, pts, reason sql.NullString
	err := tx.QueryRowContext(ctx, `select id, status, percent::text, letter, points::text, override_reason from final_grades
		where section_id = $1 and user_id = $2 for update`, secID, userID).Scan(&rowID, &before.Status, &pct, &letter, &pts, &reason)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil, httpx.NotFound("final grade")
	}
	if err != nil {
		return 0, nil, err
	}
	before.Percent, before.Letter, before.Points, before.OverrideReason = json.Number(pct.String), letter.String, json.Number(pts.String), reason.String
	var in struct {
		Status tenant.Opt[string] `json:"status"`
		Letter tenant.Opt[string] `json:"letter"`
		Reason string             `json:"reason"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	if !in.Status.Set && !in.Letter.Set {
		return 0, nil, tenant.NoFields()
	}
	o := override{UserID: userID, Status: in.Status, Letter: in.Letter, Reason: in.Reason}
	var m matrix
	if o.Status.Set && o.Status.V == "graded" && before.Status != "graded" {
		m, err = loadMatrix(ctx, tx, secID)
	} else {
		m.scale, err = loadScale(ctx, tx)
	}
	if err != nil {
		return 0, nil, err
	}
	// a student who is no longer enrolled is not in the matrix: recomputing then sees no scores
	// ponytail: such a student recomputes to 0 percent; read their scores directly if that bites.
	st := GradebookStudent{UserID: userID, Scores: map[string]json.Number{}}
	for _, x := range m.students {
		if x.UserID == userID {
			st = x
		}
	}
	if d := validateOverride("", o, map[string]bool{userID: true}, m.scale, before.Status); len(d) > 0 {
		return 0, nil, httpx.Validation(d...)
	}
	after, err := applyOverride(before, o, m, st)
	if err != nil {
		return 0, nil, err
	}
	opt := func(v string) *string {
		if v == "" {
			return nil
		}
		return &v
	}
	if _, err := tx.ExecContext(ctx, `update final_grades set status = $2, percent = $3::numeric, letter = $4, points = $5::numeric, override_reason = $6 where id = $1`,
		rowID, after.Status, opt(string(after.Percent)), opt(after.Letter), opt(string(after.Points)), opt(after.OverrideReason)); err != nil {
		return 0, nil, err
	}
	type withSection struct {
		gradeAudit
		SectionID string `json:"sectionId"`
	}
	if err := audit.Write(ctx, tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "final_grade.override", EntityType: "final_grade", EntityID: rowID,
		Before: withSection{before, secID}, After: withSection{after, secID}}); err != nil {
		return 0, nil, err
	}
	all, err := readFinal(ctx, tx, secID)
	if err != nil {
		return 0, nil, err
	}
	for _, f := range all {
		if f.id == rowID {
			return http.StatusOK, f, nil
		}
	}
	return 0, nil, httpx.NotFound("final grade")
}

func reopen(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	ctx := r.Context()
	secID := r.PathValue("id")
	if err := sectionExists(r, tx, secID); err != nil {
		return 0, nil, err
	}
	if err := teach(r, tx, s, secID); err != nil {
		return 0, nil, err
	}
	rows, err := tx.QueryContext(ctx, `delete from final_grades where section_id = $1
		returning user_id, status, percent::text, letter, points::text, override_reason`, secID)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	grades := []gradeAudit{}
	for rows.Next() {
		var g gradeAudit
		var pct, letter, pts, reason sql.NullString
		if err := rows.Scan(&g.UserID, &g.Status, &pct, &letter, &pts, &reason); err != nil {
			return 0, nil, err
		}
		g.Percent, g.Letter, g.Points, g.OverrideReason = json.Number(pct.String), letter.String, json.Number(pts.String), reason.String
		grades = append(grades, g)
	}
	if err := rows.Err(); err != nil {
		return 0, nil, err
	}
	rows.Close()
	if len(grades) == 0 {
		return 0, nil, httpx.NotFound("final grades")
	}
	if err := audit.Write(ctx, tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "final_grade.reopen", EntityType: "section", EntityID: secID,
		Before: map[string]any{"grades": grades}}); err != nil {
		return 0, nil, err
	}
	return http.StatusNoContent, nil, nil
}
