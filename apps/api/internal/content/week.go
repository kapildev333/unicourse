package content

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/kapildev333/unicourse/apps/api/internal/audit"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/blob"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

type week struct {
	ID        string     `json:"id"`
	SectionID string     `json:"sectionId"`
	Title     string     `json:"title"`
	Notes     string     `json:"notes"`
	Position  int        `json:"position"`
	Materials []material `json:"materials"`
}

type weekAudit struct {
	ID        string          `json:"id"`
	SectionID string          `json:"sectionId"`
	Title     string          `json:"title"`
	Notes     string          `json:"notes"`
	Position  int             `json:"position"`
	Materials []materialAudit `json:"materials,omitempty"` // week.delete only: what the cascade removed
}

type weekList struct {
	Items []week `json:"items"`
}

func (w week) audit() weekAudit {
	return weekAudit{w.ID, w.SectionID, w.Title, w.Notes, w.Position, nil}
}

// loadWeeks returns the section's weeks by position, each with its materials by position.
func loadWeeks(ctx context.Context, tx *sql.Tx, sectionID string) ([]week, error) {
	rows, err := tx.QueryContext(ctx, `select id, section_id, title, notes, position from weeks where section_id = $1 order by position, id`, sectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []week{}
	idx := map[string]int{}
	for rows.Next() {
		var w week
		if err := rows.Scan(&w.ID, &w.SectionID, &w.Title, &w.Notes, &w.Position); err != nil {
			return nil, err
		}
		w.Materials = []material{}
		idx[w.ID] = len(out)
		out = append(out, w)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	mrows, err := tx.QueryContext(ctx, `select `+materialCols+materialFrom+` where m.section_id = $1 order by m.position, m.id`, sectionID)
	if err != nil {
		return nil, err
	}
	defer mrows.Close()
	for mrows.Next() {
		m, err := scanMaterial(mrows)
		if err != nil {
			return nil, err
		}
		if i, ok := idx[m.WeekID]; ok {
			out[i].Materials = append(out[i].Materials, m)
		}
	}
	return out, mrows.Err()
}

// loadWeek: bad uuid, no row or another section's week -> 404.
func loadWeek(r *http.Request, tx *sql.Tx, sectionID, id string) (week, error) {
	w := week{Materials: []material{}}
	if !tenant.IsUUID(id) {
		return w, httpx.NotFound("week")
	}
	err := tx.QueryRowContext(r.Context(), `select id, section_id, title, notes, position from weeks where id = $1 and section_id = $2`, id, sectionID).
		Scan(&w.ID, &w.SectionID, &w.Title, &w.Notes, &w.Position)
	if errors.Is(err, sql.ErrNoRows) {
		return w, httpx.NotFound("week")
	}
	return w, err
}

// renumberWeeks makes the section's positions 1..n. With movedID != "", that week lands at to, coming from from.
func renumberWeeks(ctx context.Context, tx *sql.Tx, sectionID, movedID string, to, from int) error {
	var moved any
	if movedID != "" {
		moved = movedID
	}
	_, err := tx.ExecContext(ctx, `update weeks w set position = o.n
		from (select id, row_number() over (order by
		        case when id = $2::uuid then $3::int * 2 + case when $3::int > $4::int then 1 else -1 end
		             else position * 2 end, id) as n
		      from weeks where section_id = $1) o
		where w.id = o.id and w.position <> o.n`, sectionID, moved, to, from)
	return err
}

func countWeeks(ctx context.Context, tx *sql.Tx, sectionID string) (n int, err error) {
	err = tx.QueryRowContext(ctx, `select count(*) from weeks where section_id = $1`, sectionID).Scan(&n)
	return
}

func listWeeks(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	secID := r.PathValue("id")
	if err := sectionExists(r, tx, secID, false); err != nil {
		return 0, nil, err
	}
	if err := allow(authz.CanReadContent(r.Context(), tx, s, secID)); err != nil {
		return 0, nil, err
	}
	ws, err := loadWeeks(r.Context(), tx, secID)
	if err != nil {
		return 0, nil, err
	}
	return http.StatusOK, weekList{Items: ws}, nil
}

func createWeek(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	secID := r.PathValue("id")
	if err := sectionExists(r, tx, secID, true); err != nil {
		return 0, nil, err
	}
	if err := allow(authz.CanEditContent(r.Context(), tx, s, secID)); err != nil {
		return 0, nil, err
	}
	var in struct {
		Title string `json:"title"`
		Notes string `json:"notes"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	var d []httpx.FieldError
	title, fe := line("title", in.Title, 100)
	if fe != nil {
		d = append(d, *fe)
	}
	notes, fe := optText("notes", in.Notes, 5000)
	if fe != nil {
		d = append(d, *fe)
	}
	if len(d) > 0 {
		return 0, nil, httpx.Validation(d...)
	}
	n, err := countWeeks(r.Context(), tx, secID)
	if err != nil {
		return 0, nil, err
	}
	if n >= 100 {
		return 0, nil, httpx.Validation(httpx.FieldError{Field: "weeks", Message: "a section holds at most 100 weeks"})
	}
	w := week{SectionID: secID, Title: title, Notes: notes, Position: n + 1, Materials: []material{}}
	if err := tx.QueryRowContext(r.Context(), `insert into weeks (university_id, section_id, title, notes, position) values ($1, $2, $3, $4, $5) returning id`,
		s.UniversityID, secID, title, notes, w.Position).Scan(&w.ID); err != nil {
		return 0, nil, err
	}
	if err := audit.Write(r.Context(), tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "week.create", EntityType: "week", EntityID: w.ID, After: w.audit()}); err != nil {
		return 0, nil, err
	}
	return http.StatusCreated, w, nil
}

func updateWeek(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	ctx := r.Context()
	secID := r.PathValue("id")
	if err := sectionExists(r, tx, secID, true); err != nil {
		return 0, nil, err
	}
	before, err := loadWeek(r, tx, secID, r.PathValue("weekId"))
	if err != nil {
		return 0, nil, err
	}
	if err := allow(authz.CanEditContent(ctx, tx, s, secID)); err != nil {
		return 0, nil, err
	}
	var in struct {
		Title    tenant.Opt[string] `json:"title"`
		Notes    tenant.Opt[string] `json:"notes"`
		Position tenant.Opt[int]    `json:"position"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	var d []httpx.FieldError
	for _, f := range []struct {
		name string
		null bool
	}{{"title", in.Title.Null}, {"notes", in.Notes.Null}, {"position", in.Position.Null}} {
		if f.null {
			d = append(d, httpx.FieldError{Field: f.name, Message: "must not be null"})
		}
	}
	if len(d) > 0 {
		return 0, nil, httpx.Validation(d...)
	}
	if !in.Title.Set && !in.Notes.Set && !in.Position.Set {
		return 0, nil, tenant.NoFields()
	}
	title, notes := before.Title, before.Notes
	if in.Title.Set {
		var fe *httpx.FieldError
		if title, fe = line("title", in.Title.V, 100); fe != nil {
			d = append(d, *fe)
		}
	}
	if in.Notes.Set {
		var fe *httpx.FieldError
		if notes, fe = optText("notes", in.Notes.V, 5000); fe != nil {
			d = append(d, *fe)
		}
	}
	if in.Position.Set {
		n, err := countWeeks(ctx, tx, secID)
		if err != nil {
			return 0, nil, err
		}
		if p := in.Position.V; p < 1 || p > n {
			d = append(d, httpx.FieldError{Field: "position", Message: "must be between 1 and " + strconv.Itoa(n)})
		}
	}
	if len(d) > 0 {
		return 0, nil, httpx.Validation(d...)
	}
	if _, err := tx.ExecContext(ctx, `update weeks set title = $2, notes = $3 where id = $1`, before.ID, title, notes); err != nil {
		return 0, nil, err
	}
	if in.Position.Set && in.Position.V != before.Position {
		if err := renumberWeeks(ctx, tx, secID, before.ID, in.Position.V, before.Position); err != nil {
			return 0, nil, err
		}
	}
	ws, err := loadWeeks(ctx, tx, secID)
	if err != nil {
		return 0, nil, err
	}
	for _, w := range ws {
		if w.ID == before.ID {
			if err := audit.Write(ctx, tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "week.update", EntityType: "week", EntityID: w.ID, Before: before.audit(), After: w.audit()}); err != nil {
				return 0, nil, err
			}
			return http.StatusOK, w, nil
		}
	}
	return 0, nil, errors.New("week vanished during update")
}

func deleteWeek(files blob.Dir) tenant.Handler {
	return func(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
		ctx := r.Context()
		secID := r.PathValue("id")
		if err := sectionExists(r, tx, secID, true); err != nil {
			return 0, nil, err
		}
		before, err := loadWeek(r, tx, secID, r.PathValue("weekId"))
		if err != nil {
			return 0, nil, err
		}
		if err := allow(authz.CanEditContent(ctx, tx, s, secID)); err != nil {
			return 0, nil, err
		}
		rows, err := tx.QueryContext(ctx, `select `+materialCols+materialFrom+` where m.week_id = $1 order by m.position, m.id`, before.ID)
		if err != nil {
			return 0, nil, err
		}
		defer rows.Close()
		a := before.audit()
		var ids []string
		for rows.Next() {
			m, err := scanMaterial(rows)
			if err != nil {
				return 0, nil, err
			}
			a.Materials = append(a.Materials, m.audit(secID))
			ids = append(ids, m.ID)
		}
		if err := rows.Err(); err != nil {
			return 0, nil, err
		}
		if _, err := tx.ExecContext(ctx, `delete from weeks where id = $1`, before.ID); err != nil {
			return 0, nil, err
		}
		if err := renumberWeeks(ctx, tx, secID, "", 0, 0); err != nil {
			return 0, nil, err
		}
		if err := audit.Write(ctx, tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "week.delete", EntityType: "week", EntityID: before.ID, Before: a}); err != nil {
			return 0, nil, err
		}
		return http.StatusNoContent, noContentAfter(files, s.UniversityID, ids...), nil
	}
}

// noContentAfter answers 204 once the transaction committed and then removes the blobs of the deleted materials.
func noContentAfter(files blob.Dir, universityID string, ids ...string) tenant.After {
	return func(w http.ResponseWriter, _ *http.Request) error {
		files.Discard(universityID, ids...)
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
}
