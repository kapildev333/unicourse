package csvimport

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/testdb"
)

var emailSpec = Spec{
	Required: []string{"email"},
	Optional: []string{"displayname"},
	Key:      func(d map[string]string) string { return strings.ToLower(strings.TrimSpace(d["email"])) },
}

func parseFile(t *testing.T, name string, s Spec) ([]Record, string, error) {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return Parse(bytes.NewReader(b), s)
}

// wantValidation asserts err is a validation_failed with exactly these details.
func wantValidation(t *testing.T, err error, want ...httpx.FieldError) {
	t.Helper()
	var he *httpx.Error
	if !errors.As(err, &he) || he.Code != "validation_failed" {
		t.Fatalf("want validation_failed, got %v", err)
	}
	if !reflect.DeepEqual(he.Details, want) {
		t.Fatalf("details = %v, want %v", he.Details, want)
	}
}

func TestParseBOMCRLF(t *testing.T) {
	recs, sum, err := parseFile(t, "bom-crlf.csv", emailSpec)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile("testdata/bom-crlf.csv")
	h := sha256.Sum256(b)
	if sum != hex.EncodeToString(h[:]) {
		t.Fatalf("checksum %s not sha256 of raw bytes", sum)
	}
	want := []Record{
		{2, map[string]string{"email": "alice@a.edu", "displayname": "Alice"}},
		{3, map[string]string{"email": "bob@a.edu", "displayname": "Bob"}},
	}
	if !reflect.DeepEqual(recs, want) {
		t.Fatalf("recs = %v", recs)
	}
}

func TestParseQuoted(t *testing.T) {
	recs, _, err := parseFile(t, "quoted.csv", emailSpec)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 3 || recs[0].Data["email"] != "Doe, Jane" || recs[1].Data["email"] != "multi\nline" {
		t.Fatalf("recs = %v", recs)
	}
	if recs[0].Line != 2 || recs[1].Line != 3 || recs[2].Line != 5 {
		t.Fatalf("lines = %d %d %d, want 2 3 5", recs[0].Line, recs[1].Line, recs[2].Line)
	}
}

func TestParseHeaderOnly(t *testing.T) {
	recs, _, err := parseFile(t, "header-only.csv", emailSpec)
	if err != nil || len(recs) != 0 {
		t.Fatalf("recs=%v err=%v", recs, err)
	}
}

func TestParseSkipsBlankAndEmptyLines(t *testing.T) {
	recs, _, err := Parse(strings.NewReader("email,displayname\na@a.edu,A\n,\n\n   \n,\nb@a.edu,B\n,"), emailSpec)
	if err != nil || len(recs) != 2 || recs[1].Line != 7 {
		t.Fatalf("recs=%v err=%v", recs, err)
	}
}

func TestParseHeaderNormalisedUnknownIgnoredOptionalAbsent(t *testing.T) {
	recs, _, err := Parse(strings.NewReader(" EMAIL ,Extra\n a@a.edu ,zzz\n"), emailSpec)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"email": "a@a.edu", "displayname": ""}
	if len(recs) != 1 || !reflect.DeepEqual(recs[0].Data, want) {
		t.Fatalf("recs = %v", recs)
	}
}

func TestParseErrors(t *testing.T) {
	_, _, err := Parse(strings.NewReader("a,b\n1,2\n"), Spec{Required: []string{"email", "code"}})
	wantValidation(t, err, httpx.FieldError{Field: "file", Message: "missing column email"}, httpx.FieldError{Field: "file", Message: "missing column code"})

	_, _, err = Parse(strings.NewReader(""), emailSpec)
	wantValidation(t, err, httpx.FieldError{Field: "file", Message: "header row required"})

	s := emailSpec
	s.MaxBytes = 10
	_, _, err = Parse(strings.NewReader("email\n"+strings.Repeat("a", 20)), s)
	wantValidation(t, err, httpx.FieldError{Field: "file", Message: "file too large"})
	if _, _, err = Parse(strings.NewReader("email\nabcd"), s); err != nil { // exactly MaxBytes
		t.Fatal(err)
	}

	_, _, err = Parse(strings.NewReader("email,displayname\na@a.edu\n"), emailSpec)
	var he *httpx.Error
	if !errors.As(err, &he) || len(he.Details) != 1 || he.Details[0].Field != "file" || !strings.HasPrefix(he.Details[0].Message, "line 2: ") {
		t.Fatalf("err = %v", err)
	}
}

func TestParseDefaultKey(t *testing.T) {
	s := Spec{Required: []string{"a", "b"}}
	recs, _, err := Parse(strings.NewReader("a,b\n1,2\n1,2\n"), s)
	if err != nil {
		t.Fatal(err)
	}
	p, err := PreviewRows(context.Background(), nil, recs, "x", s, allAdd)
	if err != nil || p.Rows[1].Action != Error || p.Rows[1].Message != "duplicate of line 2" {
		t.Fatalf("p=%v err=%v", p, err)
	}
}

func allAdd(_ context.Context, _ *sql.Tx, recs []Record) ([]Verdict, error) {
	vs := make([]Verdict, len(recs))
	for i, r := range recs {
		vs[i] = Verdict{Action: Add, Ref: r.Line}
	}
	return vs, nil
}

func TestPreviewDuplicates(t *testing.T) {
	recs, sum, err := parseFile(t, "dupes.csv", emailSpec)
	if err != nil {
		t.Fatal(err)
	}
	var seen []int
	c := func(ctx context.Context, tx *sql.Tx, rs []Record) ([]Verdict, error) {
		for _, r := range rs {
			seen = append(seen, r.Line)
		}
		return allAdd(ctx, tx, rs)
	}
	p, err := PreviewRows(context.Background(), nil, recs, sum, emailSpec, c)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(seen, []int{2, 3}) {
		t.Fatalf("classifier saw %v", seen)
	}
	for i := 1; i < len(p.Rows); i++ { // Apply relies on this order
		if p.Rows[i].Line <= p.Rows[i-1].Line {
			t.Fatalf("rows not in line order: %+v", p.Rows)
		}
	}
	if len(p.Rows) != 4 || p.Checksum != sum {
		t.Fatalf("p = %+v", p)
	}
	for _, i := range []int{2, 3} {
		if r := p.Rows[i]; r.Action != Error || r.Message != "duplicate of line 2" {
			t.Fatalf("row %d = %+v", i, r)
		}
	}
	if p.Rows[3].Line != 7 {
		t.Fatalf("lines: %+v", p.Rows)
	}
	want := map[Action]int{Add: 2, Exists: 0, Invite: 0, Error: 2}
	if !reflect.DeepEqual(p.Counts, want) {
		t.Fatalf("counts = %v", p.Counts)
	}
}

func TestPreviewCountsAllKeysAndVerdictMismatch(t *testing.T) {
	p, err := PreviewRows(context.Background(), nil, nil, "x", emailSpec, allAdd)
	if err != nil || len(p.Counts) != 4 || p.Rows == nil {
		t.Fatalf("p=%+v err=%v", p, err)
	}
	recs := []Record{{2, map[string]string{"email": "a"}}}
	bad := func(context.Context, *sql.Tx, []Record) ([]Verdict, error) { return nil, nil }
	if _, err := PreviewRows(context.Background(), nil, recs, "x", emailSpec, bad); err == nil {
		t.Fatal("want error on verdict count mismatch")
	}
}

func preview(rows ...Row) Preview {
	return Preview{Checksum: "abc", Rows: rows}
}

func TestApply(t *testing.T) {
	ctx := context.Background()
	called := 0
	var got []Row
	a := func(_ context.Context, _ *sql.Tx, rows []Row) error { called++; got = rows; return nil }

	for _, sum := range []string{"", "other"} {
		err := Apply(ctx, nil, preview(Row{Line: 2, Action: Add}), sum, a)
		wantValidation(t, err, httpx.FieldError{Field: "checksum", Message: "file changed since preview"})
	}
	err := Apply(ctx, nil, preview(Row{Line: 2, Action: Add}, Row{Line: 3, Action: Error, Message: "m3"}, Row{Line: 5, Action: Error, Message: "m5"}), "abc", a)
	wantValidation(t, err, httpx.FieldError{Field: "line:3", Message: "m3"}, httpx.FieldError{Field: "line:5", Message: "m5"})
	if called != 0 {
		t.Fatal("applier called on failed validation")
	}

	rows := []Row{{Line: 2, Action: Add, Ref: "r2"}, {Line: 3, Action: Exists}, {Line: 4, Action: Invite, Ref: "r4"}}
	if err := Apply(ctx, nil, preview(rows...), "abc", a); err != nil {
		t.Fatal(err)
	}
	if called != 1 || len(got) != 2 || got[0].Ref != "r2" || got[1].Ref != "r4" {
		t.Fatalf("called=%d got=%v", called, got)
	}
	if err := Apply(ctx, nil, preview(Row{Line: 3, Action: Exists}), "abc", a); err != nil || called != 1 {
		t.Fatalf("applier called with nothing to do: %d %v", called, err)
	}
}

func TestApplyRollsBack(t *testing.T) {
	db := testdb.Open(t)
	f := testdb.Seed(t, db)
	ctx := context.Background()
	boom := errors.New("boom")
	a := func(ctx context.Context, tx *sql.Tx, _ []Row) error {
		if _, err := tx.ExecContext(ctx, `insert into departments (university_id, name, code) values ($1, 'X', 'X')`, f.UniversityA); err != nil {
			return err
		}
		return boom
	}
	err := pg.WithTx(ctx, db, f.UniversityA, func(tx *sql.Tx) error {
		return Apply(ctx, tx, preview(Row{Line: 2, Action: Add}), "abc", a)
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	var n int
	if err := pg.WithTx(ctx, db, f.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select count(*) from departments where code = 'X'`).Scan(&n)
	}); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("%d rows persisted", n)
	}
}

func upload(t *testing.T, file string, sum string, withFile bool) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if sum != "" {
		mw.WriteField("checksum", sum)
	}
	mw.WriteField("other", "ignored")
	if withFile {
		fw, _ := mw.CreateFormFile("file", "a.csv")
		fw.Write([]byte(file))
	}
	mw.Close()
	r := httptest.NewRequest("POST", "/", &buf)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	return r
}

func TestReadUpload(t *testing.T) {
	csv := "email\na@a.edu\n"
	recs, sum, form, err := ReadUpload(upload(t, csv, " deadbeef \n", true), emailSpec)
	h := sha256.Sum256([]byte(csv))
	if err != nil || len(recs) != 1 || sum != hex.EncodeToString(h[:]) || form != "deadbeef" {
		t.Fatalf("recs=%v sum=%s form=%q err=%v", recs, sum, form, err)
	}
	// checksum part after the file part
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "a.csv")
	fw.Write([]byte(csv))
	mw.WriteField("checksum", "late")
	mw.Close()
	r := httptest.NewRequest("POST", "/", &buf)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	if _, _, form, err = ReadUpload(r, emailSpec); err != nil || form != "late" {
		t.Fatalf("form=%q err=%v", form, err)
	}

	_, _, _, err = ReadUpload(upload(t, "", "x", false), emailSpec)
	wantValidation(t, err, httpx.FieldError{Field: "file", Message: "is required"})

	r = httptest.NewRequest("POST", "/", strings.NewReader("email\n"))
	r.Header.Set("Content-Type", "text/csv")
	_, _, _, err = ReadUpload(r, emailSpec)
	wantValidation(t, err, httpx.FieldError{Field: "file", Message: "must be a multipart/form-data upload"})

	s := emailSpec
	s.MaxBytes = 10
	_, _, _, err = ReadUpload(upload(t, "email\n"+strings.Repeat("a", 50), "", true), s)
	wantValidation(t, err, httpx.FieldError{Field: "file", Message: "file too large"})

	// body beyond MaxBytes+1MiB, in a non-file part
	var big bytes.Buffer
	mw = multipart.NewWriter(&big)
	mw.WriteField("junk", strings.Repeat("x", 2<<20))
	mw.Close()
	r = httptest.NewRequest("POST", "/", &big)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	_, _, _, err = ReadUpload(r, s)
	wantValidation(t, err, httpx.FieldError{Field: "file", Message: "file too large"})
}

func TestParseRejectsNonUTF8AndNUL(t *testing.T) {
	for _, in := range []string{"email\nJos\xe9@a.edu\n", "email\na\x00b@a.edu\n"} {
		_, _, err := Parse(strings.NewReader(in), emailSpec)
		wantValidation(t, err, httpx.FieldError{Field: "file", Message: "must be UTF-8 encoded text"})
	}
}

func TestParseWrongFieldCount(t *testing.T) {
	for in, line := range map[string]string{"email,displayname\na@a.edu\n": "line 2", "email\na@a.edu\nb@a.edu,x,y\n": "line 3"} {
		_, _, err := Parse(strings.NewReader(in), emailSpec)
		wantValidation(t, err, httpx.FieldError{Field: "file", Message: line + ": wrong number of fields"})
	}
}

func TestPreviewEmptyKeyNotDuplicate(t *testing.T) {
	recs := []Record{{2, map[string]string{"email": ""}}, {3, map[string]string{"email": ""}}}
	p, err := PreviewRows(context.Background(), nil, recs, "x", emailSpec, allAdd)
	if err != nil || p.Counts[Add] != 2 {
		t.Fatalf("p=%+v err=%v", p, err)
	}
}
