// Package csvimport is the shared engine behind the CSV imports: parse an upload, classify each
// row (preview, writes nothing), and apply the previewed rows when the file is unchanged.
package csvimport

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"sort"
	"strings"

	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
)

type Action string

const (
	Add    Action = "add"
	Exists Action = "exists"
	Invite Action = "invite"
	Error  Action = "error"
)

// Record is one data row; Line is the 1-based line it starts on (correct with quoted newlines).
type Record struct {
	Line int
	Data map[string]string
}

type Row struct {
	Line    int               `json:"line"`
	Action  Action            `json:"action"`
	Message string            `json:"message,omitempty"`
	Data    map[string]string `json:"data"`
	Ref     any               `json:"-"`
}

type Preview struct {
	Checksum string         `json:"checksum"`
	Rows     []Row          `json:"rows"`
	Counts   map[Action]int `json:"counts"` // always has all four actions
}

// Verdict is the classifier's decision for one record; Ref is handed back to the Applier.
type Verdict struct {
	Action  Action
	Message string
	Ref     any
}

type Spec struct {
	Required, Optional []string                       // lowercased header names
	MaxBytes           int64                          // 0 means 5 MiB
	Key                func(map[string]string) string // duplicate key; nil joins the Required values with "\x00"
}

// Classifier sees every non-duplicate record once, in line order, and returns one Verdict per
// record. It must not write.
type Classifier func(ctx context.Context, tx *sql.Tx, recs []Record) ([]Verdict, error)

// Applier receives all Add and Invite rows in line order, once; it is not called when there are none.
type Applier func(ctx context.Context, tx *sql.Tx, rows []Row) error

func (s Spec) max() int64 {
	if s.MaxBytes == 0 {
		return 5 << 20
	}
	return s.MaxBytes
}

func fileErr(msg string) error {
	return httpx.Validation(httpx.FieldError{Field: "file", Message: msg})
}

// Parse reads and validates a CSV; the checksum is the lowercase hex SHA-256 of the raw bytes (BOM included).
func Parse(r io.Reader, s Spec) ([]Record, string, error) {
	raw, err := io.ReadAll(io.LimitReader(r, s.max()+1))
	if err != nil {
		var mb *http.MaxBytesError
		if errors.As(err, &mb) {
			return nil, "", fileErr("file too large")
		}
		return nil, "", fmt.Errorf("read upload: %w", err)
	}
	if int64(len(raw)) > s.max() {
		return nil, "", fileErr("file too large")
	}
	sum := sha256.Sum256(raw)
	checksum := hex.EncodeToString(sum[:])

	cr := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf"))))
	header, err := cr.Read()
	if err == io.EOF {
		return nil, "", fileErr("header row required")
	}
	if err != nil {
		return nil, "", csvErr(err)
	}
	col := map[string]int{}
	for i, h := range header {
		h = strings.ToLower(strings.TrimSpace(h))
		if _, dup := col[h]; !dup {
			col[h] = i
		}
	}
	var missing []httpx.FieldError
	for _, name := range s.Required {
		if _, ok := col[name]; !ok {
			missing = append(missing, httpx.FieldError{Field: "file", Message: "missing column " + name})
		}
	}
	if len(missing) > 0 {
		return nil, "", httpx.Validation(missing...)
	}
	names := append(append([]string{}, s.Required...), s.Optional...)

	recs := []Record{}
	for {
		cells, err := cr.Read()
		if err == io.EOF {
			return recs, checksum, nil
		}
		if err != nil {
			return nil, "", csvErr(err)
		}
		empty := true
		for _, c := range cells {
			if strings.TrimSpace(c) != "" {
				empty = false
				break
			}
		}
		if empty {
			continue
		}
		line, _ := cr.FieldPos(0)
		data := make(map[string]string, len(names))
		for _, n := range names {
			data[n] = ""
			if i, ok := col[n]; ok {
				data[n] = strings.TrimSpace(cells[i])
			}
		}
		recs = append(recs, Record{Line: line, Data: data})
	}
}

func csvErr(err error) error {
	var pe *csv.ParseError
	if errors.As(err, &pe) {
		return fileErr(fmt.Sprintf("line %d: %v", pe.Line, pe.Err))
	}
	return fileErr(err.Error())
}

// ReadUpload streams a multipart upload: part "file" is parsed, part "checksum" (the checksum
// the client saw in the preview) is returned as formChecksum; other parts are ignored.
func ReadUpload(r *http.Request, s Spec) (recs []Record, checksum, formChecksum string, err error) {
	if mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); mt != "multipart/form-data" {
		return nil, "", "", fileErr("must be a multipart/form-data upload")
	}
	r.Body = http.MaxBytesReader(nil, r.Body, s.max()+1<<20)
	mr, err := r.MultipartReader()
	if err != nil {
		return nil, "", "", fileErr("must be a multipart/form-data upload")
	}
	found := false
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", "", uploadErr(err)
		}
		switch part.FormName() {
		case "file":
			if found {
				continue
			}
			found = true
			if recs, checksum, err = Parse(part, s); err != nil {
				return nil, "", "", uploadErr(err)
			}
		case "checksum":
			b, err := io.ReadAll(io.LimitReader(part, 128))
			if err != nil {
				return nil, "", "", uploadErr(err)
			}
			formChecksum = strings.TrimSpace(string(b))
		}
	}
	if !found {
		return nil, "", "", fileErr("is required")
	}
	return recs, checksum, formChecksum, nil
}

func uploadErr(err error) error {
	var mb *http.MaxBytesError
	if errors.As(err, &mb) {
		return fileErr("file too large")
	}
	var he *httpx.Error
	if errors.As(err, &he) {
		return err
	}
	return fileErr("invalid multipart upload: " + err.Error())
}

// PreviewRows classifies recs (duplicates by Spec.Key become Error rows and skip the classifier).
func PreviewRows(ctx context.Context, tx *sql.Tx, recs []Record, checksum string, s Spec, c Classifier) (Preview, error) {
	key := s.Key
	if key == nil {
		key = func(d map[string]string) string {
			vs := make([]string, len(s.Required))
			for i, n := range s.Required {
				vs[i] = d[n]
			}
			return strings.Join(vs, "\x00")
		}
	}
	first := map[string]int{}
	dupOf := make([]int, len(recs)) // 0 = not a duplicate
	var fresh []Record
	for i, rec := range recs {
		k := key(rec.Data)
		if l, ok := first[k]; ok {
			dupOf[i] = l
			continue
		}
		first[k] = rec.Line
		fresh = append(fresh, rec)
	}
	var vs []Verdict
	if len(fresh) > 0 {
		var err error
		if vs, err = c(ctx, tx, fresh); err != nil {
			return Preview{}, err
		}
		if len(vs) != len(fresh) {
			return Preview{}, fmt.Errorf("csvimport: classifier returned %d verdicts for %d records", len(vs), len(fresh))
		}
	}
	p := Preview{Checksum: checksum, Rows: make([]Row, 0, len(recs)),
		Counts: map[Action]int{Add: 0, Exists: 0, Invite: 0, Error: 0}}
	n := 0
	for i, rec := range recs {
		row := Row{Line: rec.Line, Data: rec.Data}
		if dupOf[i] != 0 {
			row.Action, row.Message = Error, fmt.Sprintf("duplicate of line %d", dupOf[i])
		} else {
			v := vs[n]
			n++
			if _, ok := p.Counts[v.Action]; !ok {
				return Preview{}, fmt.Errorf("csvimport: classifier returned unknown action %q", v.Action)
			}
			row.Action, row.Message, row.Ref = v.Action, v.Message, v.Ref
		}
		p.Counts[row.Action]++
		p.Rows = append(p.Rows, row)
	}
	return p, nil
}

// Apply hands the Add and Invite rows to a once the form checksum matches the preview and no row is an Error.
func Apply(ctx context.Context, tx *sql.Tx, p Preview, formChecksum string, a Applier) error {
	if formChecksum == "" || formChecksum != p.Checksum {
		return httpx.Validation(httpx.FieldError{Field: "checksum", Message: "file changed since preview"})
	}
	var bad []httpx.FieldError
	var todo []Row
	rows := append([]Row{}, p.Rows...)
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Line < rows[j].Line })
	for _, r := range rows {
		switch r.Action {
		case Error:
			bad = append(bad, httpx.FieldError{Field: fmt.Sprintf("line:%d", r.Line), Message: r.Message})
		case Add, Invite:
			todo = append(todo, r)
		}
	}
	if len(bad) > 0 {
		return httpx.Validation(bad...)
	}
	if len(todo) == 0 {
		return nil
	}
	return a(ctx, tx, todo)
}
