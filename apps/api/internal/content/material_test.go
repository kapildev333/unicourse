package content

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/kapildev333/unicourse/apps/api/internal/authz"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
)

type zeros struct{}

func (zeros) Read(p []byte) (int, error) { clear(p); return len(p), nil }

func (e *env) matPath(id string) string { return "/sections/" + e.SectionA + "/materials/" + id }

func (e *env) mustUpload(c *http.Cookie, weekID, name, content, title string) (m matJSON) {
	e.T.Helper()
	st, raw := e.upload(c, weekID, name, strings.NewReader(content), title)
	if st != 201 || json.Unmarshal(raw, &m) != nil {
		e.T.Fatalf("upload %s: %d %s", name, st, raw)
	}
	return m
}

func TestUpload(t *testing.T) {
	e := newEnv(t)
	w := e.week("W1")
	pdf := "%PDF-1.4\n" + strings.Repeat("x", 4000)
	m := e.mustUpload(e.Inst, w, "lecture.pdf", pdf, "")
	if m.Title != "lecture.pdf" || m.FileName != "lecture.pdf" || m.ContentType != "application/pdf" || m.SizeBytes != int64(len(pdf)) || m.Position != 1 ||
		m.UploadedBy.UserID != e.UserID["inst@a.edu"] || m.UploadedBy.Email != "inst@a.edu" || m.WeekID != w {
		t.Fatalf("%+v", m)
	}
	if b, err := os.ReadFile(e.blobPath(m.ID)); err != nil || string(b) != pdf {
		t.Fatalf("blob: %v", err)
	}
	sum := sha256.Sum256([]byte(pdf))
	var got string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select sha256 from materials where id = $1`, m.ID).Scan(&got)
	})
	if got != hex.EncodeToString(sum[:]) || len(e.tmpFiles()) != 0 {
		t.Fatalf("sha %s, tmp %v", got, e.tmpFiles())
	}
	m2 := e.mustUpload(e.TA, w, "slides.pptx", "pptx", "Slides")
	if m2.Title != "Slides" || m2.Position != 2 || !strings.HasSuffix(m2.ContentType, "presentationml.presentation") {
		t.Fatalf("%+v", m2)
	}
	if strings.Contains(string(mustJSON(m)), "sha256") {
		t.Fatal("sha256 leaked")
	}
}

func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }

func TestUploadTypes(t *testing.T) {
	e := newEnv(t)
	w := e.week("W")
	for name, want := range map[string][2]string{
		"NOTES.PDF":      {"NOTES.PDF", "application/pdf"},
		"data.CSV":       {"data.CSV", "text/plain; charset=utf-8"},
		"readme.md":      {"readme.md", "text/plain; charset=utf-8"},
		`C:\x\deck.pptx`: {"deck.pptx", "application/vnd.openxmlformats-officedocument.presentationml.presentation"},
		"evil.html.pdf":  {"evil.html.pdf", "application/pdf"},
	} {
		m := e.mustUpload(e.Inst, w, name, "x", "")
		if m.FileName != want[0] || m.ContentType != want[1] {
			t.Errorf("%s: %+v", name, m)
		}
	}
	for _, name := range []string{"page.html", "page.HTM", "logo.svg", "app.js", "run.exe", "deck.tar.gz", "README", "x.pdf.html", "x.", ".hidden"} {
		st, raw := e.upload(e.Inst, w, name, strings.NewReader("x"), "")
		if code, f := errDetails(raw); st != 400 || code != "validation_failed" || len(f) != 1 || f[0] != "file" || !strings.Contains(string(raw), "file type not allowed") {
			t.Errorf("%s: %d %s", name, st, raw)
		}
	}
	// a part with no file name at all, and a 256-rune name
	st, raw := e.uploadParts(e.Inst, "/sections/"+e.SectionA+"/weeks/"+w+"/materials", func(mw *multipart.Writer) {
		fw, _ := mw.CreateFormField("file")
		fw.Write([]byte("x"))
	})
	if st != 400 || !strings.Contains(string(raw), "must have a file name") {
		t.Errorf("no name: %d %s", st, raw)
	}
	st, raw = e.upload(e.Inst, w, strings.Repeat("é", 252)+".pdf", strings.NewReader("x"), "")
	if st != 400 || !strings.Contains(string(raw), "at most 255 characters") {
		t.Errorf("long: %d %s", st, raw)
	}
	if n := e.count(`select count(*) from materials`); n != 5 || len(e.tmpFiles()) != 0 {
		t.Errorf("rows %d, tmp %v", n, e.tmpFiles())
	}
}

func TestUploadBoundary(t *testing.T) {
	if testing.Short() {
		t.Skip("streams 100 MB")
	}
	e := newEnv(t)
	w := e.week("W")
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	st, raw := e.upload(e.Inst, w, "big.zip", io.LimitReader(zeros{}, maxFile), "")
	runtime.ReadMemStats(&after)
	var m matJSON
	if st != 201 || json.Unmarshal(raw, &m) != nil || m.SizeBytes != maxFile {
		t.Fatalf("100 MB: %d %s", st, raw)
	}
	if grew := after.TotalAlloc - before.TotalAlloc; grew > 32<<20 {
		t.Fatalf("upload allocated %d MiB: the file was buffered", grew>>20)
	}
	if fi, err := os.Stat(e.blobPath(m.ID)); err != nil || fi.Size() != maxFile {
		t.Fatalf("blob: %v", err)
	}
	st, raw = e.upload(e.Inst, w, "bigger.zip", io.LimitReader(zeros{}, maxFile+1), "")
	if code, f := errDetails(raw); st != 400 || code != "validation_failed" || len(f) != 1 || f[0] != "file" || !strings.Contains(string(raw), "is larger than 100 MB") {
		t.Fatalf("100 MB + 1: %d %s", st, raw)
	}
	if n := e.count(`select count(*) from materials`); n != 1 || len(e.tmpFiles()) != 0 {
		t.Fatalf("rows %d, tmp %v", n, e.tmpFiles())
	}
}

func TestUploadBadRequests(t *testing.T) {
	e := newEnv(t)
	w := e.week("W")
	path := "/sections/" + e.SectionA + "/weeks/" + w + "/materials"
	expect := func(name string, st int, raw []byte, field, msg string) {
		t.Helper()
		if _, f := errDetails(raw); st != 400 || len(f) != 1 || f[0] != field || !strings.Contains(string(raw), msg) {
			t.Errorf("%s: %d %s", name, st, raw)
		}
	}
	st, raw := e.upload(e.Inst, w, "a.pdf", strings.NewReader(""), "")
	expect("empty", st, raw, "file", "is empty")
	st, raw = e.uploadParts(e.Inst, path, func(mw *multipart.Writer) { mw.WriteField("title", "t") })
	expect("no file", st, raw, "file", "is required")
	st, raw = e.uploadParts(e.Inst, path, func(mw *multipart.Writer) {
		for _, n := range []string{"a.pdf", "b.pdf"} {
			fw, _ := mw.CreateFormFile("file", n)
			fw.Write([]byte("x"))
		}
	})
	expect("two files", st, raw, "file", "must be exactly one file")
	if len(e.tmpFiles()) != 0 {
		t.Errorf("tmp %v", e.tmpFiles())
	}
	st, raw = e.call(e.Inst, "POST", path, map[string]any{"a": 1})
	expect("json", st, raw, "file", "must be a multipart/form-data upload")
	st, raw = e.upload(e.Inst, w, "a.pdf", strings.NewReader("x"), strings.Repeat("t", 201))
	expect("title", st, raw, "title", "at most 200")
	if len(e.tmpFiles()) != 0 || e.count(`select count(*) from materials`) != 0 {
		t.Errorf("leftovers: tmp %v", e.tmpFiles())
	}
	st, raw = e.uploadParts(e.Inst, path, func(mw *multipart.Writer) {
		mw.WriteField("foo", "bar")
		fw, _ := mw.CreateFormFile("file", "a.pdf")
		fw.Write([]byte("x"))
	})
	if st != 201 {
		t.Errorf("extra part: %d %s", st, raw)
	}
}

func TestUploadLimit(t *testing.T) {
	e := newEnv(t)
	w := e.week("W")
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`insert into materials (university_id, section_id, week_id, title, file_name, content_type, size_bytes, sha256, position, uploaded_by_user_id)
			select $1, $2, $3, 'T', 't.pdf', 'application/pdf', 1, repeat('a', 64), g, $4 from generate_series(1, 200) g`, e.Fx.UniversityA, e.SectionA, w, e.UserID["inst@a.edu"])
		return err
	})
	st, raw := e.upload(e.Inst, w, "a.pdf", io.LimitReader(zeros{}, 1<<20), "")
	if _, f := errDetails(raw); st != 400 || len(f) != 1 || f[0] != "materials" {
		t.Fatalf("201st: %d %s", st, raw)
	}
}

func TestUploadConcurrent(t *testing.T) {
	e := newEnv(t)
	w := e.week("W")
	var wg sync.WaitGroup
	sts := make([]int, 5)
	for i := range sts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sts[i], _ = e.upload(e.Inst, w, "a.pdf", strings.NewReader("data"), "")
		}()
	}
	wg.Wait()
	for i, st := range sts {
		if st != 201 {
			t.Fatalf("upload %d: %d", i, st)
		}
	}
	ms := e.weeks()[0].Materials
	for i, m := range ms {
		if m.Position != i+1 {
			t.Fatalf("positions %+v", ms)
		}
	}
	if ents, _ := os.ReadDir(e.D.Files.Root + "/" + e.Fx.UniversityA); len(ms) != 5 || len(ents) != 6 { // 5 blobs + .tmp
		t.Fatalf("%d materials, %d entries", len(ms), len(ents))
	}
}

func TestUploadWeekDeletedMidway(t *testing.T) {
	e := newEnv(t)
	w := e.week("W")
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error { _, err := tx.Exec(`delete from weeks where id = $1`, w); return err })
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "a.pdf")
	fw.Write([]byte("data"))
	mw.Close()
	req := httptest.NewRequest("POST", "/", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	s := authz.Scope{UniversityID: e.Fx.UniversityA, UserID: e.UserID["inst@a.edu"], Role: authz.Role("member")}
	rec := httptest.NewRecorder()
	err := receive(e.D, s, rec, req, e.SectionA, w)
	var he *httpx.Error
	if !errors.As(err, &he) || he.Status != 404 || !strings.Contains(he.Message, "week") {
		t.Fatalf("err = %v", err)
	}
	if rec.Body.Len() != 0 || len(e.tmpFiles()) != 0 || e.count(`select count(*) from materials`) != 0 {
		t.Fatalf("leftovers: body %q tmp %v", rec.Body.String(), e.tmpFiles())
	}
}

func TestMaterialRenameMove(t *testing.T) {
	e := newEnv(t)
	w1, w2 := e.week("W1"), e.week("W2")
	a, b, c := e.material(w1, "a"), e.material(w1, "b"), e.material(w1, "c")
	e.material(w2, "x")
	e.material(w2, "y")
	titles := func(w string) string {
		for _, wk := range e.weeks() {
			if wk.ID == w {
				var t []string
				for i, m := range wk.Materials {
					if m.Position != i+1 {
						e.T.Fatalf("positions not dense: %+v", wk.Materials)
					}
					t = append(t, m.Title)
				}
				return strings.Join(t, ",")
			}
		}
		return ""
	}
	patch := func(id string, body map[string]any) (int, []byte) {
		return e.call(e.Inst, "PATCH", e.matPath(id), body)
	}
	if st, raw := patch(a, map[string]any{"title": " Renamed "}); st != 200 || !strings.Contains(string(raw), `"title":"Renamed"`) || !strings.Contains(string(raw), `"fileName":"t.pdf"`) {
		t.Fatalf("rename: %d %s", st, raw)
	}
	if st, raw := patch(a, map[string]any{"position": 3}); st != 200 || titles(w1) != "b,c,Renamed" {
		t.Fatalf("1->3: %d %s %s", st, raw, titles(w1))
	}
	if st, raw := patch(c, map[string]any{"weekId": w2}); st != 200 || titles(w2) != "x,y,c" || titles(w1) != "b,Renamed" {
		t.Fatalf("move last: %d %s %s | %s", st, raw, titles(w2), titles(w1))
	}
	if st, raw := patch(b, map[string]any{"weekId": w2, "position": 1}); st != 200 || titles(w2) != "b,x,y,c" || titles(w1) != "Renamed" {
		t.Fatalf("move first: %d %s %s | %s", st, raw, titles(w2), titles(w1))
	}
	// invalid targets and positions
	var other string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`insert into weeks (university_id, section_id, title, position) values ($1, $2, 'o', 1) returning id`, e.Fx.UniversityA, e.SectionA2).Scan(&other)
	})
	for _, wid := range []string{other, "nope"} {
		st, raw := patch(a, map[string]any{"weekId": wid})
		if _, f := errDetails(raw); st != 400 || len(f) != 1 || f[0] != "weekId" || !strings.Contains(string(raw), "is not a week of this section") {
			t.Errorf("week %s: %d %s", wid, st, raw)
		}
	}
	for _, p := range []int{0, 2} {
		if st, raw := patch(a, map[string]any{"position": p}); st != 400 || !strings.Contains(string(raw), "must be between 1 and 1") {
			t.Errorf("position %d: %d %s", p, st, raw)
		}
	}
	if st, raw := patch(a, map[string]any{}); st != 400 {
		t.Errorf("{}: %d %s", st, raw)
	}
	// a full target week refuses a move
	w3 := e.week("W3")
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		_, err := tx.Exec(`insert into materials (university_id, section_id, week_id, title, file_name, content_type, size_bytes, sha256, position, uploaded_by_user_id)
			select $1, $2, $3, 'T', 't.pdf', 'application/pdf', 1, repeat('a', 64), g, $4 from generate_series(1, 200) g`, e.Fx.UniversityA, e.SectionA, w3, e.UserID["inst@a.edu"])
		return err
	})
	if st, raw := patch(a, map[string]any{"weekId": w3}); st != 400 || !strings.Contains(string(raw), "week already holds 200 files") {
		t.Errorf("full: %d %s", st, raw)
	}
	if n, b, a2 := e.audit("material.update", a); n != 2 || b.String == "" || a2.String == "" {
		t.Errorf("audit %d", n)
	}
}

func TestMaterialDeleteAndAudit(t *testing.T) {
	e := newEnv(t)
	w := e.week("W")
	m1 := e.mustUpload(e.Inst, w, "a.pdf", "zzz", "")
	m2 := e.mustUpload(e.Inst, w, "b.pdf", "yyy", "")
	if n, b, after := e.audit("material.create", m1.ID); n != 1 || b.Valid || !strings.Contains(after.String, `"sha256"`) || strings.Contains(after.String, "zzz") {
		t.Fatalf("create audit %d %v %s", n, b, after.String)
	}
	if st, raw := e.call(e.TA, "DELETE", e.matPath(m1.ID), nil); st != 204 || len(raw) != 0 {
		t.Fatalf("delete: %d %s", st, raw)
	}
	if _, err := os.Stat(e.blobPath(m1.ID)); !os.IsNotExist(err) {
		t.Fatalf("blob survived: %v", err)
	}
	if _, err := os.Stat(e.blobPath(m2.ID)); err != nil {
		t.Fatal(err)
	}
	if ms := e.weeks()[0].Materials; len(ms) != 1 || ms[0].ID != m2.ID || ms[0].Position != 1 {
		t.Fatalf("%+v", ms)
	}
	if n, b, after := e.audit("material.delete", m1.ID); n != 1 || !strings.Contains(b.String, `"sha256"`) || !strings.Contains(b.String, `"a.pdf"`) || after.Valid {
		t.Fatalf("delete audit %d %s", n, b.String)
	}
}

func TestMaterialAuthz(t *testing.T) {
	e := newEnv(t)
	edit := func(success int) [8]int { return [8]int{success, success, 403, success, success, 403, 403, 403} }
	for _, row := range []struct {
		name, method string
		do           func(c *http.Cookie) (int, []byte)
		want         [8]int
	}{
		{"upload", "POST", func(c *http.Cookie) (int, []byte) {
			return e.upload(c, e.week("u"), "a.pdf", strings.NewReader("x"), "")
		}, edit(201)},
		{"patch", "PATCH", func(c *http.Cookie) (int, []byte) {
			return e.call(c, "PATCH", e.matPath(e.material(e.week("p"), "m")), map[string]any{"title": "t"})
		}, edit(200)},
		{"delete", "DELETE", func(c *http.Cookie) (int, []byte) {
			return e.call(c, "DELETE", e.matPath(e.material(e.week("d"), "m")), nil)
		}, edit(204)},
	} {
		for i, role := range roles {
			if st, raw := row.do(e.as(role)); st != row.want[i] {
				t.Errorf("%s as %s: %d, want %d (%s)", row.name, role, st, row.want[i], raw)
			}
		}
	}
	// 404s: another section's material, malformed id, B's section id
	var other string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		var wk string
		if err := tx.QueryRow(`insert into weeks (university_id, section_id, title, position) values ($1, $2, 'o', 1) returning id`, e.Fx.UniversityA, e.SectionA2).Scan(&wk); err != nil {
			return err
		}
		return tx.QueryRow(`insert into materials (university_id, section_id, week_id, title, file_name, content_type, size_bytes, sha256, position, uploaded_by_user_id)
			values ($1, $2, $3, 'o', 't.pdf', 'application/pdf', 1, repeat('a', 64), 1, $4) returning id`, e.Fx.UniversityA, e.SectionA2, wk, e.UserID["inst@a.edu"]).Scan(&other)
	})
	for _, p := range []string{other, "x"} {
		if st, _ := e.call(e.Admin, "DELETE", e.matPath(p), nil); st != 404 {
			t.Errorf("material %s: %d", p, st)
		}
	}
	if st, _ := e.call(e.Admin, "DELETE", "/sections/"+e.SectionB()+"/materials/"+other, nil); st != 404 {
		t.Errorf("B's section: %d", st)
	}
	if st, _ := e.call(e.Admin, "POST", "/sections/"+e.SectionA+"/weeks/"+"x"+"/materials", nil); st != 404 {
		t.Errorf("bad week: %d", st)
	}
}

func TestMaterialDeleteRollbackKeepsBlob(t *testing.T) {
	e := newEnv(t)
	m := e.mustUpload(e.Inst, e.week("W"), "a.pdf", "data", "")
	// run the handler in a transaction that rolls back afterwards: it must not touch the blob itself
	rollback := errors.New("rollback")
	err := pg.WithTx(context.Background(), e.DB, e.Fx.UniversityA, func(tx *sql.Tx) error {
		r := httptest.NewRequest("DELETE", "/", nil)
		r.SetPathValue("id", e.SectionA)
		r.SetPathValue("materialId", m.ID)
		s := authz.Scope{UniversityID: e.Fx.UniversityA, UserID: e.UserID["inst@a.edu"], Role: authz.Role("member")}
		if st, body, err := deleteMaterial(e.D.Files)(r, tx, s); err != nil || st != 204 || body == nil {
			t.Fatalf("handler: %d %v %v", st, body, err)
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatal(err)
	}
	if _, err := os.Stat(e.blobPath(m.ID)); err != nil || e.count(`select count(*) from materials`) != 1 {
		t.Fatalf("blob or row lost: %v", err)
	}
}
