package notification

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/audit"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

const base = "/api/v1/universities/{slug}"

// Register mounts the member's own notification routes, each through tenant.Wrap (any active member).
func Register(r *httpx.Router, d tenant.Deps) {
	r.Handle("GET", base+"/me/notifications", tenant.Wrap(d, list))
	r.Handle("GET", base+"/me/notifications/unread-count", tenant.Wrap(d, unreadCount))
	r.Handle("PATCH", base+"/me/notifications/{id}", tenant.Wrap(d, markOne))
	r.Handle("PATCH", base+"/me/notifications", tenant.Wrap(d, markAll))
	r.Handle("GET", base+"/me/notification-settings", tenant.Wrap(d, getSettings))
	r.Handle("PATCH", base+"/me/notification-settings", tenant.Wrap(d, patchSettings))
}

type Notification struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Section struct {
		ID     string `json:"id"`
		Number string `json:"number"`
	} `json:"section"`
	Course struct {
		Code  string `json:"code"`
		Title string `json:"title"`
	} `json:"course"`
	Subject        string `json:"subject,omitempty"`
	AnnouncementID string `json:"announcementId,omitempty"`
	Assessment     *struct {
		ID   string `json:"id"`
		Kind string `json:"kind"`
	} `json:"assessment,omitempty"`
	ReadAt    *time.Time `json:"readAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

// The source tables are left-joined, so the title is always the live one and a deleted source has no row at all.
const selectNotification = `select n.id, n.kind, s.id, s.number, c.code, c.title, coalesce(an.title, q.name, ''),
	coalesce(n.announcement_id::text, ''), coalesce(q.id::text, ''), coalesce(q.kind, ''), n.read_at, n.created_at
	from notifications n join sections s on s.id = n.section_id join courses c on c.id = s.course_id
	left join announcements an on an.id = n.announcement_id left join assessments q on q.id = n.assessment_id `

func scan(sc interface{ Scan(...any) error }) (n Notification, err error) {
	var qid, qkind string
	var read sql.NullTime
	if err = sc.Scan(&n.ID, &n.Kind, &n.Section.ID, &n.Section.Number, &n.Course.Code, &n.Course.Title, &n.Subject, &n.AnnouncementID, &qid, &qkind, &read, &n.CreatedAt); err != nil {
		return
	}
	n.CreatedAt = n.CreatedAt.UTC()
	if read.Valid {
		t := read.Time.UTC()
		n.ReadAt = &t
	}
	if qid != "" {
		n.Assessment = &struct {
			ID   string `json:"id"`
			Kind string `json:"kind"`
		}{qid, qkind}
	}
	return
}

func list(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	p, after, afterID, err := tenant.Page(r)
	if err != nil {
		return 0, nil, err
	}
	rows, err := tx.QueryContext(r.Context(), selectNotification+`where n.user_id = $1
		and ($2::timestamptz is null or (n.created_at, n.id) < ($2, $3::uuid))
		order by n.created_at desc, n.id desc limit $4`, s.UserID, after, afterID, p.Limit+1)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	out := httpx.List[Notification]{Items: []Notification{}}
	for rows.Next() {
		n, err := scan(rows)
		if err != nil {
			return 0, nil, err
		}
		if len(out.Items) == p.Limit {
			last := out.Items[len(out.Items)-1]
			out.NextCursor = httpx.NextCursor(last.CreatedAt, last.ID)
			break
		}
		out.Items = append(out.Items, n)
	}
	return http.StatusOK, out, rows.Err()
}

func unreadCount(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	var n int
	if err := tx.QueryRowContext(r.Context(), `select count(*) from notifications where user_id = $1 and read_at is null`, s.UserID).Scan(&n); err != nil {
		return 0, nil, err
	}
	return http.StatusOK, map[string]int{"count": n}, nil
}

type readIn struct {
	Read tenant.Opt[bool] `json:"read"`
}

func (in readIn) check() error {
	if in.Read.Null {
		return httpx.Validation(httpx.FieldError{Field: "read", Message: "must not be null"})
	}
	if !in.Read.Set {
		return tenant.NoFields()
	}
	return nil
}

func markOne(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	id := r.PathValue("id")
	if !tenant.IsUUID(id) {
		return 0, nil, httpx.NotFound("notification")
	}
	var in readIn
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	if err := in.check(); err != nil {
		return 0, nil, err
	}
	res, err := tx.ExecContext(r.Context(), `update notifications set read_at = case when $3 then coalesce(read_at, now()) end where id = $1 and user_id = $2`, id, s.UserID, in.Read.V)
	if err != nil {
		return 0, nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return 0, nil, httpx.NotFound("notification")
	}
	n, err := scan(tx.QueryRowContext(r.Context(), selectNotification+`where n.id = $1`, id))
	return http.StatusOK, n, err
}

func markAll(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	var in readIn
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	if err := in.check(); err != nil {
		return 0, nil, err
	}
	if !in.Read.V {
		return 0, nil, httpx.Validation(httpx.FieldError{Field: "read", Message: "must be true"})
	}
	if _, err := tx.ExecContext(r.Context(), `update notifications set read_at = now() where user_id = $1 and read_at is null`, s.UserID); err != nil {
		return 0, nil, err
	}
	return http.StatusNoContent, nil, nil
}

type settings struct {
	AnnouncementMail bool `json:"announcementMail"`
}

func getSettings(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	var out settings
	err := tx.QueryRowContext(r.Context(), `select announcement_mail from memberships where user_id = $1`, s.UserID).Scan(&out.AnnouncementMail)
	return http.StatusOK, out, err
}

func patchSettings(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	var in struct {
		AnnouncementMail tenant.Opt[bool] `json:"announcementMail"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	if in.AnnouncementMail.Null {
		return 0, nil, httpx.Validation(httpx.FieldError{Field: "announcementMail", Message: "must not be null"})
	}
	if !in.AnnouncementMail.Set {
		return 0, nil, tenant.NoFields()
	}
	var id string
	var before settings
	err := tx.QueryRowContext(r.Context(), `select id, announcement_mail from memberships where user_id = $1 for update`, s.UserID).Scan(&id, &before.AnnouncementMail)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil, httpx.Forbidden()
	}
	if err != nil {
		return 0, nil, err
	}
	after := settings{in.AnnouncementMail.V}
	if after == before {
		return http.StatusOK, after, nil
	}
	if _, err := tx.ExecContext(r.Context(), `update memberships set announcement_mail = $2 where id = $1`, id, after.AnnouncementMail); err != nil {
		return 0, nil, err
	}
	if err := audit.Write(r.Context(), tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "membership.update", EntityType: "membership", EntityID: id, Before: before, After: after}); err != nil {
		return 0, nil, err
	}
	return http.StatusOK, after, nil
}
