package calendar

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

// meeting is one occurrence of a section meeting: Day is YYYYMMDD in campus time, Start and End are instants.
type meeting struct {
	ID, Day, Code, Number, Title, Location string
	Start, End                             time.Time
}

// meetingsSQL expands meetings in the database: (date + time) at time zone is DST-correct. The window is
// campus today -30 .. +180 days, clipped to each term. $1 user, $2 university timezone, $3 campus today.
const meetingsSQL = `with mine as (select section_id from section_staff where user_id = $1 union select section_id from enrollments where user_id = $1)
select m.id, to_char(d::date, 'YYYYMMDD'), c.code, s.number, c.title, m.location,
  (d::date + m.starts_at) at time zone $2, (d::date + m.ends_at) at time zone $2
from mine join sections s on s.id = mine.section_id join courses c on c.id = s.course_id join terms t on t.id = s.term_id
join section_meetings m on m.section_id = s.id,
  generate_series(greatest(t.starts_on, $3::date - 30)::timestamp, least(t.ends_on, $3::date + 180)::timestamp, interval '1 day') d
where extract(isodow from d) = m.weekday
order by 7, m.id`

func meetings(ctx context.Context, tx *sql.Tx, userID, tz, today string) ([]meeting, error) {
	rows, err := tx.QueryContext(ctx, meetingsSQL, userID, tz, today)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []meeting
	for rows.Next() {
		var m meeting
		if err := rows.Scan(&m.ID, &m.Day, &m.Code, &m.Number, &m.Title, &m.Location, &m.Start, &m.End); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// tokenShape reports whether s is 43 base64url characters that decode to 32 bytes (what auth.NewToken makes).
func tokenShape(s string) bool {
	b, err := base64.RawURLEncoding.DecodeString(s)
	return len(s) == 43 && err == nil && len(b) == 32 && base64.RawURLEncoding.EncodeToString(b) == s
}

// feed serves GET /universities/{slug}/calendar-feeds/{token}.ics without a session: the token is the credential.
// Every failure is the same 404, and the log carries only the route pattern (D27).
func feed(d tenant.Deps) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		ctx := r.Context()
		token, ok := strings.CutSuffix(r.PathValue("token"), ".ics")
		if !ok || !tokenShape(token) {
			return httpx.NotFound("calendar feed")
		}
		var uniID, name, tz string
		err := pg.WithPlatformTx(ctx, d.DB, func(tx *sql.Tx) error {
			return tx.QueryRowContext(ctx, `select id, name, timezone from universities where slug = $1`, r.PathValue("slug")).Scan(&uniID, &name, &tz)
		})
		if errors.Is(err, sql.ErrNoRows) {
			return httpx.NotFound("calendar feed")
		}
		if err != nil {
			return err
		}
		httpx.WithUniversityID(ctx, uniID)

		var evs []event
		var stamp time.Time
		base := strings.TrimRight(d.BaseURL, "/") + "/" + r.PathValue("slug")
		err = pg.WithTx(ctx, d.DB, uniID, func(tx *sql.Tx) error {
			var userID, today string
			err := tx.QueryRowContext(ctx, `select f.user_id from calendar_feeds f join memberships m on m.university_id = f.university_id and m.user_id = f.user_id and m.status = 'active'
				where f.token_hash = $1`, hashToken(token)).Scan(&userID)
			if err != nil {
				return err
			}
			if err := tx.QueryRowContext(ctx, `select now(), to_char((now() at time zone $1)::date, 'YYYY-MM-DD')`, tz).Scan(&stamp, &today); err != nil {
				return err
			}
			ms, err := meetings(ctx, tx, userID, tz, today)
			if err != nil {
				return err
			}
			for _, m := range ms {
				evs = append(evs, event{UID: "meeting-" + m.ID + "-" + m.Day + "@unicourse", Summary: m.Code + "-" + m.Number + " " + m.Title, Location: m.Location, Start: m.Start, End: m.End})
			}
			ws, err := work(ctx, tx, userID, "180 days")
			if err != nil {
				return err
			}
			for _, u := range ws {
				verb, path := "Quiz closes: ", "/quizzes/"
				if u.Kind == "assignment" {
					verb, path = "Assignment due: ", "/assignments/"
				}
				evs = append(evs, event{UID: "assessment-" + u.AssessmentID + "@unicourse", Summary: u.Course.Code + "-" + u.Section.Number + " " + verb + u.Name,
					Description: base + "/sections/" + u.Section.ID + path + u.AssessmentID, Start: u.DueAt, End: u.DueAt, Transparent: true})
			}
			return nil
		})
		if errors.Is(err, sql.ErrNoRows) {
			return httpx.NotFound("calendar feed")
		}
		if err != nil {
			return err
		}
		h := w.Header()
		h.Set("Content-Type", "text/calendar; charset=utf-8")
		h.Set("Content-Disposition", `attachment; filename="unicourse.ics"`)
		h.Set("Cache-Control", "no-store")
		h.Set("Referrer-Policy", "no-referrer")
		_, _ = w.Write([]byte(render(name, stamp, evs))) // headers are sent; a write error is a client disconnect
		return nil
	}
}
