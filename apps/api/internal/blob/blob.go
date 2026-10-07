// Package blob stores material files under one directory, <root>/<university_id>/<material_id>; it is the only code that touches UNICOURSE_FILES_DIR.
package blob

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// Dir is the files directory. The zero value is valid to hold (route registration, tests that never upload)
// but every method fails on it.
type Dir struct{ Root string }

var (
	ErrBadID = errors.New("blob: id is not a uuid")
	errNoDir = errors.New("blob: no files directory")
	uuidRe   = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

// New returns the directory at root (made absolute) after checking it exists, is a directory and is
// writable (creates and removes a probe file). Errors name UNICOURSE_FILES_DIR.
func New(root string) (Dir, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return Dir{}, fmt.Errorf("UNICOURSE_FILES_DIR %s: %w", root, err)
	}
	st, err := os.Stat(abs)
	if errors.Is(err, fs.ErrNotExist) {
		return Dir{}, fmt.Errorf("UNICOURSE_FILES_DIR %s does not exist", abs)
	}
	if err != nil {
		return Dir{}, fmt.Errorf("UNICOURSE_FILES_DIR %s: %w", abs, err)
	}
	if !st.IsDir() {
		return Dir{}, fmt.Errorf("UNICOURSE_FILES_DIR %s is not a directory", abs)
	}
	f, err := os.CreateTemp(abs, ".write-check-*")
	if err != nil {
		return Dir{}, fmt.Errorf("UNICOURSE_FILES_DIR %s is not writable: %w", abs, err)
	}
	f.Close()
	os.Remove(f.Name())
	return Dir{Root: abs}, nil
}

// Temp is an uploaded file not yet committed. Remove it on every path that does not Commit it.
type Temp struct {
	SHA256 string // lowercase hex
	Size   int64
	path   string
}

func (d Dir) check(ids ...string) error {
	if d.Root == "" {
		return errNoDir
	}
	for _, id := range ids {
		if !uuidRe.MatchString(id) {
			return fmt.Errorf("%w: %q", ErrBadID, id)
		}
	}
	return nil
}

// Put streams r into <root>/<universityID>/.tmp/<random> while hashing it, fsyncs and closes it.
// On any error the temp file is already removed. Read errors from r are returned wrapped (errors.Is/As work).
func (d Dir) Put(universityID string, r io.Reader) (Temp, error) {
	if err := d.check(universityID); err != nil {
		return Temp{}, err
	}
	dir := filepath.Join(d.Root, universityID, ".tmp")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return Temp{}, fmt.Errorf("blob: %w", err)
	}
	f, err := os.CreateTemp(dir, "up-*")
	if err != nil {
		return Temp{}, fmt.Errorf("blob: %w", err)
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), r)
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(f.Name())
		return Temp{}, fmt.Errorf("blob: %w", err)
	}
	return Temp{SHA256: hex.EncodeToString(h.Sum(nil)), Size: n, path: f.Name()}, nil
}

// Remove deletes an uncommitted temp file; errors are ignored (a leftover temp is harmless).
func (t Temp) Remove() {
	if t.path != "" {
		os.Remove(t.path)
	}
}

// Commit renames the temp file to <root>/<universityID>/<id> and fsyncs <root>/<universityID>, so a
// committed row never points at a rename lost to a power cut.
func (d Dir) Commit(t Temp, universityID, id string) error {
	if err := d.check(universityID, id); err != nil {
		return err
	}
	dir := filepath.Join(d.Root, universityID)
	final := filepath.Join(dir, id)
	if err := os.Rename(t.path, final); err != nil {
		return err
	}
	if err := syncDir(dir); err != nil {
		os.Remove(final) // the temp is gone after the rename; leave neither file behind
		return err
	}
	return nil
}

// syncDir is a variable so a test can fail it.
var syncDir = func(dir string) error {
	df, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer df.Close()
	return df.Sync()
}

// Open opens a committed blob; a missing blob is an error matching fs.ErrNotExist.
func (d Dir) Open(universityID, id string) (*os.File, error) {
	if err := d.check(universityID, id); err != nil {
		return nil, err
	}
	return os.Open(filepath.Join(d.Root, universityID, id))
}

// Remove deletes a committed blob; a missing blob is not an error.
func (d Dir) Remove(universityID, id string) error {
	if err := d.check(universityID, id); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(d.Root, universityID, id)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// Discard removes each blob, best effort, logging failures. For after-commit cleanup.
func (d Dir) Discard(universityID string, ids ...string) {
	for _, id := range ids {
		if err := d.Remove(universityID, id); err != nil {
			slog.Error("blob remove failed", "universityId", universityID, "materialId", id, "err", err)
		}
	}
}

// Walk calls fn for every committed blob: each regular file whose name is a uuid inside a directory
// whose name is a uuid, directly under root. Anything else (.tmp, stray files) is skipped.
// It stops at, and returns, the first error from fn or the file system.
func (d Dir) Walk(fn func(universityID, id string, modTime time.Time) error) error {
	if d.Root == "" {
		return errNoDir
	}
	unis, err := os.ReadDir(d.Root)
	if err != nil {
		return err
	}
	for _, u := range unis {
		if !u.IsDir() || !uuidRe.MatchString(u.Name()) {
			continue
		}
		files, err := os.ReadDir(filepath.Join(d.Root, u.Name()))
		if err != nil {
			return err
		}
		for _, f := range files {
			if !f.Type().IsRegular() || !uuidRe.MatchString(f.Name()) {
				continue
			}
			info, err := f.Info()
			if err != nil {
				return err
			}
			if err := fn(u.Name(), f.Name(), info.ModTime()); err != nil {
				return err
			}
		}
	}
	return nil
}
