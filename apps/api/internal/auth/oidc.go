package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/kapildev333/unicourse/apps/api/internal/config"
)

const (
	oidcCookie = "unicourse_oidc"
	oidcPath   = "/api/v1/auth/oidc"
	stateTTL   = 10 * time.Minute
)

// errOIDC marks every callback failure that the browser sees as sign-in?error=unauthenticated. The cause
// is logged at Info by oidcCallback and kept in the chain for tests.
var errOIDC = errors.New("oidc sign-in failed")

func oidcFail(format string, args ...any) error {
	return fmt.Errorf("%w: %s", errOIDC, fmt.Sprintf(format, args...))
}

type Provider struct {
	Name     string
	OAuth    oauth2.Config
	Verifier *oidc.IDTokenVerifier
}

// NewProviders builds the providers whose client id is configured; either may be absent.
func NewProviders(ctx context.Context, cfg config.Config) (map[string]Provider, error) {
	ps := map[string]Provider{}
	redirect := func(name string) string { return cfg.BaseURL + oidcPath + "/" + name + "/callback" }
	if cfg.GoogleClientID != "" {
		p, err := newProvider(ctx, "google", "https://accounts.google.com", cfg.GoogleClientID, cfg.GoogleClientSecret, redirect("google"))
		if err != nil {
			return nil, err
		}
		ps["google"] = p
	}
	if cfg.MicrosoftClientID != "" {
		p, err := newProvider(ctx, "microsoft", "https://login.microsoftonline.com/common/v2.0", cfg.MicrosoftClientID, cfg.MicrosoftClientSecret, redirect("microsoft"))
		if err != nil {
			return nil, err
		}
		ps["microsoft"] = p
	}
	return ps, nil
}

func newProvider(ctx context.Context, name, issuer, clientID, clientSecret, redirectURL string) (Provider, error) {
	vc := &oidc.Config{ClientID: clientID}
	if name == "microsoft" {
		// The common discovery document advertises a templated issuer; the real one is checked per token.
		ctx = oidc.InsecureIssuerURLContext(ctx, "https://login.microsoftonline.com/{tenantid}/v2.0")
		vc.SkipIssuerCheck = true
	}
	op, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return Provider{}, fmt.Errorf("oidc discovery for %s: %w", name, err)
	}
	return Provider{
		Name: name,
		OAuth: oauth2.Config{
			ClientID: clientID, ClientSecret: clientSecret, RedirectURL: redirectURL,
			Endpoint: op.Endpoint(), Scopes: []string{oidc.ScopeOpenID, "email", "profile"},
		},
		Verifier: op.Verifier(vc),
	}, nil
}

type state struct {
	Provider string `json:"p"`
	Slug     string `json:"s"`
	Next     string `json:"x"`
	Nonce    string `json:"n"`
	Verifier string `json:"v"`
	Exp      int64  `json:"e"`
}

func macOf(secret []byte, payload string) string {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func signState(secret []byte, s state) string {
	b, _ := json.Marshal(s) // a struct of strings and an int cannot fail
	p := base64.RawURLEncoding.EncodeToString(b)
	return p + "." + macOf(secret, p)
}

func parseState(secret []byte, cookie string, now time.Time) (state, error) {
	i := strings.LastIndexByte(cookie, '.')
	if i < 0 {
		return state{}, errors.New("malformed state")
	}
	p, mac := cookie[:i], cookie[i+1:]
	if !hmac.Equal([]byte(mac), []byte(macOf(secret, p))) {
		return state{}, errors.New("bad state signature")
	}
	b, err := base64.RawURLEncoding.DecodeString(p)
	if err != nil {
		return state{}, errors.New("malformed state")
	}
	var s state
	if err := json.Unmarshal(b, &s); err != nil {
		return state{}, errors.New("malformed state")
	}
	if s.Exp < now.Unix() {
		return state{}, errors.New("state expired")
	}
	return s, nil
}

// begin starts an authorization-code flow: it returns the IdP URL and the signed value for the
// unicourse_oidc cookie. email, when given, is passed as login_hint.
func (p Provider) begin(secret []byte, slug, email, next string, now time.Time) (authURL, cookieValue string, err error) {
	n := make([]byte, 32)
	if _, err := rand.Read(n); err != nil {
		return "", "", err
	}
	st := state{Provider: p.Name, Slug: slug, Next: next, Nonce: base64.RawURLEncoding.EncodeToString(n),
		Verifier: oauth2.GenerateVerifier(), Exp: now.Add(stateTTL).Unix()}
	opts := []oauth2.AuthCodeOption{oidc.Nonce(st.Nonce), oauth2.S256ChallengeOption(st.Verifier), oauth2.SetAuthURLParam("prompt", "select_account")}
	if email != "" {
		opts = append(opts, oauth2.SetAuthURLParam("login_hint", email))
	}
	return p.OAuth.AuthCodeURL(st.Nonce, opts...), signState(secret, st), nil
}

func setOIDCCookie(w http.ResponseWriter, value string) {
	http.SetCookie(w, &http.Cookie{Name: oidcCookie, Value: value, Path: oidcPath, MaxAge: int(stateTTL.Seconds()),
		HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode})
}

func clearOIDCCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: oidcCookie, Value: "", Path: oidcPath, MaxAge: -1,
		HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode})
}

// flexBool accepts true, "true" and "1": providers differ on how they encode boolean claims.
type flexBool bool

func (b *flexBool) UnmarshalJSON(d []byte) error {
	switch strings.Trim(string(d), `"`) {
	case "true", "1":
		*b = true
	default:
		*b = false
	}
	return nil
}

type claims struct {
	Iss           string   `json:"iss"`
	Email         string   `json:"email"`
	EmailVerified flexBool `json:"email_verified"`
	Name          string   `json:"name"`
	Tid           string   `json:"tid"`
	XmsEdov       flexBool `json:"xms_edov"`
}

// checkClaims applies the per-provider rule that makes the email trustworthy.
func checkClaims(provider string, c claims) error {
	if c.Email == "" {
		return oidcFail("no email claim")
	}
	switch provider {
	case "google":
		if !c.EmailVerified {
			return oidcFail("email not verified")
		}
	case "microsoft":
		if !c.XmsEdov {
			return oidcFail("email domain not verified by owner (xms_edov)")
		}
		if c.Tid == "" || c.Iss != "https://login.microsoftonline.com/"+c.Tid+"/v2.0" {
			return oidcFail("unexpected issuer %q for tenant %q", c.Iss, c.Tid)
		}
	default:
		return oidcFail("unknown provider %q", provider)
	}
	return nil
}

// Deps carries what the OIDC callback core needs; P1-L1c adds the rest.
type Deps struct {
	DB        *sql.DB
	Providers map[string]Provider
	Secret    []byte
}

// oidcCallback validates the callback of provider {provider} and signs the user in. It clears the state
// cookie and returns the user and the state (slug, next) for the caller to create the session and pick
// the landing page. Errors: errOIDC (any verification failure; logged at Info), ErrNotAllowed, or an
// unexpected error.
func (d Deps) oidcCallback(w http.ResponseWriter, r *http.Request) (userID string, st state, err error) {
	userID, st, err = d.verifyCallback(w, r)
	if errors.Is(err, errOIDC) {
		slog.InfoContext(r.Context(), "oidc callback rejected", "reason", err.Error())
	}
	return userID, st, err
}

func (d Deps) verifyCallback(w http.ResponseWriter, r *http.Request) (string, state, error) {
	ctx := r.Context()
	c, cerr := r.Cookie(oidcCookie)
	clearOIDCCookie(w)
	if cerr != nil {
		return "", state{}, oidcFail("state cookie missing")
	}
	st, err := parseState(d.Secret, c.Value, time.Now())
	if err != nil {
		return "", state{}, oidcFail("%v", err)
	}
	name := r.PathValue("provider")
	p, ok := d.Providers[name]
	if !ok || st.Provider != name {
		return "", state{}, oidcFail("provider mismatch (path %q, state %q)", name, st.Provider)
	}
	q := r.URL.Query()
	if e := q.Get("error"); e != "" {
		return "", state{}, oidcFail("idp error %q", e)
	}
	if q.Get("state") != st.Nonce {
		return "", state{}, oidcFail("state parameter does not match cookie")
	}
	tok, err := p.OAuth.Exchange(ctx, q.Get("code"), oauth2.VerifierOption(st.Verifier))
	if err != nil {
		return "", state{}, oidcFail("code exchange: %v", err)
	}
	raw, _ := tok.Extra("id_token").(string)
	if raw == "" {
		return "", state{}, oidcFail("no id_token in token response")
	}
	idt, err := p.Verifier.Verify(ctx, raw)
	if err != nil {
		return "", state{}, oidcFail("id_token: %v", err)
	}
	if idt.Nonce != st.Nonce {
		return "", state{}, oidcFail("nonce mismatch")
	}
	var cl claims
	if err := idt.Claims(&cl); err != nil {
		return "", state{}, oidcFail("claims: %v", err)
	}
	if err := checkClaims(p.Name, cl); err != nil {
		return "", state{}, err
	}
	userID, err := SignIn(ctx, d.DB, cl.Email, cl.Name)
	return userID, st, err
}
