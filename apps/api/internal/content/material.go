package content

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"maps"
	"mime"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/kapildev333/unicourse/apps/api/internal/audit"
	"github.com/kapildev333/unicourse/apps/api/internal/auth"
	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/blob"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

// fileTypes is the upload allowlist (spec 6 section 6): lowercase extension -> the Content-Type served.
// The only place that knows it.
var fileTypes = map[string]string{
	"pdf":  "application/pdf",
	"png":  "image/png",
	"jpg":  "image/jpeg",
	"jpeg": "image/jpeg",
	"gif":  "image/gif",
	"webp": "image/webp",
	"zip":  "application/zip",
	"ppt":  "application/vnd.ms-powerpoint",
	"pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	"doc":  "application/msword",
	"docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"xls":  "application/vnd.ms-excel",
	"xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"txt":  "text/plain; charset=utf-8",
	"md":   "text/plain; charset=utf-8",
	"csv":  "text/plain; charset=utf-8",
}

const (
	maxPerWeek = 200
)

type material struct {
	ID          string    `json:"id"`
	WeekID      string    `json:"weekId"`
	Title       string    `json:"title"`
	FileName    string    `json:"fileName"`
	ContentType string    `json:"contentType"`
	SizeBytes   int64     `json:"sizeBytes"`
	Position    int       `json:"position"`
	UploadedBy  person    `json:"uploadedBy"`
	CreatedAt   time.Time `json:"createdAt"`
	sha256      string    // ETag; not in the contract
}

type materialAudit struct {
	ID          string `json:"id"`
	SectionID   string `json:"sectionId"`
	WeekID      string `json:"weekId"`
	Title       string `json:"title"`
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	SizeBytes   int64  `json:"sizeBytes"`
	SHA256      string `json:"sha256"`
	Position    int    `json:"position"`
}

func (m material) audit(sectionID string) materialAudit {
	return materialAudit{m.ID, sectionID, m.WeekID, m.Title, m.FileName, m.ContentType, m.SizeBytes, m.sha256, m.Position}
}

const materialCols = `m.id, m.week_id, m.title, m.file_name, m.content_type, m.size_bytes, m.position, m.uploaded_by_user_id, u.display_name, u.email, m.created_at, m.sha256`
const materialFrom = ` from materials m join users u on u.id = m.uploaded_by_user_id`

func scanMaterial(sc interface{ Scan(...any) error }) (m material, err error) {
	err = sc.Scan(&m.ID, &m.WeekID, &m.Title, &m.FileName, &m.ContentType, &m.SizeBytes, &m.Position, &m.UploadedBy.UserID, &m.UploadedBy.DisplayName, &m.UploadedBy.Email, &m.CreatedAt, &m.sha256)
	m.CreatedAt = m.CreatedAt.UTC()
	return
}

// loadMaterial: bad uuid, no row or another section's material -> 404.
func loadMaterial(r *http.Request, tx *sql.Tx, sectionID, id string) (material, error) {
	if !tenant.IsUUID(id) {
		return material{}, httpx.NotFound("material")
	}
	m, err := scanMaterial(tx.QueryRowContext(r.Context(), `select `+materialCols+materialFrom+` where m.id = $1 and m.section_id = $2`, id, sectionID))
	if errors.Is(err, sql.ErrNoRows) {
		return m, httpx.NotFound("material")
	}
	return m, err
}

// renumberMaterials is renumberWeeks for the files of one week.
func renumberMaterials(ctx context.Context, tx *sql.Tx, weekID, movedID string, to, from int) error {
	var moved any
	if movedID != "" {
		moved = movedID
	}
	_, err := tx.ExecContext(ctx, `update materials m set position = o.n
		from (select id, row_number() over (order by
		        case when id = $2::uuid then $3::int * 2 + case when $3::int > $4::int then 1 else -1 end
		             else position * 2 end, id) as n
		      from materials where week_id = $1) o
		where m.id = o.id and m.position <> o.n`, weekID, moved, to, from)
	return err
}

func countMaterials(ctx context.Context, tx *sql.Tx, weekID string) (n int, err error) {
	err = tx.QueryRowContext(ctx, `select count(*) from materials where week_id = $1`, weekID).Scan(&n)
	return
}

func fileErr(msg string) *httpx.FieldError { return &httpx.FieldError{Field: "file", Message: msg} }

// fileName cleans a multipart file name to its base name and returns it with the served content type.
func fileName(raw string) (name, contentType string, fe *httpx.FieldError) {
	name = strings.TrimSpace(raw[strings.LastIndexAny(raw, "/\\")+1:])
	if name == "" || name == "." || name == ".." {
		return "", "", fileErr("must have a file name")
	}
	if strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return "", "", fileErr("file name must not contain control characters")
	}
	if utf8.RuneCountInString(name) > 255 {
		return "", "", fileErr("file name must be at most 255 characters")
	}
	ext := ""
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		ext = strings.ToLower(name[i+1:])
	}
	ct, ok := fileTypes[ext]
	if !ok {
		return "", "", fileErr("file type not allowed (allowed: " + strings.Join(slices.Sorted(maps.Keys(fileTypes)), ", ") + ")")
	}
	return name, ct, nil
}

// countErr remembers the first non-EOF read error of the request body, so a client problem is not a 500.
type countErr struct {
	r   io.Reader
	err error
}

func (c *countErr) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	if err != nil && err != io.EOF && c.err == nil {
		c.err = err
	}
	return n, err
}

// sizeText is n as "<k> MB" when it is a whole number of MiB, else "<n> bytes".
func sizeText(n int64) string {
	if n%(1<<20) == 0 {
		return strconv.FormatInt(n>>20, 10) + " MB"
	}
	return strconv.FormatInt(n, 10) + " bytes"
}

// uploadPolicy answers the upload limit and allowlist (B9): the web's copy of both is gone.
func uploadPolicy(d tenant.Deps) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		if _, err := auth.Authenticate(w, r, d.Sessions); err != nil {
			return err
		}
		httpx.WriteJSON(w, http.StatusOK, uploadPolicyBody{MaxBytes: d.Files.Limit(), AllowedExtensions: slices.Sorted(maps.Keys(fileTypes))})
		return nil
	}
}

type uploadPolicyBody struct {
	MaxBytes          int64    `json:"maxBytes"`
	AllowedExtensions []string `json:"allowedExtensions"`
}

func uploadErr(err error, limit int64) error {
	var mb *http.MaxBytesError
	if errors.As(err, &mb) {
		return httpx.Validation(*fileErr("is larger than " + sizeText(limit)))
	}
	return httpx.Validation(*fileErr("invalid multipart upload: " + err.Error()))
}

func uploadMaterial(d tenant.Deps) tenant.Handler {
	return func(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
		secID := r.PathValue("id")
		if err := sectionExists(r, tx, secID, false); err != nil {
			return 0, nil, err
		}
		wk, err := loadWeek(r, tx, secID, r.PathValue("weekId"))
		if err != nil {
			return 0, nil, err
		}
		if err := allow(authz.CanEditContent(r.Context(), tx, s, secID)); err != nil {
			return 0, nil, err
		}
		n, err := countMaterials(r.Context(), tx, wk.ID)
		if err != nil {
			return 0, nil, err
		}
		if n >= maxPerWeek {
			return 0, nil, httpx.Validation(httpx.FieldError{Field: "materials", Message: "a week holds at most 200 files"})
		}
		return http.StatusCreated, tenant.After(func(w http.ResponseWriter, r *http.Request) error {
			return receive(d, s, w, r, secID, wk.ID)
		}), nil
	}
}

// receive is phase 2 of an upload: it runs after the request transaction committed, streams the file part
// to a temp file with no transaction open, then inserts, audits and renames in a second transaction.
func receive(d tenant.Deps, s authz.Scope, w http.ResponseWriter, r *http.Request, secID, weekID string) error {
	notMultipart := httpx.Validation(*fileErr("must be a multipart/form-data upload"))
	if mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); mt != "multipart/form-data" {
		return notMultipart
	}
	limit := d.Files.Limit()
	r.Body = http.MaxBytesReader(w, r.Body, limit+1<<20) // plus 1 MB of multipart overhead (spec 6 section 5)
	mr, err := r.MultipartReader()
	if err != nil {
		return notMultipart
	}
	var (
		tmp        blob.Temp
		haveFile   bool
		name, ct   string
		title      string
		removeTemp = func() {
			if haveFile {
				tmp.Remove()
			}
		}
	)
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			removeTemp()
			return uploadErr(err, limit)
		}
		switch part.FormName() {
		case "title":
			b, err := io.ReadAll(io.LimitReader(part, 4096))
			if err != nil {
				removeTemp()
				return uploadErr(err, limit)
			}
			title = string(b)
		case "file":
			if haveFile {
				removeTemp()
				return httpx.Validation(*fileErr("must be exactly one file"))
			}
			var fe *httpx.FieldError
			if name, ct, fe = fileName(part.FileName()); fe != nil {
				return httpx.Validation(*fe)
			}
			src := &countErr{r: part}
			t, err := d.Files.Put(s.UniversityID, io.LimitReader(src, limit+1))
			if err != nil {
				if src.err != nil {
					return uploadErr(src.err, limit)
				}
				return err
			}
			tmp, haveFile = t, true
			if tmp.Size > limit {
				removeTemp()
				return httpx.Validation(*fileErr("is larger than " + sizeText(limit)))
			}
			if tmp.Size == 0 {
				removeTemp()
				return httpx.Validation(*fileErr("is empty"))
			}
		}
	}
	if !haveFile {
		return httpx.Validation(*fileErr("is required"))
	}
	if strings.TrimSpace(title) == "" {
		title = name
		if utf8.RuneCountInString(title) > 200 {
			title = string([]rune(title)[:200])
		}
	} else {
		var fe *httpx.FieldError
		if title, fe = line("title", title, 200); fe != nil {
			removeTemp()
			return httpx.Validation(*fe)
		}
	}

	var m material
	var id string
	committed := false
	err = pg.WithTx(r.Context(), d.DB, s.UniversityID, func(tx *sql.Tx) error {
		ctx := r.Context()
		if err := sectionExists(r, tx, secID, true); err != nil {
			return err
		}
		if _, err := loadWeek(r, tx, secID, weekID); err != nil {
			return err
		}
		if err := allow(authz.CanEditContent(ctx, tx, s, secID)); err != nil {
			return err
		}
		n, err := countMaterials(ctx, tx, weekID)
		if err != nil {
			return err
		}
		if n >= maxPerWeek {
			return httpx.Validation(httpx.FieldError{Field: "materials", Message: "a week holds at most 200 files"})
		}
		if err := tx.QueryRowContext(ctx, `insert into materials (university_id, section_id, week_id, title, file_name, content_type, size_bytes, sha256, position, uploaded_by_user_id)
			values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) returning id`,
			s.UniversityID, secID, weekID, title, name, ct, tmp.Size, tmp.SHA256, n+1, s.UserID).Scan(&id); err != nil {
			return err
		}
		if m, err = loadMaterial(r, tx, secID, id); err != nil {
			return err
		}
		if err := audit.Write(ctx, tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "material.create", EntityType: "material", EntityID: id, After: m.audit(secID)}); err != nil {
			return err
		}
		if err := d.Files.Commit(tmp, s.UniversityID, id); err != nil {
			return err
		}
		committed = true // last statement before WithTx commits
		return nil
	})
	if err != nil {
		if committed {
			d.Files.Discard(s.UniversityID, id)
		} else {
			tmp.Remove()
		}
		return err
	}
	httpx.WriteJSON(w, http.StatusCreated, m)
	return nil
}

func updateMaterial(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
	ctx := r.Context()
	secID := r.PathValue("id")
	if err := sectionExists(r, tx, secID, true); err != nil {
		return 0, nil, err
	}
	before, err := loadMaterial(r, tx, secID, r.PathValue("materialId"))
	if err != nil {
		return 0, nil, err
	}
	if err := allow(authz.CanEditContent(ctx, tx, s, secID)); err != nil {
		return 0, nil, err
	}
	var in struct {
		Title    tenant.Opt[string] `json:"title"`
		WeekID   tenant.Opt[string] `json:"weekId"`
		Position tenant.Opt[int]    `json:"position"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		return 0, nil, err
	}
	var d []httpx.FieldError
	for _, f := range []struct {
		name string
		null bool
	}{{"title", in.Title.Null}, {"weekId", in.WeekID.Null}, {"position", in.Position.Null}} {
		if f.null {
			d = append(d, httpx.FieldError{Field: f.name, Message: "must not be null"})
		}
	}
	if len(d) > 0 {
		return 0, nil, httpx.Validation(d...)
	}
	if !in.Title.Set && !in.WeekID.Set && !in.Position.Set {
		return 0, nil, tenant.NoFields()
	}
	title := before.Title
	if in.Title.Set {
		var fe *httpx.FieldError
		if title, fe = line("title", in.Title.V, 200); fe != nil {
			d = append(d, *fe)
		}
	}
	target := before.WeekID
	weekOK := true
	if in.WeekID.Set {
		var c int
		if tenant.IsUUID(in.WeekID.V) {
			if err := tx.QueryRowContext(ctx, `select count(*) from weeks where id = $1 and section_id = $2`, in.WeekID.V, secID).Scan(&c); err != nil {
				return 0, nil, err
			}
		}
		if c == 0 {
			weekOK = false
			d = append(d, httpx.FieldError{Field: "weekId", Message: "is not a week of this section"})
		} else {
			target = in.WeekID.V
		}
	}
	moving := target != before.WeekID
	n := 0
	if weekOK {
		if n, err = countMaterials(ctx, tx, target); err != nil {
			return 0, nil, err
		}
		if moving && n >= maxPerWeek {
			d = append(d, httpx.FieldError{Field: "weekId", Message: "week already holds 200 files"})
		}
	}
	limit := n
	if moving {
		limit = n + 1
	}
	if in.Position.Set && weekOK {
		if p := in.Position.V; p < 1 || p > limit {
			d = append(d, httpx.FieldError{Field: "position", Message: "must be between 1 and " + strconv.Itoa(limit)})
		}
	}
	if len(d) > 0 {
		return 0, nil, httpx.Validation(d...)
	}
	if _, err := tx.ExecContext(ctx, `update materials set title = $2, week_id = $3 where id = $1`, before.ID, title, target); err != nil {
		return 0, nil, err
	}
	switch {
	case moving:
		to := n + 1
		if in.Position.Set {
			to = in.Position.V
		}
		if err := renumberMaterials(ctx, tx, target, before.ID, to, n+1); err != nil {
			return 0, nil, err
		}
		if err := renumberMaterials(ctx, tx, before.WeekID, "", 0, 0); err != nil {
			return 0, nil, err
		}
	case in.Position.Set && in.Position.V != before.Position:
		if err := renumberMaterials(ctx, tx, target, before.ID, in.Position.V, before.Position); err != nil {
			return 0, nil, err
		}
	}
	after, err := loadMaterial(r, tx, secID, before.ID)
	if err != nil {
		return 0, nil, err
	}
	if err := audit.Write(ctx, tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "material.update", EntityType: "material", EntityID: before.ID, Before: before.audit(secID), After: after.audit(secID)}); err != nil {
		return 0, nil, err
	}
	return http.StatusOK, after, nil
}

func deleteMaterial(files blob.Dir) tenant.Handler {
	return func(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
		ctx := r.Context()
		secID := r.PathValue("id")
		if err := sectionExists(r, tx, secID, true); err != nil {
			return 0, nil, err
		}
		before, err := loadMaterial(r, tx, secID, r.PathValue("materialId"))
		if err != nil {
			return 0, nil, err
		}
		if err := allow(authz.CanEditContent(ctx, tx, s, secID)); err != nil {
			return 0, nil, err
		}
		if _, err := tx.ExecContext(ctx, `delete from materials where id = $1`, before.ID); err != nil {
			return 0, nil, err
		}
		if err := renumberMaterials(ctx, tx, before.WeekID, "", 0, 0); err != nil {
			return 0, nil, err
		}
		if err := audit.Write(ctx, tx, s.UniversityID, audit.Entry{ActorUserID: s.UserID, Action: "material.delete", EntityType: "material", EntityID: before.ID, Before: before.audit(secID)}); err != nil {
			return 0, nil, err
		}
		return http.StatusNoContent, noContentAfter(files, s.UniversityID, before.ID), nil
	}
}
