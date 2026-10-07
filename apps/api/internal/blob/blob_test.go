package blob

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	uni1 = "11111111-1111-1111-1111-111111111111"
	uni2 = "22222222-2222-2222-2222-222222222222"
	idA  = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	idB  = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
)

func newDir(t *testing.T) Dir {
	t.Helper()
	d, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func entries(t *testing.T, path string) []string {
	t.Helper()
	es, err := os.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range es {
		out = append(out, e.Name())
	}
	return out
}

func TestNew(t *testing.T) {
	root := t.TempDir()
	if _, err := New(root); err != nil {
		t.Fatal(err)
	}
	if n := entries(t, root); len(n) != 0 {
		t.Fatalf("probe left behind: %v", n)
	}
	file := filepath.Join(root, "f")
	os.WriteFile(file, nil, 0o600)
	bad := map[string]string{"missing": filepath.Join(root, "nope"), "file": file}
	if os.Geteuid() != 0 {
		ro := filepath.Join(root, "ro")
		os.Mkdir(ro, 0o500)
		bad["readonly"] = ro
	}
	for name, p := range bad {
		if _, err := New(p); err == nil || !strings.Contains(err.Error(), "UNICOURSE_FILES_DIR") {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

func TestPutCommitOpen(t *testing.T) {
	d := newDir(t)
	tmp, err := d.Put(uni1, strings.NewReader("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if tmp.Size != 5 || tmp.SHA256 != "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Fatalf("%+v", tmp)
	}
	final := filepath.Join(d.Root, uni1, idA)
	if _, err := os.Stat(final); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("final exists before commit")
	}
	if n := entries(t, filepath.Join(d.Root, uni1, ".tmp")); len(n) != 1 {
		t.Fatalf(".tmp = %v", n)
	}
	if err := d.Commit(tmp, uni1, idA); err != nil {
		t.Fatal(err)
	}
	if n := entries(t, filepath.Join(d.Root, uni1, ".tmp")); len(n) != 0 {
		t.Fatalf(".tmp = %v", n)
	}
	f, err := d.Open(uni1, idA)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if b, _ := io.ReadAll(f); string(b) != "hello" {
		t.Fatalf("%q", b)
	}
}

var errBoom = errors.New("boom")

type errReader struct{ n int }

func (e *errReader) Read(p []byte) (int, error) {
	if e.n == 0 {
		e.n = 1
		return copy(p, "ab"), nil
	}
	return 0, errBoom
}

func TestPutReadErrorCleansUp(t *testing.T) {
	d := newDir(t)
	_, err := d.Put(uni1, &errReader{})
	if err == nil || !errors.Is(err, errBoom) {
		t.Fatalf("err = %v", err)
	}
	if n := entries(t, filepath.Join(d.Root, uni1, ".tmp")); len(n) != 0 {
		t.Fatalf(".tmp = %v", n)
	}
}

func TestTempRemove(t *testing.T) {
	d := newDir(t)
	tmp, _ := d.Put(uni1, strings.NewReader("x"))
	tmp.Remove()
	tmp.Remove()
	if n := entries(t, filepath.Join(d.Root, uni1, ".tmp")); len(n) != 0 {
		t.Fatalf(".tmp = %v", n)
	}
}

func commit(t *testing.T, d Dir, uni, id string) {
	t.Helper()
	tmp, err := d.Put(uni, strings.NewReader("x"))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Commit(tmp, uni, id); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveAndDiscard(t *testing.T) {
	d := newDir(t)
	commit(t, d, uni1, idA)
	commit(t, d, uni1, idB)
	if err := d.Remove(uni1, idA); err != nil {
		t.Fatal(err)
	}
	if err := d.Remove(uni1, idA); err != nil {
		t.Fatalf("missing: %v", err)
	}
	commit(t, d, uni1, idA)
	d.Discard(uni1, idA, idB)
	if n := entries(t, filepath.Join(d.Root, uni1)); len(n) != 1 || n[0] != ".tmp" {
		t.Fatalf("%v", n)
	}
}

func TestOpenMissing(t *testing.T) {
	if _, err := newDir(t).Open(uni1, idA); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("err = %v", err)
	}
}

func TestBadIDs(t *testing.T) {
	d := newDir(t)
	if _, err := d.Put("../x", strings.NewReader("x")); !errors.Is(err, ErrBadID) {
		t.Errorf("put: %v", err)
	}
	tmp, _ := d.Put(uni1, strings.NewReader("x"))
	defer tmp.Remove()
	if err := d.Commit(tmp, uni1, "../../etc/passwd"); !errors.Is(err, ErrBadID) {
		t.Errorf("commit: %v", err)
	}
	if _, err := d.Open(uni1, "x"); !errors.Is(err, ErrBadID) {
		t.Errorf("open: %v", err)
	}
	if err := d.Remove("", idA); !errors.Is(err, ErrBadID) {
		t.Errorf("remove: %v", err)
	}
	if n := entries(t, filepath.Dir(d.Root)); len(n) != 1 {
		t.Errorf("created outside root: %v", n)
	}
}

func TestWalk(t *testing.T) {
	d := newDir(t)
	commit(t, d, uni1, idA)
	commit(t, d, uni2, idB)
	os.WriteFile(filepath.Join(d.Root, uni1, "notes.txt"), nil, 0o600)
	os.MkdirAll(filepath.Join(d.Root, "not-a-uuid"), 0o750)
	os.WriteFile(filepath.Join(d.Root, "not-a-uuid", idA), nil, 0o600)
	os.WriteFile(filepath.Join(d.Root, "stray"), nil, 0o600)
	os.WriteFile(filepath.Join(d.Root, uni1, ".tmp", "up-1"), nil, 0o600)
	want := map[string]time.Time{}
	for _, k := range [][2]string{{uni1, idA}, {uni2, idB}} {
		st, err := os.Stat(filepath.Join(d.Root, k[0], k[1]))
		if err != nil {
			t.Fatal(err)
		}
		want[k[0]+"/"+k[1]] = st.ModTime()
	}
	got := map[string]time.Time{}
	if err := d.Walk(func(u, id string, mt time.Time) error { got[u+"/"+id] = mt; return nil }); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %v", got)
	}
	for k, mt := range want {
		if !got[k].Equal(mt) {
			t.Errorf("%s: %v != %v", k, got[k], mt)
		}
	}
	stop := errors.New("stop")
	if err := d.Walk(func(string, string, time.Time) error { return stop }); !errors.Is(err, stop) {
		t.Errorf("walk err = %v", err)
	}
}

func TestZeroDir(t *testing.T) {
	var d Dir
	if _, err := d.Put(uni1, strings.NewReader("x")); err == nil {
		t.Error("put")
	}
	if err := d.Commit(Temp{}, uni1, idA); err == nil {
		t.Error("commit")
	}
	if _, err := d.Open(uni1, idA); err == nil {
		t.Error("open")
	}
	if err := d.Remove(uni1, idA); err == nil {
		t.Error("remove")
	}
	if err := d.Walk(func(string, string, time.Time) error { return nil }); err == nil {
		t.Error("walk")
	}
	if _, err := os.Stat(uni1); err == nil {
		t.Error("created in working directory")
	}
}

func TestCommitSyncFailureLeavesNothing(t *testing.T) {
	d := newDir(t)
	tmp, err := d.Put(uni1, strings.NewReader("hello"))
	if err != nil {
		t.Fatal(err)
	}
	old := syncDir
	syncDir = func(string) error { return errBoom }
	t.Cleanup(func() { syncDir = old })
	if err := d.Commit(tmp, uni1, idA); !errors.Is(err, errBoom) {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(d.Root, uni1, idA)); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("target left behind")
	}
	if n := entries(t, filepath.Join(d.Root, uni1, ".tmp")); len(n) != 0 {
		t.Fatalf(".tmp = %v", n)
	}
}
