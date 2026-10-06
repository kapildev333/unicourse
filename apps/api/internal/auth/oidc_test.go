package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
	"github.com/kapildev333/unicourse/apps/api/internal/testdb"
)

var testSecret = []byte("0123456789abcdef0123456789abcdef")

func TestStateRoundTripAndTamper(t *testing.T) {
	now := time.Now()
	st := state{Provider: "google", Slug: "a", Next: "/a/admin", Nonce: "n", Verifier: "v", Exp: now.Add(time.Minute).Unix()}
	c := signState(testSecret, st)
	got, err := parseState(testSecret, c, now)
	if err != nil || got != st {
		t.Fatalf("round trip: %v %+v", err, got)
	}
	dot := strings.LastIndexByte(c, '.')
	forged := base64.RawURLEncoding.EncodeToString([]byte(`{"p":"google","s":"b","x":"","n":"n","v":"v","e":99999999999}`))
	for name, bad := range map[string]string{
		"payload tampered": forged + c[dot:],
		"mac tampered":     c[:dot+1] + "AAAA" + c[dot+5:],
		"no dot":           "abc",
		"empty":            "",
	} {
		if _, err := parseState(testSecret, bad, now); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := parseState([]byte("another secret another secret!!!"), c, now); err == nil {
		t.Error("wrong secret accepted")
	}
	if _, err := parseState(testSecret, c, now.Add(2*time.Minute)); err == nil {
		t.Error("expired state accepted")
	}
	// begin sets a 10 minute expiry.
	exp := signState(testSecret, state{Exp: now.Add(-time.Second).Unix()})
	if _, err := parseState(testSecret, exp, now); err == nil {
		t.Error("past exp accepted")
	}
}

func TestCheckClaims(t *testing.T) {
	msIss := "https://login.microsoftonline.com/t1/v2.0"
	cases := []struct {
		name, provider string
		c              claims
		ok             bool
	}{
		{"google ok", "google", claims{Email: "a@a.edu", EmailVerified: true}, true},
		{"google unverified", "google", claims{Email: "a@a.edu"}, false},
		{"google no email", "google", claims{EmailVerified: true}, false},
		{"microsoft ok", "microsoft", claims{Email: "a@a.edu", XmsEdov: true, Tid: "t1", Iss: msIss}, true},
		{"microsoft no edov", "microsoft", claims{Email: "a@a.edu", Tid: "t1", Iss: msIss}, false},
		{"microsoft email_verified is not enough", "microsoft", claims{Email: "a@a.edu", EmailVerified: true, Tid: "t1", Iss: msIss}, false},
		{"microsoft wrong issuer", "microsoft", claims{Email: "a@a.edu", XmsEdov: true, Tid: "t1", Iss: "https://login.microsoftonline.com/t2/v2.0"}, false},
		{"microsoft no tid", "microsoft", claims{Email: "a@a.edu", XmsEdov: true, Iss: msIss}, false},
		{"unknown provider", "github", claims{Email: "a@a.edu", EmailVerified: true}, false},
	}
	for _, tc := range cases {
		err := checkClaims(tc.provider, tc.c)
		if (err == nil) != tc.ok || (err != nil && !errors.Is(err, errOIDC)) {
			t.Errorf("%s: %v", tc.name, err)
		}
	}
	var c claims
	if err := json.Unmarshal([]byte(`{"email_verified":"true","xms_edov":"1"}`), &c); err != nil || !c.EmailVerified || !c.XmsEdov {
		t.Errorf("flexBool: %v %+v", err, c)
	}
}

// fakeIssuer serves discovery, JWKS and a token endpoint that returns whatever idToken is set to.
type fakeIssuer struct {
	*httptest.Server
	key *rsa.PrivateKey
	mu  sync.Mutex
	tok string
	// challenge, when set, is the code_challenge the token endpoint requires a matching code_verifier for.
	challenge string
}

func newFakeIssuer(t *testing.T) *fakeIssuer {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeIssuer{key: key}
	mux := http.NewServeMux()
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	b64 := base64.RawURLEncoding.EncodeToString
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"issuer": f.URL, "authorization_endpoint": f.URL + "/authorize",
			"token_endpoint": f.URL + "/token", "jwks_uri": f.URL + "/jwks", "id_token_signing_alg_values_supported": []string{"RS256"}})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]any{{"kty": "RSA", "kid": "k1", "alg": "RS256", "use": "sig",
			"n": b64(key.N.Bytes()), "e": b64(big.NewInt(int64(key.E)).Bytes())}}})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.challenge != "" {
			sum := sha256.Sum256([]byte(r.FormValue("code_verifier")))
			if b64(sum[:]) != f.challenge {
				http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"access_token": "x", "token_type": "Bearer", "expires_in": 3600, "id_token": f.tok})
	})
	return f
}

// mint signs an ID token for clientID "client" and makes it the next token-endpoint response.
func (f *fakeIssuer) mint(t *testing.T, claims map[string]any) {
	t.Helper()
	b64 := base64.RawURLEncoding.EncodeToString
	full := map[string]any{"iss": f.URL, "aud": "client", "sub": "s1", "iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix()}
	for k, v := range claims {
		if v == nil {
			delete(full, k)
		} else {
			full[k] = v
		}
	}
	h, _ := json.Marshal(map[string]any{"alg": "RS256", "kid": "k1", "typ": "JWT"})
	c, _ := json.Marshal(full)
	in := b64(h) + "." + b64(c)
	sum := sha256.Sum256([]byte(in))
	sig, err := rsa.SignPKCS1v15(rand.Reader, f.key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	f.tok = in + "." + b64(sig)
	f.mu.Unlock()
}

type cbFixture struct {
	iss    *fakeIssuer
	d      Deps
	db     *sql.DB
	fx     testdb.Fixture
	cookie string
	nonce  string
}

// startFlow runs Provider.begin against the fake issuer and returns the cookie and the nonce the IdP saw.
func newCB(t *testing.T) *cbFixture {
	t.Helper()
	db := testdb.Open(t)
	fx := testdb.Seed(t, db)
	iss := newFakeIssuer(t)
	p, err := newProvider(context.Background(), "google", iss.URL, "client", "secret", "http://api.test/api/v1/auth/oidc/google/callback")
	if err != nil {
		t.Fatal(err)
	}
	f := &cbFixture{iss: iss, db: db, fx: fx, d: Deps{DB: db, Providers: map[string]Provider{"google": p}, Secret: testSecret}}
	u, cookie, err := p.begin(testSecret, "a", "x@a.edu", "/a/admin", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	pu, _ := url.Parse(u)
	q := pu.Query()
	if q.Get("code_challenge") == "" || q.Get("code_challenge_method") != "S256" || q.Get("nonce") == "" || q.Get("state") != q.Get("nonce") || q.Get("login_hint") != "x@a.edu" {
		t.Fatalf("auth url %s", u)
	}
	f.cookie, f.nonce = cookie, q.Get("nonce")
	iss.challenge = q.Get("code_challenge") // the fake token endpoint enforces PKCE against it
	return f
}

func (f *cbFixture) call(provider, cookie, stateParam string) (string, state, error, *httptest.ResponseRecorder) {
	r := httptest.NewRequest("GET", "/api/v1/auth/oidc/"+provider+"/callback?code=x&state="+url.QueryEscape(stateParam), nil)
	r.SetPathValue("provider", provider)
	if cookie != "" {
		r.AddCookie(&http.Cookie{Name: oidcCookie, Value: cookie})
	}
	w := httptest.NewRecorder()
	uid, st, err := f.d.oidcCallback(w, r)
	return uid, st, err, w
}

func TestOIDCCallbackHappyPath(t *testing.T) {
	f := newCB(t)
	f.iss.mint(t, map[string]any{"nonce": f.nonce, "email": "Zed@A.edu", "email_verified": true, "name": "Zed Z"})
	uid, st, err, w := f.call("google", f.cookie, f.nonce)
	if err != nil {
		t.Fatal(err)
	}
	if st.Next != "/a/admin" || st.Slug != "a" {
		t.Fatalf("state %+v", st)
	}
	var email, name string
	if err := pg.WithPlatformTx(context.Background(), f.db, func(tx *sql.Tx) error {
		return tx.QueryRow(`select email, display_name from users where id = $1`, uid).Scan(&email, &name)
	}); err != nil || email != "zed@a.edu" || name != "Zed Z" {
		t.Fatalf("user %q %q %v", email, name, err)
	}
	if c := w.Result().Cookies(); len(c) != 1 || c[0].Name != oidcCookie || c[0].MaxAge >= 0 {
		t.Fatalf("state cookie not cleared: %v", c)
	}
}

func TestOIDCCallbackRejections(t *testing.T) {
	good := func(f *cbFixture) map[string]any {
		return map[string]any{"nonce": f.nonce, "email": "zed@a.edu", "email_verified": true, "name": "Z"}
	}
	cases := []struct {
		name, reason string
		run          func(f *cbFixture) error
	}{
		{"wrong nonce in id token", "nonce mismatch", func(f *cbFixture) error {
			c := good(f)
			c["nonce"] = "other"
			f.iss.mint(t, c)
			_, _, err, _ := f.call("google", f.cookie, f.nonce)
			return err
		}},
		{"state param differs from cookie", "state parameter", func(f *cbFixture) error {
			f.iss.mint(t, good(f))
			_, _, err, _ := f.call("google", f.cookie, "other")
			return err
		}},
		{"email_verified false", "not verified", func(f *cbFixture) error {
			c := good(f)
			c["email_verified"] = false
			f.iss.mint(t, c)
			_, _, err, _ := f.call("google", f.cookie, f.nonce)
			return err
		}},
		{"cookie missing", "cookie missing", func(f *cbFixture) error {
			f.iss.mint(t, good(f))
			_, _, err, _ := f.call("google", "", f.nonce)
			return err
		}},
		{"cookie tampered", "signature", func(f *cbFixture) error {
			f.iss.mint(t, good(f))
			_, _, err, _ := f.call("google", "x"+f.cookie, f.nonce)
			return err
		}},
		{"cookie for another provider", "provider mismatch", func(f *cbFixture) error {
			f.iss.mint(t, good(f))
			f.d.Providers["microsoft"] = f.d.Providers["google"]
			_, _, err, _ := f.call("microsoft", f.cookie, f.nonce)
			return err
		}},
		{"expired state", "expired", func(f *cbFixture) error {
			f.iss.mint(t, good(f))
			st, _ := parseState(testSecret, f.cookie, time.Now())
			st.Exp = time.Now().Add(-time.Minute).Unix()
			_, _, err, _ := f.call("google", signState(testSecret, st), f.nonce)
			return err
		}},
		{"wrong audience", "id_token", func(f *cbFixture) error {
			c := good(f)
			c["aud"] = "someone-else"
			f.iss.mint(t, c)
			_, _, err, _ := f.call("google", f.cookie, f.nonce)
			return err
		}},
		{"expired id token", "id_token", func(f *cbFixture) error {
			c := good(f)
			c["exp"] = time.Now().Add(-time.Hour).Unix()
			f.iss.mint(t, c)
			_, _, err, _ := f.call("google", f.cookie, f.nonce)
			return err
		}},
		{"no id token", "no id_token", func(f *cbFixture) error {
			f.iss.tok = ""
			_, _, err, _ := f.call("google", f.cookie, f.nonce)
			return err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newCB(t)
			if err := tc.run(f); !errors.Is(err, errOIDC) || !strings.Contains(err.Error(), tc.reason) {
				t.Fatalf("want errOIDC, got %v", err)
			}
			if n := sqlCount(t, f.db, `select count(*) from users where email = 'zed@a.edu'`); n != 0 {
				t.Fatal("user created on a rejected callback")
			}
		})
	}
}

func TestOIDCCallbackNotAllowed(t *testing.T) {
	f := newCB(t)
	f.iss.mint(t, map[string]any{"nonce": f.nonce, "email": "x@other.org", "email_verified": true})
	if _, _, err, _ := f.call("google", f.cookie, f.nonce); !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("want ErrNotAllowed, got %v", err)
	}
}

func TestOIDCCallbackPKCEEnforced(t *testing.T) {
	f := newCB(t)
	f.iss.mint(t, map[string]any{"nonce": f.nonce, "email": "pk@a.edu", "email_verified": true})
	if _, _, err, _ := f.call("google", f.cookie, f.nonce); err != nil {
		t.Fatalf("matching verifier rejected: %v", err)
	}
	// A cookie whose verifier does not match the challenge the IdP saw must fail the exchange.
	st, _ := parseState(testSecret, f.cookie, time.Now())
	st.Verifier = "wrong-verifier-wrong-verifier-wrong-verifier-123"
	if _, _, err, _ := f.call("google", signState(testSecret, st), f.nonce); !errors.Is(err, errOIDC) || !strings.Contains(err.Error(), "code exchange") {
		t.Fatalf("want code exchange failure, got %v", err)
	}
}

func TestOIDCCallbackBadEmailClaim(t *testing.T) {
	f := newCB(t)
	f.iss.mint(t, map[string]any{"nonce": f.nonce, "email": "not an email", "email_verified": true})
	if _, _, err, _ := f.call("google", f.cookie, f.nonce); !errors.Is(err, errOIDC) || !strings.Contains(err.Error(), "email claim") {
		t.Fatalf("want errOIDC, got %v", err)
	}
}

func TestFailRedirectUnexpectedIs500(t *testing.T) {
	x := newHX(t)
	for _, in := range []error{pg.ErrNotFound, pg.ErrConflict, errors.New("boom")} {
		w := httptest.NewRecorder()
		err := x.d.failRedirect(w, httptest.NewRequest("GET", "/", nil), in)
		var he *httpx.Error
		if !errors.As(err, &he) || he.Status != 500 || errors.Is(err, pg.ErrNotFound) || errors.Is(err, pg.ErrConflict) {
			t.Errorf("%v -> %v", in, err)
		}
	}
}

func TestLandingSanitisesNextFromState(t *testing.T) {
	x := newHX(t)
	_, nonce := x.oidcStart(`{"slug":"a"}`)
	st := state{Provider: "google", Slug: "a", Next: "//evil.com", Nonce: nonce, Verifier: "v", Exp: time.Now().Add(time.Minute).Unix()}
	x.iss.mint(t, map[string]any{"nonce": nonce, "email": "admin@a.edu", "email_verified": true})
	w := x.do("GET", "/api/v1/auth/oidc/google/callback?code=x&state="+nonce, "", &http.Cookie{Name: oidcCookie, Value: signState(testSecret, st)})
	wantRedirect(t, w, "/a/me")
}
