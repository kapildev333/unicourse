// Package calendar serves the member's Upcoming work and the private ICS calendar feed (Spec 13 R1, R4).
package calendar

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/quiz"
)

// Upcoming is one quiz or assignment of a section the caller is in. It carries no keys, scores or other
// students; counts appear only for staff.
type Upcoming struct {
	AssessmentID string `json:"assessmentId"`
	Kind         string `json:"kind"`
	Name         string `json:"name"`
	Section      struct {
		ID     string `json:"id"`
		Number string `json:"number"`
	} `json:"section"`
	Course struct {
		Code  string `json:"code"`
		Title string `json:"title"`
	} `json:"course"`
	MyRole           string     `json:"myRole"`
	OpensAt          time.Time  `json:"opensAt"`
	DueAt            time.Time  `json:"dueAt"`
	LateUntil        *time.Time `json:"lateUntil,omitempty"`
	TimeLimitMinutes *int       `json:"timeLimitMinutes,omitempty"`
	Started          bool       `json:"started"`
	SubmittedCount   *int       `json:"submittedCount,omitempty"`
	EnrolledCount    *int       `json:"enrolledCount,omitempty"`
}

// deadline is the effective deadline (quiz.DeadlineSQL, D150); end is when the work stops being actionable:
// an assignment inside its late window stays listed for the student.
const upcomingSQL = `with mine as (
  select section_id, role from section_staff where user_id = $1
  union all
  select e.section_id, 'student' from enrollments e where e.user_id = $1
    and not exists (select 1 from section_staff ss where ss.section_id = e.section_id and ss.user_id = $1))
select q.id, q.kind, q.name, s.id, s.number, c.code, c.title, mine.role, q.opens_at,
  ` + quiz.DeadlineSQL + ` as due_at, q.late_until, q.time_limit_minutes, a.started_at is not null,
  case when mine.role <> 'student' then (select count(*) from enrollments e where e.section_id = s.id) end,
  case when mine.role <> 'student' then case when q.kind = 'quiz'
    then (select count(*) from attempts x join enrollments e on e.section_id = s.id and e.user_id = x.user_id where x.assessment_id = q.id and x.submitted_at is not null)
    else (select count(*) from submissions x join enrollments e on e.section_id = s.id and e.user_id = x.user_id where x.assessment_id = q.id) end end
from mine join sections s on s.id = mine.section_id join courses c on c.id = s.course_id
join assessments q on q.section_id = s.id and q.kind in ('quiz', 'assignment')
left join attempts a on a.assessment_id = q.id and a.user_id = $1 and mine.role = 'student'
where (mine.role <> 'student' or (a.submitted_at is null
       and not exists (select 1 from submissions x where x.assessment_id = q.id and x.user_id = $1)))
  and (case when q.kind = 'assignment' and mine.role = 'student' then coalesce(q.late_until, q.closes_at) else ` + quiz.DeadlineSQL + ` end) > now()
  and (q.opens_at <= now() or ` + quiz.DeadlineSQL + ` <= now() + $2::interval)
order by due_at, q.id`

// work lists the caller's open or soon-due work; horizon is a SQL interval literal ('14 days' here, '180 days' for the feed).
func work(ctx context.Context, tx *sql.Tx, userID, horizon string) ([]Upcoming, error) {
	rows, err := tx.QueryContext(ctx, upcomingSQL, userID, horizon)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Upcoming{}
	for rows.Next() {
		var u Upcoming
		var late sql.NullTime
		var limit, enrolled, submitted sql.NullInt64
		if err := rows.Scan(&u.AssessmentID, &u.Kind, &u.Name, &u.Section.ID, &u.Section.Number, &u.Course.Code, &u.Course.Title, &u.MyRole,
			&u.OpensAt, &u.DueAt, &late, &limit, &u.Started, &enrolled, &submitted); err != nil {
			return nil, err
		}
		u.OpensAt, u.DueAt = u.OpensAt.UTC(), u.DueAt.UTC()
		if late.Valid {
			t := late.Time.UTC()
			u.LateUntil = &t
		}
		if limit.Valid {
			n := int(limit.Int64)
			u.TimeLimitMinutes = &n
		}
		if enrolled.Valid {
			n, m := int(enrolled.Int64), int(submitted.Int64)
			u.EnrolledCount, u.SubmittedCount = &n, &m
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func upcoming(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	items, err := work(r.Context(), tx, s.UserID, "14 days")
	return http.StatusOK, map[string]any{"items": items}, err
}
