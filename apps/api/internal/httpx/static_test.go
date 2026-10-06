package httpx

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func staticFixture(t *testing.T) http.Handler {
	t.Helper()
	base := t.TempDir()
	dir := filepath.Join(base, "dist")
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.MkdirAll(filepath.Join(dir, "assets"), 0o755))
	must(os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>app</html>"), 0o644))
	must(os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("console.log(1)"), 0o644))
	must(os.WriteFile(filepath.Join(dir, "favicon.ico"), []byte("ico"), 0o644))
	must(os.WriteFile(filepath.Join(dir, ".env"), []byte("DOTSECRET"), 0o644))
	must(os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	must(os.WriteFile(filepath.Join(dir, ".git", "config"), []byte("DOTSECRET"), 0o644))
	must(os.WriteFile(filepath.Join(dir, "assets", ".hidden"), []byte("DOTSECRET"), 0o644))
	must(os.WriteFile(filepath.Join(base, "secret"), []byte("TOPSECRET"), 0o644))
	must(os.Symlink(filepath.Join(base, "secret"), filepath.Join(dir, "link.txt")))
	h, err := Static(dir)
	must(err)
	return h
}

func get(h http.Handler, method, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

func isEnvelope404(rec *httptest.ResponseRecorder) bool {
	return rec.Code == 404 && strings.Contains(rec.Body.String(), `"code":"not_found"`) && strings.Contains(rec.Header().Get("Content-Type"), "json")
}

func TestStaticAsset(t *testing.T) {
	rec := get(staticFixture(t), "GET", "/assets/app.js")
	if rec.Code != 200 || rec.Body.String() != "console.log(1)" {
		t.Fatal(rec.Code, rec.Body)
	}
	if rec.Header().Get("Cache-Control") != "public, max-age=31536000, immutable" || !strings.Contains(rec.Header().Get("Content-Type"), "javascript") || rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal(rec.Header())
	}
}

func TestStaticFallback(t *testing.T) {
	h := staticFixture(t)
	for _, p := range []string{"/", "/a/courses", "/sign-in?next=/a/me", "/invite/tok123", "/index.html", "/assets/"} {
		rec := get(h, "GET", p)
		if rec.Code != 200 || rec.Body.String() != "<html>app</html>" || rec.Header().Get("Cache-Control") != "no-cache" ||
			rec.Header().Get("X-Frame-Options") != "DENY" || rec.Header().Get("Referrer-Policy") != "same-origin" {
			t.Errorf("%s: %d %q %v", p, rec.Code, rec.Body, rec.Header())
		}
	}
	if rec := get(h, "HEAD", "/"); rec.Code != 200 || rec.Body.Len() != 0 {
		t.Fatal(rec.Code, rec.Body)
	}
	if rec := get(h, "GET", "/favicon.ico"); rec.Code != 200 || rec.Body.String() != "ico" {
		t.Fatal(rec.Code)
	}
}

func TestStaticNotFound(t *testing.T) {
	h := staticFixture(t)
	for _, c := range [][2]string{{"GET", "/assets/missing.js"}, {"GET", "/missing.ico"}, {"GET", "/api/v1/nothing"}, {"GET", "/api/other"}, {"GET", "/api"}, {"POST", "/a/courses"}, {"DELETE", "/"}} {
		if rec := get(h, c[0], c[1]); !isEnvelope404(rec) {
			t.Errorf("%v: %d %s %s", c, rec.Code, rec.Header().Get("Content-Type"), rec.Body)
		}
	}
}

func TestStaticNoEscape(t *testing.T) {
	h := staticFixture(t)
	for _, p := range []string{"/../secret", "/assets/../../secret", "/%2e%2e/secret", "/assets/%2e%2e%2f%2e%2e%2fsecret", "/link.txt", "//etc/passwd"} {
		rec := get(h, "GET", p)
		if strings.Contains(rec.Body.String(), "TOPSECRET") || strings.Contains(rec.Body.String(), "root:") {
			t.Errorf("%s leaked", p)
		}
	}
}

func TestStaticRequiresIndex(t *testing.T) {
	if _, err := Static(t.TempDir()); err == nil {
		t.Fatal("want error")
	}
}

func TestStaticDotfiles(t *testing.T) {
	h := staticFixture(t)
	for _, p := range []string{"/.env", "/.git/config", "/assets/.hidden", "/.git", "/assets/.hidden/x"} {
		rec := get(h, "GET", p)
		if !isEnvelope404(rec) || strings.Contains(rec.Body.String(), "DOTSECRET") {
			t.Errorf("%s: %d %q", p, rec.Code, rec.Body)
		}
	}
}

func TestStaticNotFoundNosniff(t *testing.T) {
	h := staticFixture(t)
	for _, p := range []string{"/missing.js", "/api/v1/x", "/.env"} {
		if rec := get(h, "GET", p); rec.Code != 404 || rec.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("%s: %d %v", p, rec.Code, rec.Header())
		}
	}
}

func TestStaticCSP(t *testing.T) {
	const want = "default-src 'self'; object-src 'none'; base-uri 'self'; frame-ancestors 'none'"
	h := staticFixture(t)
	for _, p := range []string{"/", "/a/b", "/index.html"} {
		if got := get(h, "GET", p).Header().Get("Content-Security-Policy"); got != want {
			t.Errorf("%s: CSP %q", p, got)
		}
	}
}

func TestStaticLogHidesFallbackPath(t *testing.T) {
	var buf bytes.Buffer
	h := LogMiddleware(slog.New(slog.NewJSONHandler(&buf, nil)), staticFixture(t))
	for _, p := range []string{"/invite/SECRETTOKEN?email=a@b.c", "/index.html"} {
		if rec := get(h, "GET", p); rec.Code != 200 {
			t.Fatal(p, rec.Code)
		}
	}
	if strings.Contains(buf.String(), "SECRETTOKEN") || !strings.Contains(buf.String(), `"path":"/*"`) {
		t.Fatal(buf.String())
	}
}
