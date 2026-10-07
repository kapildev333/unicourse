package content

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
)

type lockedBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuf) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}
func (l *lockedBuf) String() string { l.mu.Lock(); defer l.mu.Unlock(); return l.b.String() }

func (e *env) dlPath(id string) string { return e.matPath(id) + "/download" }

func TestDownload(t *testing.T) {
	e := newEnv(t)
	w := e.week("W")
	pdf := "%PDF-1.4\nhello"
	m := e.mustUpload(e.Inst, w, "lecture.pdf", pdf, "")
	sum := sha256.Sum256([]byte(pdf))
	res, body := e.get(e.Stud, e.dlPath(m.ID), nil)
	h := res.Header
	if res.StatusCode != 200 || string(body) != pdf {
		t.Fatalf("%d %q", res.StatusCode, body)
	}
	for k, v := range map[string]string{
		"Content-Type":            "application/pdf",
		"Content-Disposition":     "attachment; filename*=UTF-8''lecture.pdf",
		"ETag":                    `"` + hex.EncodeToString(sum[:]) + `"`,
		"Cache-Control":           "private, no-store",
		"X-Content-Type-Options":  "nosniff",
		"Content-Security-Policy": "sandbox",
		"Content-Length":          "14",
	} {
		if h.Get(k) != v {
			t.Errorf("%s = %q, want %q", k, h.Get(k), v)
		}
	}
	md := e.mustUpload(e.Inst, w, "<script>.md", "# hi", "")
	res, _ = e.get(e.Stud, e.dlPath(md.ID), nil)
	if res.StatusCode != 200 || res.Header.Get("Content-Type") != "text/plain; charset=utf-8" || !strings.HasPrefix(res.Header.Get("Content-Disposition"), "attachment; ") {
		t.Fatalf("md: %d %v", res.StatusCode, res.Header)
	}
}

func TestDownloadRangeAndETag(t *testing.T) {
	e := newEnv(t)
	w := e.week("W")
	m := e.mustUpload(e.Inst, w, "a.pdf", "0123456789abcdef", "")
	res, body := e.get(e.Stud, e.dlPath(m.ID), map[string]string{"Range": "bytes=0-9"})
	if res.StatusCode != 206 || string(body) != "0123456789" || res.Header.Get("Content-Range") != "bytes 0-9/16" {
		t.Fatalf("range: %d %q %v", res.StatusCode, body, res.Header)
	}
	etag := res.Header.Get("ETag")
	res, body = e.get(e.Stud, e.dlPath(m.ID), map[string]string{"If-None-Match": etag})
	if res.StatusCode != 304 || len(body) != 0 {
		t.Fatalf("etag: %d %q", res.StatusCode, body)
	}
}

func TestDownloadMissingBlob(t *testing.T) {
	e := newEnv(t)
	buf := &lockedBuf{}
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(buf, nil)))
	t.Cleanup(func() { slog.SetDefault(old) })
	m := e.mustUpload(e.Inst, e.week("W"), "a.pdf", "x", "")
	if err := os.Remove(e.blobPath(m.ID)); err != nil {
		t.Fatal(err)
	}
	res, body := e.get(e.Stud, e.dlPath(m.ID), nil)
	if code, _ := errDetails(body); res.StatusCode != 500 || code != "internal" {
		t.Fatalf("%d %s", res.StatusCode, body)
	}
	if !strings.Contains(buf.String(), m.ID) {
		t.Fatalf("log lacks the material id: %s", buf.String())
	}
}

func TestContentDisposition(t *testing.T) {
	for in, want := range map[string]string{
		"lecture.pdf":     "lecture.pdf",
		"résumé (v2).pdf": "r%C3%A9sum%C3%A9%20%28v2%29.pdf",
		`a"b;c.txt`:       "a%22b%3Bc.txt",
	} {
		if got := contentDisposition(in); got != "attachment; filename*=UTF-8''"+want {
			t.Errorf("%q -> %q", in, got)
		}
	}
}

func TestDownloadAuthz(t *testing.T) {
	e := newEnv(t)
	m := e.mustUpload(e.Inst, e.week("W"), "a.pdf", "x", "")
	want := [8]int{200, 200, 403, 200, 200, 200, 403, 403}
	for i, role := range roles {
		if res, raw := e.get(e.as(role), e.dlPath(m.ID), nil); res.StatusCode != want[i] {
			t.Errorf("%s: %d, want %d (%s)", role, res.StatusCode, want[i], raw)
		}
	}
	var other string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		var wk string
		if err := tx.QueryRow(`insert into weeks (university_id, section_id, title, position) values ($1, $2, 'o', 1) returning id`, e.Fx.UniversityA, e.SectionA2).Scan(&wk); err != nil {
			return err
		}
		return tx.QueryRow(`insert into materials (university_id, section_id, week_id, title, file_name, content_type, size_bytes, sha256, position, uploaded_by_user_id)
			values ($1, $2, $3, 'o', 't.pdf', 'application/pdf', 1, repeat('a', 64), 1, $4) returning id`, e.Fx.UniversityA, e.SectionA2, wk, e.UserID["inst@a.edu"]).Scan(&other)
	})
	// Stud is not enrolled in SectionA2
	if res, _ := e.get(e.Stud, "/sections/"+e.SectionA2+"/materials/"+other+"/download", nil); res.StatusCode != 403 {
		t.Errorf("Stud on SectionA2: %d", res.StatusCode)
	}
	// another section's material under SectionA's path, and a malformed id
	for _, id := range []string{other, "x"} {
		if res, _ := e.get(e.Admin, e.dlPath(id), nil); res.StatusCode != 404 {
			t.Errorf("material %s: %d", id, res.StatusCode)
		}
	}
	if res, _ := e.get(e.Admin, "/sections/"+e.SectionB()+"/materials/"+m.ID+"/download", nil); res.StatusCode != 404 {
		t.Errorf("B's section: %d", res.StatusCode)
	}
}
