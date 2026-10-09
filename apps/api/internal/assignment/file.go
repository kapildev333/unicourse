package assignment

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/content"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

var errTooMany = httpx.Validation(httpx.FieldError{Field: "file", Message: "a submission holds at most 10 files"})

// fileCount is the number of files of the caller's submission (0 when none).
func fileCount(r *http.Request, tx *sql.Tx, assessmentID, userID string) (n int, err error) {
	err = tx.QueryRowContext(r.Context(), `select count(*) from submission_files f join submissions s on s.id = f.submission_id
		where s.assessment_id = $1 and s.user_id = $2`, assessmentID, userID).Scan(&n)
	return
}

// checkWrite is the shared prefix of the student file writes: section, assignment, enrolled, window.
func checkWrite(r *http.Request, tx *sql.Tx, s authz.Scope, lock string) (work, error) {
	secID := r.PathValue("id")
	if err := sectionExists(r, tx, secID); err != nil {
		return work{}, err
	}
	w, err := loadWork(r, tx, secID, r.PathValue("assessmentId"), lock)
	if err != nil {
		return w, err
	}
	return w, mustEnroll(r, tx, s, secID)
}

// uploadFile is phase 1 of an upload (checks in the request transaction); phase 2 streams the file with
// no transaction open and stores it in a second one, as materials (D96).
func uploadFile(d tenant.Deps) tenant.Handler {
	return func(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
		w, err := checkWrite(r, tx, s, "for share")
		if err != nil {
			return 0, nil, err
		}
		if err := w.window(); err != nil {
			return 0, nil, err
		}
		n, err := fileCount(r, tx, w.ID, s.UserID)
		if err != nil {
			return 0, nil, err
		}
		if n >= maxFiles {
			return 0, nil, errTooMany
		}
		return http.StatusCreated, tenant.After(func(rw http.ResponseWriter, r *http.Request) error {
			return receive(d, s, rw, r)
		}), nil
	}
}

func receive(d tenant.Deps, s authz.Scope, w http.ResponseWriter, r *http.Request) error {
	up, err := content.ReceiveFile(d.Files, d.UploadTimeout, s.UniversityID, w, r, nil)
	if err != nil {
		return err
	}
	var (
		id        string
		out       Submission
		committed bool
		wk        work
	)
	err = pg.WithTx(r.Context(), d.DB, s.UniversityID, func(tx *sql.Tx) error {
		ctx := r.Context()
		var err error
		if wk, err = checkWrite(r, tx, s, "for share"); err != nil {
			return err
		}
		if err := wk.window(); err != nil {
			return err
		}
		// the upsert also locks the submission row, so concurrent uploads count one at a time
		var subID string
		if err := tx.QueryRowContext(ctx, `insert into submissions (university_id, assessment_id, user_id) values ($1, $2, $3)
			on conflict (university_id, assessment_id, user_id) do update set submitted_at = now() returning id`,
			s.UniversityID, wk.ID, s.UserID).Scan(&subID); err != nil {
			return err
		}
		var n int
		if err := tx.QueryRowContext(ctx, `select count(*) from submission_files where submission_id = $1`, subID).Scan(&n); err != nil {
			return err
		}
		if n >= maxFiles {
			return errTooMany
		}
		if err := tx.QueryRowContext(ctx, `insert into submission_files (university_id, submission_id, file_name, content_type, size_bytes, sha256, position)
			values ($1, $2, $3, $4, $5, $6, $7) returning id`,
			s.UniversityID, subID, up.FileName, up.ContentType, up.Temp.Size, up.Temp.SHA256, n+1).Scan(&id); err != nil {
			return err
		}
		if err := auditUpdate(ctx, tx, s, wk.ID); err != nil {
			return err
		}
		if out, err = loadSubmission(ctx, tx, wk.ID, s.UserID, ""); err != nil {
			return err
		}
		if err := d.Files.Commit(up.Temp, s.UniversityID, id); err != nil {
			return err
		}
		committed = true // last statement before WithTx commits
		return nil
	})
	if err != nil {
		if committed {
			d.Files.Discard(s.UniversityID, id)
		} else {
			up.Temp.Remove()
		}
		return err
	}
	httpx.WriteJSON(w, http.StatusCreated, own(out, wk))
	return nil
}

func deleteFile(d tenant.Deps) tenant.Handler {
	return func(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
		ctx := r.Context()
		w, err := checkWrite(r, tx, s, "for share")
		if err != nil {
			return 0, nil, err
		}
		sub, err := loadSubmission(ctx, tx, w.ID, s.UserID, "for update")
		if err != nil {
			return 0, nil, fileErr(err)
		}
		fileID := r.PathValue("fileId")
		found := false
		for _, f := range sub.Files {
			found = found || f.ID == fileID
		}
		if !found {
			return 0, nil, httpx.NotFound("file")
		}
		if err := w.window(); err != nil {
			return 0, nil, err
		}
		if _, err := tx.ExecContext(ctx, `delete from submission_files where id = $1`, fileID); err != nil {
			return 0, nil, err
		}
		if _, err := tx.ExecContext(ctx, `update submission_files f set position = o.n
			from (select id, row_number() over (order by position, id) as n from submission_files where submission_id = $1) o
			where f.id = o.id and f.position <> o.n`, sub.id); err != nil {
			return 0, nil, err
		}
		if _, err := tx.ExecContext(ctx, `update submissions set submitted_at = now() where id = $1`, sub.id); err != nil {
			return 0, nil, err
		}
		// the "text or a file" rule: an emptied submission goes away (its scores row, if any, stays)
		if _, err := tx.ExecContext(ctx, `delete from submissions where id = $1 and body = '' and not exists (select 1 from submission_files where submission_id = $1)`, sub.id); err != nil {
			return 0, nil, err
		}
		if err := auditUpdate(ctx, tx, s, w.ID); err != nil {
			return 0, nil, err
		}
		return http.StatusNoContent, tenant.After(func(rw http.ResponseWriter, _ *http.Request) error {
			d.Files.Discard(s.UniversityID, fileID)
			rw.WriteHeader(http.StatusNoContent)
			return nil
		}), nil
	}
}

func downloadFile(d tenant.Deps) tenant.Handler {
	return func(r *http.Request, tx *sql.Tx, s authz.Scope) (int, any, error) {
		secID := r.PathValue("id")
		if err := sectionExists(r, tx, secID); err != nil {
			return 0, nil, err
		}
		w, err := loadWork(r, tx, secID, r.PathValue("assessmentId"), "")
		if err != nil {
			return 0, nil, err
		}
		userID, _, err := target(r, tx, s, secID)
		if err != nil {
			return 0, nil, fileErr(err)
		}
		sub, err := loadSubmission(r.Context(), tx, w.ID, userID, "")
		if err != nil {
			return 0, nil, fileErr(err)
		}
		for _, f := range sub.Files {
			if f.ID == r.PathValue("fileId") {
				return http.StatusOK, tenant.After(func(rw http.ResponseWriter, r *http.Request) error {
					return content.ServeFile(rw, r, d.Files, s.UniversityID, f.ID, f.FileName, f.ContentType, f.sha256, f.UploadedAt)
				}), nil
			}
		}
		return 0, nil, httpx.NotFound("file")
	}
}

// fileErr turns the "no submission" 404 into the 404 of the file; other errors pass.
func fileErr(err error) error {
	if he := new(httpx.Error); errors.As(err, &he) && he.Status == http.StatusNotFound {
		return httpx.NotFound("file")
	}
	return err
}
