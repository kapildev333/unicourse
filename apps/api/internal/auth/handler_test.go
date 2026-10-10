package auth

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/outbox"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/testdb"
)

const webBase = "http://web.test"

// hx is a handler-test harness: the real httpx.Router with auth.Register, the middleware chain of
// cmd/api, real deps and a test database. Copy it (and signedIn) for other packages' handler tests.
type hx struct {
	t    *testing.T
	db   *sql.DB
	fx   testdb.Fixture
	d    Deps
	h    http.Handler
	logs *bytes.Buffer // request log
	iss  *fakeIssuer
}

func newHX(t *testing.T) *hx { return newHXLimit(t, 100, 300) }

func newHXLimit(t *testing.T, perEmail, perIP int) *hx {
	t.Helper()
	db := testdb.Open(t)
	x := &hx{t: t, db: db, fx: testdb.Seed(t, db), logs: &bytes.Buffer{}, iss: newFakeIssuer(t)}
	p, err := newProvider(context.Background(), "google", x.iss.URL, "client", "secret", "http://api.test/api/v1/auth/oidc/google/callback")
	if err != nil {
		t.Fatal(err)
	}
	x.d = Deps{
		DB: db, Sessions: &SessionStore{DB: db}, Providers: map[string]Provider{"google": p}, Secret: testSecret, WebBase: webBase,
		Magic:      &MagicLinks{DB: db, BaseURL: "http://api.test"},
		EmailLimit: NewLimiter(perEmail, time.Hour), IPLimit: NewLimiter(perIP, time.Hour),
	}
	rt := httpx.NewRouter()
	Register(rt, x.d)
	// httptest requests come from 192.0.2.1: the harness models "behind a trusted proxy".
	x.h = httpx.ClientIPMiddleware([]netip.Prefix{netip.MustParsePrefix("192.0.2.0/24")},
		httpx.RequestIDMiddleware(httpx.LogMiddleware(slog.New(slog.NewJSONHandler(x.logs, nil)), httpx.Recover(rt))))
	return x
}

func (x *hx) do(method, path, body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	return x.doH(method, path, body, nil, cookies...)
}

// from sends a request with the given socket peer and, when non-empty, X-Forwarded-For.
func (x *hx) from(remote, xff, method, path, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.RemoteAddr = remote
	if xff != "" {
		r.Header.Set("X-Forwarded-For", xff)
	}
	w := httptest.NewRecorder()
	x.h.ServeHTTP(w, r)
	return w
}

func (x *hx) doH(method, path, body string, hdr map[string]string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	for k, v := range hdr {
		r.Header.Set(k, v)
	}
	for _, c := range cookies {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	x.h.ServeHTTP(w, r)
	return w
}

// signedIn returns a session cookie for an existing user (no HTTP flow, no Secure-cookie jar issues).
func (x *hx) signedIn(email string) *http.Cookie {
	x.t.Helper()
	uid, err := SignIn(context.Background(), x.db, email, "")
	if err != nil {
		x.t.Fatal(err)
	}
	tok, err := x.d.Sessions.Create(context.Background(), uid)
	if err != nil {
		x.t.Fatal(err)
	}
	return &http.Cookie{Name: SessionCookie, Value: tok}
}

func cookieNamed(w *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range w.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// link requests a magic link for email and returns the path (plus query) of the mailed URL.
func (x *hx) link(email, next string) string {
	x.t.Helper()
	sqlExec(x.t, x.db, `delete from outbox_mails`)
	b, _ := json.Marshal(map[string]string{"email": email, "next": next})
	if w := x.do("POST", "/api/v1/auth/magic-link", string(b)); w.Code != 202 || w.Body.Len() != 0 {
		x.t.Fatalf("request link: %d %s", w.Code, w.Body)
	}
	u, err := url.Parse(lastURL(x.t, x.db))
	if err != nil {
		x.t.Fatal(err)
	}
	return u.RequestURI()
}

func wantRedirect(t *testing.T, w *httptest.ResponseRecorder, loc string) {
	t.Helper()
	if w.Code != 302 || w.Header().Get("Location") != webBase+loc {
		t.Fatalf("got %d Location %q, want 302 %q", w.Code, w.Header().Get("Location"), webBase+loc)
	}
}

func TestSafeNext(t *testing.T) {
	for in, want := range map[string]string{"/a/courses": "/a/courses", "": "", "//evil.com": "", `/\evil.com`: "", "https://evil.com": "",
		"/a\r\nX: y": "", "/a\x00b": "", "/a\x1fb": "", "/a\x7fb": "", "/a\tb": "", "a/b": "", "/" + strings.Repeat("x", 512): ""} {
		if got := safeNext(in); got != want {
			t.Errorf("safeNext(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMagicLinkUnknownEmailGets202AndNoMail(t *testing.T) {
	x := newHX(t)
	w := x.do("POST", "/api/v1/auth/magic-link", `{"email":"stranger@nowhere.com"}`)
	if w.Code != 202 || w.Body.Len() != 0 || sqlCount(t, x.db, `select count(*) from outbox_mails`) != 0 {
		t.Fatalf("%d %q", w.Code, w.Body)
	}
	if w := x.do("POST", "/api/v1/auth/magic-link", `{"email":"nope"}`); w.Code != 400 {
		t.Fatalf("bad email: %d", w.Code)
	}
}

func TestMagicLinkFlow(t *testing.T) {
	x := newHX(t)
	path := x.link("Admin@A.edu", "")

	for i := 0; i < 2; i++ { // GET never consumes
		w := x.do("GET", path, "")
		b := w.Body.String()
		if w.Code != 200 || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/html") || !strings.Contains(b, `<form method="post"`) ||
			!strings.Contains(b, "Continue signing in") || strings.Contains(b, "<script") ||
			w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Referrer-Policy") != "no-referrer" {
			t.Fatalf("GET %d: %d %v %s", i, w.Code, w.Header(), b)
		}
	}
	w := x.do("POST", path, "")
	wantRedirect(t, w, "/a/me")
	c := cookieNamed(w, SessionCookie)
	if c == nil || !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode || c.Path != "/" || c.Value == "" {
		t.Fatalf("session cookie %+v", c)
	}
	if m := x.do("GET", "/api/v1/me", "", c); m.Code != 200 {
		t.Fatalf("me with new session: %d", m.Code)
	}
	wantRedirect(t, x.do("POST", path, ""), "/sign-in?error=validation_failed") // used

	path = x.link("admin@a.edu", "")
	sqlExec(t, x.db, `update magic_links set expires_at = now() - interval '1 second'`)
	wantRedirect(t, x.do("POST", path, ""), "/sign-in?error=validation_failed") // expired
	wantRedirect(t, x.do("POST", "/api/v1/auth/magic-link/garbage", ""), "/sign-in?error=validation_failed")
	if w := x.do("GET", "/api/v1/auth/magic-link/garbage<x>", ""); w.Code != 200 || strings.Contains(w.Body.String(), "<x>") {
		t.Fatalf("garbage token page: %d %s", w.Code, w.Body)
	}
}

func TestMagicLinkNoMembership(t *testing.T) {
	x := newHX(t)
	// new@a.edu passes the domain rule but has no membership: signed in, sent to the sign-in page.
	w := x.do("POST", x.link("new@a.edu", ""), "")
	wantRedirect(t, w, "/sign-in?error=no_membership")
	if cookieNamed(w, SessionCookie) == nil {
		t.Fatal("session should still be set (design 6: landing follows session creation)")
	}
}

func TestMagicLinkNext(t *testing.T) {
	x := newHX(t)
	wantRedirect(t, x.do("POST", x.link("admin@a.edu", "/a/courses"), ""), "/a/courses")
	for _, bad := range []string{"//evil.com", "https://evil.com"} {
		wantRedirect(t, x.do("POST", x.link("admin@a.edu", bad), ""), "/a/me")
		// a hand-edited link is ignored the same way
		p := strings.SplitN(x.link("admin@a.edu", ""), "?", 2)[0]
		wantRedirect(t, x.do("POST", p+"?next="+url.QueryEscape(bad), ""), "/a/me")
	}
	// the confirm page carries a valid next into the form action, and drops an invalid one
	p := x.link("admin@a.edu", "/a/courses")
	if b := x.do("GET", p, "").Body.String(); !strings.Contains(b, "next=%2Fa%2Fcourses") {
		t.Fatalf("form action lost next: %s", b)
	}
}

func TestMagicLinkRateLimit(t *testing.T) {
	x := newHXLimit(t, 5, 300)
	for i := 0; i < 5; i++ {
		if w := x.do("POST", "/api/v1/auth/magic-link", `{"email":"admin@a.edu"}`); w.Code != 202 {
			t.Fatalf("request %d: %d", i, w.Code)
		}
	}
	w := x.do("POST", "/api/v1/auth/magic-link", `{"email":"ADMIN@a.edu"}`)
	if w.Code != 429 || !strings.Contains(w.Body.String(), `"code":"rate_limited"`) {
		t.Fatalf("6th: %d %s", w.Code, w.Body)
	}
	if w := x.do("POST", "/api/v1/auth/magic-link", `{"email":"other@a.edu"}`); w.Code != 202 {
		t.Fatalf("other email: %d", w.Code)
	}
}

func TestIPLimitPerClientBehindProxy(t *testing.T) {
	x := newHXLimit(t, 100, 1)
	n := 0
	ml := func(remote, xff string) *httptest.ResponseRecorder {
		n++
		return x.from(remote, xff, "POST", "/api/v1/auth/magic-link", fmt.Sprintf(`{"email":"nobody%d@nowhere.test"}`, n))
	}
	if w := ml("192.0.2.1:1", "203.0.113.1"); w.Code != 202 {
		t.Fatalf("first: %d", w.Code)
	}
	if w := ml("192.0.2.1:1", "203.0.113.1"); w.Code != 429 || !strings.Contains(w.Body.String(), `"code":"rate_limited"`) {
		t.Fatalf("again: %d %s", w.Code, w.Body)
	}
	if w := ml("192.0.2.1:1", "203.0.113.2"); w.Code != 202 {
		t.Fatalf("second client: %d", w.Code)
	}
	if w := ml("198.51.100.1:1", "203.0.113.3"); w.Code != 202 {
		t.Fatalf("untrusted first: %d", w.Code)
	}
	if w := ml("198.51.100.1:1", "203.0.113.4"); w.Code != 429 {
		t.Fatalf("untrusted second: %d", w.Code)
	}
}

func TestOIDCStartRateLimit(t *testing.T) {
	x := newHXLimit(t, 100, 1)
	const path = "/api/v1/auth/oidc/google/start"
	if w := x.do("POST", path, `{"slug":"a"}`); w.Code != 200 {
		t.Fatalf("first: %d %s", w.Code, w.Body)
	}
	w := x.do("POST", path, `{"slug":"a"}`)
	if w.Code != 429 || !strings.Contains(w.Body.String(), `"code":"rate_limited"`) || cookieNamed(w, oidcCookie) != nil {
		t.Fatalf("second: %d %s", w.Code, w.Body)
	}
	if w := x.do("POST", path, `{}`); w.Code != 400 {
		t.Fatalf("validation first: %d", w.Code)
	}
	if w := x.do("POST", "/api/v1/auth/magic-link", `{"email":"nobody@nowhere.test"}`); w.Code != 202 {
		t.Fatalf("magic link bucket: %d", w.Code)
	}
}

func TestMagicLinkTokenNotInRequestLog(t *testing.T) {
	x := newHX(t)
	path := x.link("admin@a.edu", "")
	tok := tokenFromURL(t, "http://api.test"+path)
	x.logs.Reset()
	x.do("GET", path, "")
	x.do("POST", path, "")
	if strings.Contains(x.logs.String(), tok) || !strings.Contains(x.logs.String(), `"path":"/api/v1/auth/magic-link/{token}"`) {
		t.Fatalf("log: %s", x.logs)
	}
}

func TestMe(t *testing.T) {
	x := newHX(t)
	if w := x.do("GET", "/api/v1/me", ""); w.Code != 401 || !strings.Contains(w.Body.String(), `"code":"unauthenticated"`) {
		t.Fatalf("no cookie: %d %s", w.Code, w.Body)
	}
	w := x.do("GET", "/api/v1/me", "", x.signedIn("admin@a.edu"))
	var got struct {
		User        struct{ ID, Email, DisplayName string }
		Memberships []struct {
			University    struct{ ID, Slug, Name, Timezone, Locale string }
			Role          string
			DepartmentIDs []string
		}
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || w.Code != 200 {
		t.Fatalf("%d %s %v", w.Code, w.Body, err)
	}
	m := got.Memberships
	if got.User.Email != "admin@a.edu" || len(m) != 1 || m[0].University.Slug != "a" || m[0].University.ID != x.fx.UniversityA || m[0].Role != "admin" {
		t.Fatalf("%s", w.Body)
	}
	if m[0].University.Timezone != "America/New_York" || m[0].University.Locale != "en-US" {
		t.Fatalf("campus fields: %s", w.Body)
	}
	if !strings.Contains(w.Body.String(), `"departmentIds":[]`) || !strings.Contains(w.Body.String(), `"displayName"`) {
		t.Fatalf("shape: %s", w.Body)
	}
	if w := x.do("GET", "/api/v1/me", "", x.signedIn("new@a.edu")); !strings.Contains(w.Body.String(), `"memberships":[]`) {
		t.Fatalf("no memberships must be []: %s", w.Body)
	}
}

func TestSignOut(t *testing.T) {
	x := newHX(t)
	c := x.signedIn("admin@a.edu")
	w := x.do("POST", "/api/v1/auth/sign-out", "", c)
	if cc := cookieNamed(w, SessionCookie); w.Code != 204 || cc == nil || cc.MaxAge >= 0 {
		t.Fatalf("sign-out: %d %v", w.Code, w.Result().Cookies())
	}
	if w := x.do("GET", "/api/v1/me", "", c); w.Code != 401 {
		t.Fatalf("me after sign-out: %d", w.Code)
	}
	if w := x.do("POST", "/api/v1/auth/sign-out", "", c); w.Code != 204 {
		t.Fatalf("second sign-out: %d", w.Code)
	}
	if w := x.do("POST", "/api/v1/auth/sign-out", ""); w.Code != 204 {
		t.Fatalf("no cookie sign-out: %d", w.Code)
	}
}

// oidcStart runs the start endpoint and returns the state cookie and the nonce the IdP would see.
func (x *hx) oidcStart(body string) (*http.Cookie, string) {
	x.t.Helper()
	w := x.do("POST", "/api/v1/auth/oidc/google/start", body)
	var out struct{ URL string }
	c := cookieNamed(w, oidcCookie)
	if err := json.Unmarshal(w.Body.Bytes(), &out); w.Code != 200 || err != nil || c == nil || !c.HttpOnly || !c.Secure {
		x.t.Fatalf("start: %d %s %v", w.Code, w.Body, c)
	}
	u, _ := url.Parse(out.URL)
	return c, u.Query().Get("nonce")
}

func TestOIDCEndToEnd(t *testing.T) {
	x := newHX(t)
	c, nonce := x.oidcStart(`{"email":"admin@a.edu","next":"/a/courses"}`)
	x.iss.mint(t, map[string]any{"nonce": nonce, "email": "admin@a.edu", "email_verified": true, "name": "Admin"})
	w := x.do("GET", "/api/v1/auth/oidc/google/callback?code=x&state="+nonce, "", c)
	wantRedirect(t, w, "/a/courses")
	if s := cookieNamed(w, SessionCookie); s == nil || s.Value == "" {
		t.Fatal("no session cookie")
	}

	c, nonce = x.oidcStart(`{"slug":"a"}`) // no next: lands on the slug home
	x.iss.mint(t, map[string]any{"nonce": nonce, "email": "admin@a.edu", "email_verified": true})
	wantRedirect(t, x.do("GET", "/api/v1/auth/oidc/google/callback?code=x&state="+nonce, "", c), "/a/me")

	c, nonce = x.oidcStart(`{"slug":"a"}`)
	x.iss.mint(t, map[string]any{"nonce": nonce, "email": "stranger@nowhere.com", "email_verified": true})
	wantRedirect(t, x.do("GET", "/api/v1/auth/oidc/google/callback?code=x&state="+nonce, "", c), "/sign-in?error=forbidden")

	c, nonce = x.oidcStart(`{"slug":"a"}`)
	wantRedirect(t, x.do("GET", "/api/v1/auth/oidc/google/callback?code=x&state=tampered", "", c), "/sign-in?error=unauthenticated")
	_ = nonce
	wantRedirect(t, x.do("GET", "/api/v1/auth/oidc/google/callback?code=x&state="+nonce, ""), "/sign-in?error=unauthenticated") // no cookie
}

func TestOIDCStartErrors(t *testing.T) {
	x := newHX(t)
	if w := x.do("POST", "/api/v1/auth/oidc/microsoft/start", `{"slug":"a"}`); w.Code != 404 || !strings.Contains(w.Body.String(), `"code":"not_found"`) {
		t.Fatalf("unconfigured: %d %s", w.Code, w.Body)
	}
	for _, b := range []string{`{}`, `{"slug":"a","email":"x@a.edu"}`, `{"email":"nope"}`} {
		if w := x.do("POST", "/api/v1/auth/oidc/google/start", b); w.Code != 400 {
			t.Errorf("%s: %d", b, w.Code)
		}
	}
}

func TestMagicLinkCrossSitePostDoesNotConsume(t *testing.T) {
	x := newHX(t)
	path := x.link("admin@a.edu", "")
	w := x.doH("POST", path, "", map[string]string{"Sec-Fetch-Site": "cross-site"})
	wantRedirect(t, w, "/sign-in?error=validation_failed")
	if len(w.Result().Cookies()) != 0 {
		t.Fatalf("cookies set: %v", w.Result().Cookies())
	}
	wantRedirect(t, x.doH("POST", path, "", map[string]string{"Sec-Fetch-Site": "same-origin"}), "/a/me") // token survived
}

func TestMagicLinkNoFetchMetadataStillWorks(t *testing.T) {
	x := newHX(t)
	wantRedirect(t, x.do("POST", x.link("admin@a.edu", ""), ""), "/a/me")
}

func TestConfirmPageFramingHeaders(t *testing.T) {
	for name, c := range map[string]struct{ web, want string }{
		"different origin": {webBase + "/ignored/path", "'self' " + webBase},
		"same origin":      {"http://api.test", "'self'"},
	} {
		t.Run(name, func(t *testing.T) {
			x := newHX(t)
			x.d.WebBase = c.web
			rt := httpx.NewRouter()
			Register(rt, x.d)
			x.h = rt
			w := x.do("GET", x.link("admin@a.edu", ""), "")
			want := "default-src 'none'; style-src 'unsafe-inline'; form-action " + c.want + "; frame-ancestors 'none'"
			if w.Header().Get("X-Frame-Options") != "DENY" || w.Header().Get("Content-Security-Policy") != want {
				t.Fatalf("headers: %v want CSP %q", w.Header(), want)
			}
		})
	}
}

func TestConfirmPageDropsInvalidNext(t *testing.T) {
	x := newHX(t)
	p := strings.SplitN(x.link("admin@a.edu", ""), "?", 2)[0]
	if b := x.do("GET", p+"?next="+url.QueryEscape("//evil.com"), "").Body.String(); strings.Contains(b, "next=") || strings.Contains(b, "evil") {
		t.Fatalf("invalid next rendered: %s", b)
	}
}

func TestRequestMagicLinkDropsInvalidNext(t *testing.T) {
	x := newHX(t)
	x.link("admin@a.edu", "//evil.com")
	var body string
	if err := x.db.QueryRow(`select body from outbox_mails order by created_at desc, id desc limit 1`).Scan(&body); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(body, "next=") || strings.Contains(body, "evil") {
		t.Fatalf("mailed link carries next: %s", body)
	}
}

func TestNoMembershipIgnoresNext(t *testing.T) {
	x := newHX(t)
	wantRedirect(t, x.do("POST", x.link("new@a.edu", "/a/courses"), ""), "/sign-in?error=no_membership")
}

func TestOIDCStartDropsInvalidNext(t *testing.T) {
	x := newHX(t)
	for _, bad := range []string{"@evil.com", "//evil.com"} {
		c, nonce := x.oidcStart(`{"email":"admin@a.edu","next":"` + bad + `"}`)
		st, err := parseState(x.d.Secret, c.Value, time.Now())
		if err != nil || st.Next != "" {
			t.Fatalf("%q: state %+v err %v", bad, st, err)
		}
		x.iss.mint(t, map[string]any{"nonce": nonce, "email": "admin@a.edu", "email_verified": true})
		wantRedirect(t, x.do("GET", "/api/v1/auth/oidc/google/callback?code=x&state="+nonce, "", c), "/a/me")
	}
}

func TestMeActiveMembershipsAndOwnDepartments(t *testing.T) {
	x := newHX(t)
	f := x.fx
	tx := func(uni, q string, args ...any) {
		t.Helper()
		err := pg.WithTx(context.Background(), x.db, uni, func(tx *sql.Tx) error { _, err := tx.Exec(q, args...); return err })
		if err != nil {
			t.Fatal(err)
		}
	}
	sqlExec(t, x.db, `insert into users (email, display_name) values ('multi@a.edu', 'Multi'), ('inv@a.edu', 'Inv')`)
	for _, u := range []string{f.UniversityA, f.UniversityB} {
		tx(u, `insert into memberships (university_id, user_id, role, status) select $1, id, 'dept_admin', 'active' from users where email = 'multi@a.edu'`, u)
	}
	tx(f.UniversityB, `insert into departments (university_id, name, code) values ($1, 'Math', 'MA')`, f.UniversityB)
	tx(f.UniversityA, `insert into department_admins (university_id, department_id, user_id) select $1, $2, id from users where email = 'multi@a.edu'`, f.UniversityA, f.DeptA)
	tx(f.UniversityB, `insert into department_admins (university_id, department_id, user_id) select $1, d.id, u.id from departments d, users u where d.university_id = $1 and u.email = 'multi@a.edu'`, f.UniversityB)
	tx(f.UniversityA, `insert into memberships (university_id, user_id, role, status) select $1, id, 'member', 'active' from users where email = 'inv@a.edu'`, f.UniversityA)
	tx(f.UniversityB, `insert into memberships (university_id, user_id, role, status) select $1, id, 'member', 'invited' from users where email = 'inv@a.edu'`, f.UniversityB)

	var got struct {
		Memberships []struct {
			University    struct{ ID string }
			DepartmentIDs []string
		}
	}
	w := x.do("GET", "/api/v1/me", "", x.signedIn("multi@a.edu"))
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || len(got.Memberships) != 2 {
		t.Fatalf("%d %s %v", w.Code, w.Body, err)
	}
	for _, m := range got.Memberships {
		if len(m.DepartmentIDs) != 1 {
			t.Fatalf("each university lists only its own department: %s", w.Body)
		}
		if m.University.ID == f.UniversityA && m.DepartmentIDs[0] != f.DeptA {
			t.Fatalf("A lists foreign department: %s", w.Body)
		}
	}
	w = x.do("GET", "/api/v1/me", "", x.signedIn("inv@a.edu"))
	if strings.Contains(w.Body.String(), f.UniversityB) || !strings.Contains(w.Body.String(), f.UniversityA) {
		t.Fatalf("invited membership listed: %s", w.Body)
	}
}

func TestReSignInRevokesOldSession(t *testing.T) {
	x := newHX(t)
	first := cookieNamed(x.do("POST", x.link("admin@a.edu", ""), ""), SessionCookie)
	second := cookieNamed(x.do("POST", x.link("admin@a.edu", ""), "", first), SessionCookie)
	if w := x.do("GET", "/api/v1/me", "", first); w.Code != 401 {
		t.Fatalf("old session still valid: %d", w.Code)
	}
	if w := x.do("GET", "/api/v1/me", "", second); w.Code != 200 {
		t.Fatalf("new session: %d", w.Code)
	}
}

func TestRejectionLogsCarryRequestID(t *testing.T) {
	x := newHX(t)
	var buf bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	defer slog.SetDefault(old)
	x.do("POST", "/api/v1/auth/magic-link/garbage", "")
	x.do("GET", "/api/v1/auth/oidc/google/callback?code=x&state=tampered", "")
	for _, msg := range []string{"sign-in rejected", "oidc callback rejected"} {
		found := false
		for _, line := range strings.Split(buf.String(), "\n") {
			if strings.Contains(line, msg) {
				found = true
				if !strings.Contains(line, `"requestId":"`) || strings.Contains(line, `"requestId":""`) {
					t.Errorf("%s: no requestId: %s", msg, line)
				}
			}
		}
		if !found {
			t.Errorf("no %q log line in %s", msg, buf.String())
		}
	}
}

// blockMailer signals entered, then blocks until ctx is done; or always fails when fail is set.
type blockMailer struct {
	entered chan struct{}
	fail    bool
}

func (m *blockMailer) Send(ctx context.Context, _, _, _ string) error {
	if m.fail {
		return errors.New("smtp down")
	}
	select {
	case m.entered <- struct{}{}:
	default:
	}
	<-ctx.Done()
	return ctx.Err()
}

func TestMagicLinkDoesNotWaitForMailer(t *testing.T) {
	x := newHX(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	t.Cleanup(func() { cancel(); <-done })
	bm := &blockMailer{entered: make(chan struct{}, 1)}
	s := outbox.NewSender(x.db, bm, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	x.d.Magic.Wake = s.Wake
	go func() { s.Run(ctx); close(done) }()

	start := time.Now()
	if w := x.do("POST", "/api/v1/auth/magic-link", `{"email":"admin@a.edu"}`); w.Code != 202 || time.Since(start) > time.Second {
		t.Fatalf("%d after %v", w.Code, time.Since(start))
	}
	select {
	case <-bm.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("sender was not woken")
	}

}

func (x *hx) userByEmail(email string) (id, name string) {
	x.t.Helper()
	if err := pg.WithPlatformTx(context.Background(), x.db, func(tx *sql.Tx) error {
		return tx.QueryRow(`select id, display_name from users where email = $1`, email).Scan(&id, &name)
	}); err != nil {
		x.t.Fatal(err)
	}
	return
}

func TestUpdateMe(t *testing.T) {
	x := newHX(t)
	const path = "/api/v1/me"
	if w := x.do("PATCH", path, `{"displayName":"X"}`); w.Code != 401 || !strings.Contains(w.Body.String(), `"code":"unauthenticated"`) {
		t.Fatalf("no cookie: %d %s", w.Code, w.Body)
	}
	c := x.signedIn("admin@a.edu")
	w := x.do("PATCH", path, `{"displayName":"  Alice Doe  "}`, c)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"displayName":"Alice Doe"`) || !strings.Contains(w.Body.String(), `"departmentIds":[]`) {
		t.Fatalf("happy: %d %s", w.Code, w.Body)
	}
	if g := x.do("GET", path, "", c); g.Body.String() != w.Body.String() {
		t.Fatalf("GET /me differs from PATCH body: %s vs %s", g.Body, w.Body)
	}
	for body, msg := range map[string]string{
		`{"displayName":""}`:    "is required",
		`{"displayName":"   "}`: "is required",
		`{}`:                    "is required",
		`{"displayName":null}`:  "is required",
		`{"displayName":"` + strings.Repeat("é", 101) + `"}`: "must be at most 100 characters",
		`{"displayName":"a\u0000b"}`:                         "must not contain NUL",
	} {
		w := x.do("PATCH", path, body, c)
		if w.Code != 400 || !strings.Contains(w.Body.String(), `"validation_failed"`) || !strings.Contains(w.Body.String(), `"displayName"`) || !strings.Contains(w.Body.String(), msg) {
			t.Fatalf("%s: %d %s", body, w.Code, w.Body)
		}
	}
	if w := x.do("PATCH", path, `{"displayName":"`+strings.Repeat("é", 100)+`"}`, c); w.Code != 200 {
		t.Fatalf("100 runes: %d %s", w.Code, w.Body)
	}
	x.do("PATCH", path, `{"displayName":"Alice Doe"}`, c)
	// another user's name cannot be set
	adminB, _ := x.userByEmail("admin@b.edu")
	for _, body := range []string{`{"displayName":5}`, `{"displayName":"X","userId":"` + adminB + `"}`} {
		if w := x.do("PATCH", path, body, c); w.Code != 400 {
			t.Fatalf("%s: %d %s", body, w.Code, w.Body)
		}
	}
	if _, n := x.userByEmail("admin@b.edu"); n != "Admin" {
		t.Fatalf("admin B renamed to %q", n)
	}
	if n := len(auditRows(t, x.db, x.fx.UniversityB)); n != 0 {
		t.Fatalf("B audit rows %d", n)
	}
}

func TestUpdateMeAudit(t *testing.T) {
	x := newHX(t)
	c := x.signedIn("admin@a.edu")
	adminA, _ := x.userByEmail("admin@a.edu")
	var mA string
	if err := pg.WithTx(context.Background(), x.db, x.fx.UniversityA, func(tx *sql.Tx) error {
		return tx.QueryRow(`select id from memberships where user_id = $1`, adminA).Scan(&mA)
	}); err != nil {
		t.Fatal(err)
	}
	if w := x.do("PATCH", "/api/v1/me", `{"displayName":"Alice Doe"}`, c); w.Code != 200 {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	r := findAudit(t, auditRows(t, x.db, x.fx.UniversityA), "membership.update")
	if r.EntityType != "membership" || r.EntityID != mA || r.Actor != adminA || r.Before["displayName"] != "Admin" || r.After["displayName"] != "Alice Doe" {
		t.Fatalf("%+v", r)
	}
	if w := x.do("PATCH", "/api/v1/me", `{"displayName":"Alice Doe"}`, c); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if n := len(auditRows(t, x.db, x.fx.UniversityA)); n != 1 {
		t.Fatalf("unchanged name audited: %d rows", n)
	}
	// second active membership: one row per university
	if err := pg.WithTx(context.Background(), x.db, x.fx.UniversityB, func(tx *sql.Tx) error {
		_, err := tx.Exec(`insert into memberships (university_id, user_id, role, status) values ($1,$2,'member','active')`, x.fx.UniversityB, adminA)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if w := x.do("PATCH", "/api/v1/me", `{"displayName":"Alice Two"}`, c); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if a, b := len(auditRows(t, x.db, x.fx.UniversityA)), len(auditRows(t, x.db, x.fx.UniversityB)); a != 2 || b != 1 {
		t.Fatalf("audit rows A=%d B=%d", a, b)
	}
	// no membership: name saved, nothing audited
	n := x.signedIn("new@a.edu")
	if w := x.do("PATCH", "/api/v1/me", `{"displayName":"Newbie"}`, n); w.Code != 200 {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	if _, name := x.userByEmail("new@a.edu"); name != "Newbie" {
		t.Fatal(name)
	}
	if a := len(auditRows(t, x.db, x.fx.UniversityA)); a != 2 {
		t.Fatalf("A rows %d", a)
	}
}
