package content

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/blob"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

// ServeFile writes the Spec 6 download response for a committed blob (attachment, nosniff,
// sandbox CSP, ETag "<sha256>", private no-store, http.ServeContent with modTime).
// A row without a blob is a logged 500, never a 404.
func ServeFile(w http.ResponseWriter, r *http.Request, files blob.Dir, universityID, id, fileName, contentType, sha256 string, modTime time.Time) error {
	f, err := files.Open(universityID, id)
	if err != nil {
		return fmt.Errorf("file %s: open: %w", id, err)
	}
	defer f.Close()
	h := w.Header()
	h.Set("Content-Type", contentType)
	h.Set("Content-Disposition", contentDisposition(fileName))
	h.Set("ETag", `"`+sha256+`"`)
	h.Set("Cache-Control", "private, no-store")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Content-Security-Policy", "sandbox")
	http.ServeContent(w, r, "", modTime, f)
	return nil
}

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
			return ServeFile(w, r, files, s.UniversityID, m.ID, m.FileName, m.ContentType, m.sha256, m.CreatedAt)
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
