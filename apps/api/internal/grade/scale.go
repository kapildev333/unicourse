package grade

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/kapildev333/unicourse/apps/api/internal/audit"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

type ScaleRow struct {
	Letter     string      `json:"letter"`
	MinPercent json.Number `json:"minPercent"`
	Points     json.Number `json:"points"`
}

type Scale struct {
	Rows []ScaleRow `json:"rows"`
}

// loadScale returns the university's rows, highest minPercent first (RLS scopes it).
func loadScale(ctx context.Context, tx *sql.Tx) ([]ScaleRow, error) {
	rows, err := tx.QueryContext(ctx, `select letter, min_percent::text, points::text from grade_scales order by min_percent desc`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ScaleRow{}
	for rows.Next() {
		var x ScaleRow
		if err := rows.Scan(&x.Letter, &x.MinPercent, &x.Points); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func getScale(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	rows, err := loadScale(r.Context(), tx)
	return http.StatusOK, Scale{rows}, err
}

func putScale(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	if !authz.IsAdmin(s) {
		return 0, nil, httpx.Forbidden()
	}
	var in struct {
		Rows []struct {
			Letter     string      `json:"letter"`
			MinPercent json.Number `json:"minPercent"`
			Points     json.Number `json:"points"`
		} `json:"rows"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	if len(in.Rows) < 1 || len(in.Rows) > 50 {
		return 0, nil, httpx.Validation(httpx.FieldError{Field: "rows", Message: "must have 1 to 50 entries"})
	}
	var d []httpx.FieldError
	var ls, ms, ps []string
	letters, mins := map[string]bool{}, map[string]bool{}
	for i, x := range in.Rows {
		f := func(name string) string { return "rows[" + strconv.Itoa(i) + "]." + name }
		l, fe := tenant.TrimLen(f("letter"), x.Letter, 10)
		if fe != nil {
			d = append(d, *fe)
		} else if letters[l] {
			d = append(d, httpx.FieldError{Field: f("letter"), Message: "duplicate letter"})
		}
		letters[l] = true
		m, fe := tenant.Decimal(f("minPercent"), x.MinPercent, 2, "0", "100")
		if fe != nil {
			d = append(d, *fe)
		} else if mins[m] {
			d = append(d, httpx.FieldError{Field: f("minPercent"), Message: "duplicate minPercent"})
		}
		mins[m] = true
		p, fe := tenant.Decimal(f("points"), x.Points, 2, "0", "9.99")
		if fe != nil {
			d = append(d, *fe)
		}
		ls, ms, ps = append(ls, l), append(ms, m), append(ps, p)
	}
	if len(d) == 0 && !mins["0.00"] {
		d = append(d, httpx.FieldError{Field: "rows", Message: "must include a row with minPercent 0"})
	}
	if len(d) > 0 {
		return 0, nil, httpx.Validation(d...)
	}
	ctx := r.Context()
	before, err := loadScale(ctx, tx)
	if err != nil {
		return 0, nil, err
	}
	if _, err := tx.ExecContext(ctx, `delete from grade_scales where university_id = $1`, s.UniversityID); err != nil {
		return 0, nil, err
	}
	if _, err := tx.ExecContext(ctx, `insert into grade_scales (university_id, letter, min_percent, points)
		select $1, l, m::numeric, p::numeric from unnest($2::text[], $3::text[], $4::text[]) as x(l, m, p)`, s.UniversityID, ls, ms, ps); err != nil {
		if err = pg.MapError(err); errors.Is(err, pg.ErrConflict) {
			return 0, nil, httpx.Conflict("grade scale changed concurrently")
		}
		return 0, nil, err
	}
	after, err := loadScale(ctx, tx)
	if err != nil {
		return 0, nil, err
	}
	if err := audit.Write(ctx, tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "grade_scale.update", EntityType: "grade_scale", EntityID: s.UniversityID, Before: Scale{before}, After: Scale{after}}); err != nil {
		return 0, nil, err
	}
	return http.StatusOK, Scale{after}, nil
}
