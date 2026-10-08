package assignment

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"mime/multipart"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestFileUpload(t *testing.T) {
	e := newEnv(t)
	a := e.open()
	// the allowlist, empty files and non-multipart bodies
	for _, name := range []string{"evil.exe", "noext", "a.php"} {
		st, raw := e.upload(e.Stud, a, name, strings.NewReader("x"))
		if code, f := errDetails(raw); st != 400 || code != "validation_failed" || len(f) != 1 || f[0] != "file" {
			t.Errorf("%s: %d %s", name, st, raw)
		}
	}
	if st, raw := e.upload(e.Stud, a, "a.txt", strings.NewReader("")); st != 400 || !strings.Contains(string(raw), "is empty") {
		t.Errorf("empty: %d %s", st, raw)
	}
	if st, _ := e.call(e.Stud, "POST", e.subPath(a, "/me/files"), map[string]any{}); st != 400 {
		t.Errorf("json body: %d", st)
	}
	if st, raw := e.uploadParts(e.Stud, e.subPath(a, "/me/files"), func(mw *multipart.Writer) { mw.WriteField("x", "y") }); st != 400 || !strings.Contains(string(raw), "is required") {
		t.Errorf("no file: %d %s", st, raw)
	}
	if e.count(`select count(*) from submissions`) != 0 {
		t.Fatal("a rejected upload created a submission")
	}
	// the base name is cleaned, the type comes from the extension, the submission row is created
	st, raw := e.upload(e.Stud, a, `..\dir/Report.PDF`, strings.NewReader("pdf"))
	m := jmap(t, raw)
	files := m["files"].([]any)
	if st != 201 || len(files) != 1 || m["body"] != "" {
		t.Fatalf("upload: %d %s", st, raw)
	}
	f := files[0].(map[string]any)
	if f["fileName"] != "Report.PDF" || f["contentType"] != "application/pdf" || f["sizeBytes"] != 3.0 || f["position"] != 1.0 {
		t.Fatalf("%s", raw)
	}
	if _, err := os.Stat(e.blobPath(f["id"].(string))); err != nil {
		t.Fatal(err)
	}
	if tmp, _ := os.ReadDir(e.D.Files.Root + "/" + e.Fx.UniversityA + "/.tmp"); len(tmp) != 0 {
		t.Fatalf("temp files left: %d", len(tmp))
	}
	// the 11th file
	for i := 2; i <= 10; i++ {
		if st, raw := e.upload(e.Stud, a, "f"+strconv.Itoa(i)+".txt", strings.NewReader("x")); st != 201 {
			t.Fatalf("file %d: %d %s", i, st, raw)
		}
	}
	st, raw = e.upload(e.Stud, a, "f11.txt", strings.NewReader("x"))
	if code, fl := errDetails(raw); st != 400 || code != "validation_failed" || len(fl) != 1 || fl[0] != "file" || !strings.Contains(string(raw), "a submission holds at most 10 files") {
		t.Fatalf("11th: %d %s", st, raw)
	}
	if e.count(`select count(*) from submission_files`) != 10 {
		t.Fatal("11th file stored")
	}
	if ents, _ := os.ReadDir(e.D.Files.Root + "/" + e.Fx.UniversityA); len(ents) != 11 { // 10 blobs and .tmp
		t.Fatalf("blob dir holds %d entries", len(ents))
	}
}

func TestFileUploadLimit(t *testing.T) {
	e := newEnvLimit(t, 1000)
	a := e.open()
	if st, raw := e.upload(e.Stud, a, "ok.txt", bytes.NewReader(make([]byte, 1000))); st != 201 {
		t.Fatalf("at the limit: %d %s", st, raw)
	}
	st, raw := e.upload(e.Stud, a, "big.txt", bytes.NewReader(make([]byte, 1001)))
	if st != 400 || !strings.Contains(string(raw), "is larger than 1000 bytes") {
		t.Fatalf("over the limit: %d %s", st, raw)
	}
	if e.count(`select count(*) from submission_files`) != 1 {
		t.Fatal("over-limit file stored")
	}
}

func TestFileUploadAfterClose(t *testing.T) {
	e := newEnv(t)
	up, closed := e.assignment("1 hour", "2 hours"), e.assignment("-2 hours", "-1 hour")
	for id, msg := range map[string]string{up: "not open yet", closed: "closed"} {
		st, raw := e.upload(e.Stud, id, "a.txt", strings.NewReader("x"))
		if st != 409 || !strings.Contains(string(raw), msg) {
			t.Errorf("%s: %d %s", msg, st, raw)
		}
	}
	if e.count(`select count(*) from submissions`) != 0 {
		t.Error("row written")
	}
	// a delete after the close is a 409 and keeps the blob
	a := e.open()
	_, raw := e.upload(e.Stud, a, "a.txt", strings.NewReader("x"))
	id := fileIDs(t, raw)[0]
	e.setWindow(a, "-2 hours", "-1 hour")
	if st, _ := e.call(e.Stud, "DELETE", e.subPath(a, "/me/files/"+id), nil); st != 409 {
		t.Errorf("delete after close: %d", st)
	}
	if _, err := os.Stat(e.blobPath(id)); err != nil {
		t.Errorf("blob removed by a rejected delete: %v", err)
	}
}

func TestFileRemove(t *testing.T) {
	e := newEnv(t)
	a := e.open()
	e.put(e.Stud, a, "text")
	var ids []string
	for _, n := range []string{"a.txt", "b.txt", "c.txt"} {
		_, raw := e.upload(e.Stud, a, n, strings.NewReader(n))
		ids = fileIDs(t, raw)
	}
	if st, _ := e.call(e.Stud, "DELETE", e.subPath(a, "/me/files/"+ids[0]), nil); st != 204 {
		t.Fatalf("delete: %d", st)
	}
	if _, err := os.Stat(e.blobPath(ids[0])); !os.IsNotExist(err) {
		t.Errorf("blob still there: %v", err)
	}
	if _, err := os.Stat(e.blobPath(ids[1])); err != nil {
		t.Errorf("other blob gone: %v", err)
	}
	_, raw := e.call(e.Stud, "GET", e.subPath(a, "/me"), nil)
	fs := jmap(t, raw)["files"].([]any)
	if len(fs) != 2 || fs[0].(map[string]any)["fileName"] != "b.txt" || fs[0].(map[string]any)["position"] != 1.0 || fs[1].(map[string]any)["position"] != 2.0 {
		t.Fatalf("renumber: %s", raw)
	}
	// 404s: gone, malformed, someone else's file
	_, raw2 := e.upload(e.Stud2, a, "s2.txt", strings.NewReader("x"))
	for _, id := range []string{ids[0], "x", fileIDs(t, raw2)[0]} {
		if st, _ := e.call(e.Stud, "DELETE", e.subPath(a, "/me/files/"+id), nil); st != 404 {
			t.Errorf("%s: %d", id, st)
		}
	}
	// a rolled-back delete keeps the blob: the file belongs to another assignment's path
	other := e.open()
	if st, _ := e.call(e.Stud, "DELETE", e.subPath(other, "/me/files/"+ids[1]), nil); st != 404 {
		t.Errorf("other assignment: %d", st)
	}
	if _, err := os.Stat(e.blobPath(ids[1])); err != nil {
		t.Errorf("blob lost: %v", err)
	}
}

func TestFileDownload(t *testing.T) {
	e := newEnv(t)
	a := e.open()
	content := "hello submission bytes"
	_, raw := e.upload(e.Stud, a, "résumé (v2).txt", strings.NewReader(content))
	id := fileIDs(t, raw)[0]
	sum := sha256.Sum256([]byte(content))
	etag := `"` + hex.EncodeToString(sum[:]) + `"`
	path := e.subPath(a, "/"+e.UserID["stud@a.edu"]+"/files/"+id+"/download")
	for _, c := range []struct {
		name string
		path string
	}{{"owner as me", e.subPath(a, "/me/files/"+id+"/download")}, {"inst by id", path}} {
		cookie := e.Inst
		if c.name == "owner as me" {
			cookie = e.Stud
		}
		res, body := e.get(cookie, c.path, nil)
		h := res.Header
		if res.StatusCode != 200 || string(body) != content || h.Get("Content-Type") != "text/plain; charset=utf-8" ||
			h.Get("Content-Disposition") != "attachment; filename*=UTF-8''r%C3%A9sum%C3%A9%20%28v2%29.txt" || h.Get("ETag") != etag ||
			h.Get("X-Content-Type-Options") != "nosniff" || h.Get("Content-Security-Policy") != "sandbox" || h.Get("Cache-Control") != "private, no-store" {
			t.Errorf("%s: %d %v %q", c.name, res.StatusCode, h, body)
		}
	}
	res, body := e.get(e.Inst, path, map[string]string{"Range": "bytes=6-15"})
	if res.StatusCode != 206 || string(body) != content[6:16] || res.Header.Get("Content-Range") != "bytes 6-15/"+strconv.Itoa(len(content)) {
		t.Errorf("range: %d %q %v", res.StatusCode, body, res.Header)
	}
	if res, _ = e.get(e.Inst, path, map[string]string{"If-None-Match": etag}); res.StatusCode != 304 {
		t.Errorf("etag: %d", res.StatusCode)
	}
	// download works after the window closed; unknown, malformed and foreign file ids are 404
	e.setWindow(a, "-2 hours", "-1 hour")
	if res, _ = e.get(e.Stud, e.subPath(a, "/me/files/"+id+"/download"), nil); res.StatusCode != 200 {
		t.Errorf("after close: %d", res.StatusCode)
	}
	for _, p := range []string{
		e.subPath(a, "/me/files/00000000-0000-0000-0000-000000000000/download"),
		e.subPath(a, "/me/files/x/download"),
		e.subPath(a, "/"+e.UserID["stud2@a.edu"]+"/files/"+id+"/download"), // stud2 has no submission
		e.subPath(a, "/x/files/"+id+"/download"),
	} {
		if res, _ := e.get(e.Inst, p, nil); res.StatusCode != 404 {
			t.Errorf("%s: %d", p, res.StatusCode)
		}
	}
	if _, b := e.get(e.Inst, e.subPath(a, "/x/files/"+id+"/download"), nil); !strings.Contains(string(b), "file not found") {
		t.Errorf("non-uuid user: %s", b)
	}
	// a row without a blob is a logged 500, never a 404
	os.Remove(e.blobPath(id))
	if res, _ = e.get(e.Inst, path, nil); res.StatusCode != 500 {
		t.Errorf("missing blob: %d", res.StatusCode)
	}
}

func TestFileCascadeRemovesBlobs(t *testing.T) {
	e := newEnv(t)
	blobs := func(a string) (ids []string) {
		e.put(e.Stud, a, "t")
		e.put(e.Stud2, a, "t")
		for _, c := range []string{"stud", "stud2", "stud"} {
			_, raw := e.upload(e.as(c), a, "a.txt", strings.NewReader("x"))
			fs := fileIDs(t, raw)
			ids = append(ids, fs[len(fs)-1])
		}
		return
	}
	gone := func(ids []string) bool {
		for _, id := range ids {
			if _, err := os.Stat(e.blobPath(id)); !os.IsNotExist(err) {
				return false
			}
		}
		return true
	}
	// assessment delete
	a1 := e.open()
	ids := blobs(a1)
	if len(ids) != 3 {
		t.Fatalf("ids %v", ids)
	}
	if st, raw := e.call(e.TA, "DELETE", "/sections/"+e.SectionA+"/assessments/"+a1, nil); st != 204 || len(raw) != 0 {
		t.Fatalf("delete: %d %s", st, raw)
	}
	if !gone(ids) || e.count(`select count(*) from submission_files`) != 0 {
		t.Fatal("blobs or rows left after the assessment delete")
	}
	// section delete
	a2 := e.open()
	ids = blobs(a2)
	if st, raw := e.call(e.Admin, "DELETE", "/sections/"+e.SectionA, nil); st != 204 {
		t.Fatalf("section delete: %d %s", st, raw)
	}
	if !gone(ids) || e.count(`select count(*) from submission_files`) != 0 {
		t.Fatal("blobs or rows left after the section delete")
	}
}

func TestSubmissionAudit(t *testing.T) {
	e := newEnv(t)
	a := e.open()
	e.put(e.Stud, a, "abc")
	_, raw := e.upload(e.Stud, a, "a.txt", strings.NewReader("x"))
	id := fileIDs(t, raw)[0]
	e.put(e.Stud, a, "abcd")
	e.call(e.Stud, "DELETE", e.subPath(a, "/me/files/"+id), nil)
	n, _, _ := e.audit("submission.update", a)
	if n != 4 {
		t.Fatalf("%d audit rows, want 4", n)
	}
	// the last write: the file is gone, the body length 4
	var got string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select after::text from audit_log where action = 'submission.update' and entity_id = $1 order by created_at desc, id desc limit 1`, a).Scan(&got)
	})
	if !strings.Contains(got, `"bodyLength": 4`) && !strings.Contains(got, `"bodyLength":4`) || !strings.Contains(got, `"fileNames": []`) && !strings.Contains(got, `"fileNames":[]`) || !strings.Contains(got, e.UserID["stud@a.edu"]) {
		t.Fatalf("after %s", got)
	}
	var actor string
	e.tx(e.Fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select distinct actor_user_id::text from audit_log where action = 'submission.update'`).Scan(&actor)
	})
	if actor != e.UserID["stud@a.edu"] {
		t.Fatalf("actor %s", actor)
	}
}

// TestFileUploadPhaseTwoFailure closes the window while the file streams (after phase 1 passed): the
// second transaction answers 409 and leaves no temp file, blob or row.
func TestFileUploadPhaseTwoFailure(t *testing.T) {
	e := newEnv(t)
	a := e.open()
	st, raw := e.uploadParts(e.Stud, e.subPath(a, "/me/files"), func(mw *multipart.Writer) {
		w, err := mw.CreateFormFile("file", "a.txt")
		if err != nil {
			return
		}
		w.Write([]byte("first chunk"))
		time.Sleep(500 * time.Millisecond) // phase 1 has long passed
		e.setWindow(a, "-2 hours", "-1 hour")
		w.Write([]byte("second chunk"))
	})
	if st != 409 {
		t.Fatalf("%d %s", st, raw)
	}
	if tmp, _ := os.ReadDir(e.D.Files.Root + "/" + e.Fx.UniversityA + "/.tmp"); len(tmp) != 0 {
		t.Fatalf("temp files left: %d", len(tmp))
	}
	if e.count(`select count(*) from submissions`) != 0 || e.count(`select count(*) from submission_files`) != 0 {
		t.Fatal("rows written")
	}
}
