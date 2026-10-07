package content

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/blob"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

func downloadMaterial(files blob.Dir) tenant.Handler {
	return func(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
		secID := r.PathValue("id")
		if err := sectionExists(r, tx, secID, false); err != nil {
			return 0, nil, err
		}
		m, err := loadMaterial(r, tx, secID, r.PathValue("materialId"))
		if err != nil {
			return 0, nil, err
		}
		if err := allow(authz.CanReadContent(r.Context(), tx, s, secID)); err != nil {
			return 0, nil, err
		}
		return http.StatusOK, tenant.After(func(w http.ResponseWriter, r *http.Request) error {
			// a row without a blob is a logged 500, never a 404
			f, err := files.Open(s.UniversityID, m.ID)
			if err != nil {
				return fmt.Errorf("material %s: open file: %w", m.ID, err)
			}
			defer f.Close()
			h := w.Header()
			h.Set("Content-Type", m.ContentType)
			h.Set("Content-Disposition", contentDisposition(m.FileName))
			h.Set("ETag", `"`+m.sha256+`"`)
			h.Set("Cache-Control", "private, no-store")
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("Content-Security-Policy", "sandbox")
			http.ServeContent(w, r, "", m.CreatedAt, f)
			return nil
		}), nil
	}
}

// contentDisposition is an attachment with the RFC 8187 encoded name: every byte outside attr-char is %XX.
func contentDisposition(name string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	b.WriteString("attachment; filename*=UTF-8''")
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || strings.IndexByte("!#$&+-.^_`|~", c) >= 0 {
			b.WriteByte(c)
		} else {
			b.WriteByte('%')
			b.WriteByte(hex[c>>4])
			b.WriteByte(hex[c&15])
		}
	}
	return b.String()
}
