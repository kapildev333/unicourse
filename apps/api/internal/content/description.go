package content

import (
	"database/sql"
	"net/http"

	"github.com/kapildev333/unicourse/apps/api/internal/audit"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/course"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
)

type descriptionAudit struct {
	Description string `json:"description"`
}

func putDescription(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	id := r.PathValue("id")
	if err := sectionExists(r, tx, id, false); err != nil {
		return 0, nil, err
	}
	if err := allow(authz.CanEditContent(r.Context(), tx, s, id)); err != nil {
		return 0, nil, err
	}
	var in struct {
		Description *string `json:"description"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	if in.Description == nil {
		return 0, nil, httpx.Validation(httpx.FieldError{Field: "description", Message: "is required"})
	}
	v, fe := optText("description", *in.Description, 10000)
	if fe != nil {
		return 0, nil, httpx.Validation(*fe)
	}
	var before string
	if err := tx.QueryRowContext(r.Context(), `select description from sections where id = $1 for update`, id).Scan(&before); err != nil {
		return 0, nil, err
	}
	if _, err := tx.ExecContext(r.Context(), `update sections set description = $2 where id = $1`, id, v); err != nil {
		return 0, nil, err
	}
	if err := audit.Write(r.Context(), tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "section.update", EntityType: "section", EntityID: id,
		Before: descriptionAudit{before}, After: descriptionAudit{v}}); err != nil {
		return 0, nil, err
	}
	sec, err := course.ReadSection(r, tx, id)
	return http.StatusOK, sec, err
}
