package tenant

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/audit"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
)

// Term is the contract resource; createdAt only feeds the list cursor.
type Term struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	StartsOn  string `json:"startsOn"`
	EndsOn    string `json:"endsOn"`
	createdAt time.Time
}

const dateLayout = "2006-01-02"

// termFields validates a full (merged) term; empty dates are "is required".
func termFields(t Term) (Term, error) {
	var details []httpx.FieldError
	name, fe := trimLen("name", t.Name, 100)
	if fe != nil {
		details = append(details, *fe)
	}
	t.Name = name
	var d [2]time.Time
	for i, f := range []struct{ field, v string }{{"startsOn", t.StartsOn}, {"endsOn", t.EndsOn}} {
		if f.v == "" {
			details = append(details, httpx.FieldError{Field: f.field, Message: "is required"})
			continue
		}
		var err error
		if d[i], err = time.Parse(dateLayout, f.v); err != nil {
			details = append(details, httpx.FieldError{Field: f.field, Message: "must be a date YYYY-MM-DD"})
		}
	}
	if !d[0].IsZero() && !d[1].IsZero() && !d[1].After(d[0]) {
		details = append(details, httpx.FieldError{Field: "endsOn", Message: "must be after startsOn"})
	}
	if len(details) > 0 {
		return t, httpx.Validation(details...)
	}
	return t, nil
}

func termWriteErr(err error) error {
	if err = pg.MapError(err); errors.Is(err, pg.ErrConflict) {
		return httpx.Conflict("term name already exists")
	}
	return err
}

func listTerms(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	p, after, afterID, err := page(r)
	if err != nil {
		return 0, nil, err
	}
	rows, err := tx.QueryContext(r.Context(), `select id, name, to_char(starts_on,'YYYY-MM-DD'), to_char(ends_on,'YYYY-MM-DD'), created_at from terms
		where ($1::timestamptz is null or (created_at, id) > ($1, $2::uuid))
		order by created_at, id limit $3`, after, afterID, p.Limit+1)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	out := httpx.List[Term]{Items: []Term{}}
	var last Term
	for rows.Next() {
		var t Term
		if err := rows.Scan(&t.ID, &t.Name, &t.StartsOn, &t.EndsOn, &t.createdAt); err != nil {
			return 0, nil, err
		}
		if len(out.Items) == p.Limit {
			out.NextCursor = httpx.NextCursor(last.createdAt.UTC(), last.ID)
			break
		}
		out.Items = append(out.Items, t)
		last = t
	}
	if err := rows.Err(); err != nil {
		return 0, nil, err
	}
	return http.StatusOK, out, nil
}

func createTerm(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	if !authz.IsAdmin(s) {
		return 0, nil, httpx.Forbidden()
	}
	var in struct{ Name, StartsOn, EndsOn string }
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	t, err := termFields(Term{Name: in.Name, StartsOn: in.StartsOn, EndsOn: in.EndsOn})
	if err != nil {
		return 0, nil, err
	}
	if err := tx.QueryRowContext(r.Context(), `insert into terms (university_id, name, starts_on, ends_on) values ($1,$2,$3::date,$4::date) returning id, created_at`,
		s.UniversityID, t.Name, t.StartsOn, t.EndsOn).Scan(&t.ID, &t.createdAt); err != nil {
		return 0, nil, termWriteErr(err)
	}
	if err := audit.Write(r.Context(), tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "term.create", EntityType: "term", EntityID: t.ID, After: t}); err != nil {
		return 0, nil, err
	}
	return http.StatusCreated, t, nil
}

func updateTerm(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	if !authz.IsAdmin(s) {
		return 0, nil, httpx.Forbidden()
	}
	id := r.PathValue("id")
	if !isUUID(id) {
		return 0, nil, httpx.NotFound("term")
	}
	var in struct{ Name, StartsOn, EndsOn *string }
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	if in.Name == nil && in.StartsOn == nil && in.EndsOn == nil {
		return 0, nil, noFields()
	}
	var before Term
	err := tx.QueryRowContext(r.Context(), `select id, name, to_char(starts_on,'YYYY-MM-DD'), to_char(ends_on,'YYYY-MM-DD'), created_at from terms where id = $1 for update`, id).
		Scan(&before.ID, &before.Name, &before.StartsOn, &before.EndsOn, &before.createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil, httpx.NotFound("term")
	} else if err != nil {
		return 0, nil, err
	}
	after := before
	if in.Name != nil {
		after.Name = *in.Name
	}
	if in.StartsOn != nil {
		after.StartsOn = *in.StartsOn
	}
	if in.EndsOn != nil {
		after.EndsOn = *in.EndsOn
	}
	if after, err = termFields(after); err != nil {
		return 0, nil, err
	}
	if _, err := tx.ExecContext(r.Context(), `update terms set name = $2, starts_on = $3::date, ends_on = $4::date where id = $1`, id, after.Name, after.StartsOn, after.EndsOn); err != nil {
		return 0, nil, termWriteErr(err)
	}
	if err := audit.Write(r.Context(), tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "term.update", EntityType: "term", EntityID: id, Before: before, After: after}); err != nil {
		return 0, nil, err
	}
	return http.StatusOK, after, nil
}

func deleteTerm(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	if !authz.IsAdmin(s) {
		return 0, nil, httpx.Forbidden()
	}
	id := r.PathValue("id")
	if !isUUID(id) {
		return 0, nil, httpx.NotFound("term")
	}
	var before Term
	err := tx.QueryRowContext(r.Context(), `delete from terms where id = $1 returning id, name, to_char(starts_on,'YYYY-MM-DD'), to_char(ends_on,'YYYY-MM-DD')`, id).
		Scan(&before.ID, &before.Name, &before.StartsOn, &before.EndsOn)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil, httpx.NotFound("term")
	}
	if err != nil {
		if err = pg.MapDeleteError(err); errors.Is(err, pg.ErrConflict) {
			return 0, nil, httpx.Conflict("term is still in use")
		}
		return 0, nil, err
	}
	if err := audit.Write(r.Context(), tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "term.delete", EntityType: "term", EntityID: id, Before: before}); err != nil {
		return 0, nil, err
	}
	return http.StatusNoContent, nil, nil
}
