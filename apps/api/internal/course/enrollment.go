package course

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/audit"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

type Enrollment struct {
	UserID      string    `json:"userId"`
	Email       string    `json:"email"`
	DisplayName string    `json:"displayName"`
	CreatedAt   time.Time `json:"createdAt"`
}

type enrollmentAudit struct {
	SectionID string `json:"sectionId"`
	UserID    string `json:"userId"`
	Email     string `json:"email"`
}

func listEnrollments(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	id := r.PathValue("id")
	if err := sectionExists(r, tx, id); err != nil {
		return 0, nil, err
	}
	if err := allow(authz.CanReadRoster(r.Context(), tx, s, id)); err != nil {
		return 0, nil, err
	}
	p, after, afterID, err := tenant.Page(r)
	if err != nil {
		return 0, nil, err
	}
	rows, err := tx.QueryContext(r.Context(), `select u.id, u.email, u.display_name, e.created_at, e.id
		from enrollments e join users u on u.id = e.user_id
		where e.section_id = $4 and ($1::timestamptz is null or (e.created_at, e.id) > ($1, $2::uuid))
		order by e.created_at, e.id limit $3`, after, afterID, p.Limit+1, id)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	out := httpx.List[Enrollment]{Items: []Enrollment{}}
	var last Enrollment
	var lastID string
	for rows.Next() {
		var en Enrollment
		var rowID string
		if err := rows.Scan(&en.UserID, &en.Email, &en.DisplayName, &en.CreatedAt, &rowID); err != nil {
			return 0, nil, err
		}
		en.CreatedAt = en.CreatedAt.UTC()
		if len(out.Items) == p.Limit {
			out.NextCursor = httpx.NextCursor(last.CreatedAt, lastID)
			break
		}
		out.Items = append(out.Items, en)
		last, lastID = en, rowID
	}
	if err := rows.Err(); err != nil {
		return 0, nil, err
	}
	return http.StatusOK, out, nil
}

func deleteEnrollment(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	ctx, secID := r.Context(), r.PathValue("id")
	if err := sectionExists(r, tx, secID); err != nil {
		return 0, nil, err
	}
	if err := allow(authz.CanManageSection(ctx, tx, s, secID)); err != nil {
		return 0, nil, err
	}
	userID := r.PathValue("userId")
	if !tenant.IsUUID(userID) {
		return 0, nil, httpx.NotFound("enrollment")
	}
	a := enrollmentAudit{SectionID: secID, UserID: userID}
	var rowID string
	err := tx.QueryRowContext(ctx, `delete from enrollments e using users u
		where e.section_id = $1 and e.user_id = $2 and u.id = e.user_id returning e.id, u.email`, secID, userID).Scan(&rowID, &a.Email)
	if err == sql.ErrNoRows {
		return 0, nil, httpx.NotFound("enrollment")
	} else if err != nil {
		return 0, nil, err
	}
	if err := audit.Write(ctx, tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "enrollment.delete", EntityType: "enrollment", EntityID: rowID, Before: a}); err != nil {
		return 0, nil, err
	}
	return http.StatusNoContent, nil, nil
}
