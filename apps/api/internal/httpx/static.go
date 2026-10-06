package httpx

import (
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
)

// Static serves the built single-page app in dir: existing files as they are, any other extensionless
// path as index.html (client-side routes). /api/ paths and missing files with an extension answer the
// JSON 404 envelope. The root is opened with os.OpenRoot, so neither ".." nor symlinks escape dir.
func Static(dir string) (http.Handler, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("web dir: %w", err)
	}
	fsys := root.FS()
	if fi, err := fs.Stat(fsys, "index.html"); err != nil || !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("web dir %s: no index.html", dir)
	}
	serve := func(w http.ResponseWriter, r *http.Request, name string) {
		f, err := fsys.Open(name)
		if err != nil {
			WriteError(w, r, NotFound("route"))
			return
		}
		defer f.Close()
		fi, err := f.Stat()
		rs, ok := f.(io.ReadSeeker)
		if err != nil || !ok {
			WriteError(w, r, NotFound("route"))
			return
		}
		h := w.Header()
		if name == "index.html" {
			setRoute(r.Context(), "/*") // the log line must not carry client-side route params such as invite tokens
		}
		switch {
		case name == "index.html":
			h.Set("Cache-Control", "no-cache")
		case strings.HasPrefix(name, "assets/"):
			h.Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		if path.Ext(name) == ".html" {
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "same-origin")
			h.Set("Content-Security-Policy", "default-src 'self'; object-src 'none'; base-uri 'self'; frame-ancestors 'none'")
		}
		http.ServeContent(w, r, name, fi.ModTime(), rs)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		p := path.Clean("/" + r.URL.Path)
		if (r.Method != http.MethodGet && r.Method != http.MethodHead) || hasDotSegment(p) || p == "/api" || strings.HasPrefix(p, "/api/") {
			WriteError(w, r, NotFound("route"))
			return
		}
		name := strings.TrimPrefix(p, "/")
		if name != "" {
			if fi, err := fs.Stat(fsys, name); err == nil && fi.Mode().IsRegular() {
				serve(w, r, name)
				return
			}
		}
		if path.Ext(p) != "" {
			WriteError(w, r, NotFound("route"))
			return
		}
		serve(w, r, "index.html")
	}), nil
}

func hasDotSegment(p string) bool {
	for _, s := range strings.Split(p, "/") {
		if strings.HasPrefix(s, ".") {
			return true
		}
	}
	return false
}
