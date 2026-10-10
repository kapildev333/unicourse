package content

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/audit"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/notification"
	"github.com/kapildev333/unicourse/apps/api/internal/outbox"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

type announcement struct {
	ID        string    `json:"id"`
	SectionID string    `json:"sectionId"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Author    person    `json:"author"`
	Emailed   bool      `json:"emailed"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type announcementAudit struct {
	ID        string `json:"id"`
	SectionID string `json:"sectionId"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Emailed   bool   `json:"emailed"`
}

func (a announcement) audit() announcementAudit {
	return announcementAudit{a.ID, a.SectionID, a.Title, a.Body, a.Emailed}
}

const announcementCols = `a.id, a.section_id, a.title, a.body, a.author_user_id, u.display_name, u.email, a.emailed, a.created_at, a.updated_at`
const announcementFrom = ` from announcements a join users u on u.id = a.author_user_id`

func scanAnnouncement(sc interface{ Scan(...any) error }) (a announcement, err error) {
	err = sc.Scan(&a.ID, &a.SectionID, &a.Title, &a.Body, &a.Author.UserID, &a.Author.DisplayName, &a.Author.Email, &a.Emailed, &a.CreatedAt, &a.UpdatedAt)
	a.CreatedAt, a.UpdatedAt = a.CreatedAt.UTC(), a.UpdatedAt.UTC()
	return
}

// loadAnnouncement: bad uuid, no row, or another section's announcement -> 404.
func loadAnnouncement(r *http.Request, tx *sql.Tx, sectionID, id string, forUpdate bool) (announcement, error) {
	if !tenant.IsUUID(id) {
		return announcement{}, httpx.NotFound("announcement")
	}
	q := `select ` + announcementCols + announcementFrom + ` where a.id = $1 and a.section_id = $2`
	if forUpdate {
		q += ` for update of a`
	}
	a, err := scanAnnouncement(tx.QueryRowContext(r.Context(), q, id, sectionID))
	if errors.Is(err, sql.ErrNoRows) {
		return a, httpx.NotFound("announcement")
	}
	return a, err
}

func listAnnouncements(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	secID := r.PathValue("id")
	if err := sectionExists(r, tx, secID, false); err != nil {
		return 0, nil, err
	}
	if err := allow(authz.CanReadContent(r.Context(), tx, s, secID)); err != nil {
		return 0, nil, err
	}
	p, after, afterID, err := tenant.Page(r)
	if err != nil {
		return 0, nil, err
	}
	rows, err := tx.QueryContext(r.Context(), `select `+announcementCols+announcementFrom+` where a.section_id = $1
		and ($2::timestamptz is null or (a.created_at, a.id) < ($2, $3::uuid))
		order by a.created_at desc, a.id desc limit $4`, secID, after, afterID, p.Limit+1)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	out := httpx.List[announcement]{Items: []announcement{}}
	for rows.Next() {
		a, err := scanAnnouncement(rows)
		if err != nil {
			return 0, nil, err
		}
		if len(out.Items) == p.Limit {
			last := out.Items[len(out.Items)-1]
			out.NextCursor = httpx.NextCursor(last.CreatedAt, last.ID)
			break
		}
		out.Items = append(out.Items, a)
	}
	return http.StatusOK, out, rows.Err()
}

func createAnnouncement(baseURL string) tenant.Handler {
	return func(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
		secID := r.PathValue("id")
		if err := sectionExists(r, tx, secID, false); err != nil {
			return 0, nil, err
		}
		if err := allow(authz.CanEditContent(r.Context(), tx, s, secID)); err != nil {
			return 0, nil, err
		}
		var in struct {
			Title string           `json:"title"`
			Body  string           `json:"body"`
			Email tenant.Opt[bool] `json:"email"`
		}
		if err := httpx.DecodeJSON(r, &in); err != nil {
			return 0, nil, err
		}
		var d []httpx.FieldError
		title, fe := line("title", in.Title, 200)
		if fe != nil {
			d = append(d, *fe)
		}
		body, fe := tenant.TrimLen("body", in.Body, 20000)
		if fe != nil {
			d = append(d, *fe)
		}
		if in.Email.Null {
			d = append(d, httpx.FieldError{Field: "email", Message: "must not be null"})
		}
		if len(d) > 0 {
			return 0, nil, httpx.Validation(d...)
		}
		var mails []outbox.Mail
		if !in.Email.Set || in.Email.V {
			var err error
			if mails, err = announcementMails(r.Context(), tx, baseURL, r.PathValue("slug"), secID, s.UserID, title, body); err != nil {
				return 0, nil, err
			}
		}
		var id string
		if err := tx.QueryRowContext(r.Context(), `insert into announcements (university_id, section_id, author_user_id, title, body, emailed)
			values ($1, $2, $3, $4, $5, $6) returning id`, s.UniversityID, secID, s.UserID, title, body, len(mails) > 0).Scan(&id); err != nil {
			return 0, nil, err
		}
		if err := notification.Announcement(r.Context(), tx, s.UniversityID, s.UserID, secID, id); err != nil {
			return 0, nil, err
		}
		if err := outbox.Enqueue(r.Context(), tx, mails...); err != nil {
			return 0, nil, err
		}
		a, err := loadAnnouncement(r, tx, secID, id, false)
		if err != nil {
			return 0, nil, err
		}
		if err := audit.Write(r.Context(), tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "announcement.create", EntityType: "announcement", EntityID: id, After: a.audit()}); err != nil {
			return 0, nil, err
		}
		return http.StatusCreated, a, nil
	}
}

func updateAnnouncement(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	secID := r.PathValue("id")
	if err := sectionExists(r, tx, secID, false); err != nil {
		return 0, nil, err
	}
	before, err := loadAnnouncement(r, tx, secID, r.PathValue("announcementId"), true)
	if err != nil {
		return 0, nil, err
	}
	if err := allow(authz.CanEditContent(r.Context(), tx, s, secID)); err != nil {
		return 0, nil, err
	}
	var in struct {
		Title tenant.Opt[string] `json:"title"`
		Body  tenant.Opt[string] `json:"body"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	var d []httpx.FieldError
	for _, f := range []struct {
		name string
		null bool
	}{{"title", in.Title.Null}, {"body", in.Body.Null}} {
		if f.null {
			d = append(d, httpx.FieldError{Field: f.name, Message: "must not be null"})
		}
	}
	if len(d) > 0 {
		return 0, nil, httpx.Validation(d...)
	}
	if !in.Title.Set && !in.Body.Set {
		return 0, nil, tenant.NoFields()
	}
	title, body := before.Title, before.Body
	if in.Title.Set {
		var fe *httpx.FieldError
		if title, fe = line("title", in.Title.V, 200); fe != nil {
			d = append(d, *fe)
		}
	}
	if in.Body.Set {
		var fe *httpx.FieldError
		if body, fe = tenant.TrimLen("body", in.Body.V, 20000); fe != nil {
			d = append(d, *fe)
		}
	}
	if len(d) > 0 {
		return 0, nil, httpx.Validation(d...)
	}
	if _, err := tx.ExecContext(r.Context(), `update announcements set title = $2, body = $3 where id = $1`, before.ID, title, body); err != nil {
		return 0, nil, err
	}
	after, err := loadAnnouncement(r, tx, secID, before.ID, false)
	if err != nil {
		return 0, nil, err
	}
	if err := audit.Write(r.Context(), tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "announcement.update", EntityType: "announcement", EntityID: before.ID,
		Before: before.audit(), After: after.audit()}); err != nil {
		return 0, nil, err
	}
	return http.StatusOK, after, nil
}

func deleteAnnouncement(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	secID := r.PathValue("id")
	if err := sectionExists(r, tx, secID, false); err != nil {
		return 0, nil, err
	}
	before, err := loadAnnouncement(r, tx, secID, r.PathValue("announcementId"), true)
	if err != nil {
		return 0, nil, err
	}
	if err := allow(authz.CanEditContent(r.Context(), tx, s, secID)); err != nil {
		return 0, nil, err
	}
	if _, err := tx.ExecContext(r.Context(), `delete from announcements where id = $1`, before.ID); err != nil {
		return 0, nil, err
	}
	if err := audit.Write(r.Context(), tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "announcement.delete", EntityType: "announcement", EntityID: before.ID, Before: before.audit()}); err != nil {
		return 0, nil, err
	}
	return http.StatusNoContent, nil, nil
}
