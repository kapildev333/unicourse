# P1-L1b - Sign-in and OIDC (apps/api/internal/auth)
Status: done
## Doing
Finished; awaiting orchestrator verification.
## Found
Exported/shared surface as built (package `auth`):
- `SignIn(ctx, db *sql.DB, email, displayName string) (userID string, err error)`; `ErrNotAllowed`; `InvitationURL(baseURL, token, email string) string` -> `{baseURL}/invite/{token}?email={escaped}`. For P1-L2c `CreateInvitation`: `NewToken()` (P1-L1a) + `InvitationURL`; the invitation row's `token_hash` is the hash from `NewToken`.
- SignIn: one `pg.WithUserTx(ctx, db, "", email)`. Reads invitations by email BEFORE the first `pg.SetUniversity` (design 16.2), then per invitation: SetUniversity, claim (update accepted_at where unaccepted and unexpired), upsert membership (role only goes up), `section_staff` from `pending_section_staff`, `enrollments` from `pending_enrollments`, `department_admins` when role is dept_admin. Never sets `updated_at`. Concurrent sign-ins for one email accept each invitation exactly once (tested, 8 goroutines).
- `Provider{Name, OAuth oauth2.Config, Verifier}`; `NewProviders(ctx, cfg config.Config) (map[string]Provider, error)`; `newProvider(ctx, name, issuer, clientID, clientSecret, redirectURL)`.
- `(Provider).begin(secret []byte, slug, email, next string, now time.Time) (authURL, cookieValue string, err error)` - everything the start handler needs (nonce, PKCE, state==nonce, login_hint, prompt=select_account). Handler (P1-L1c) must: validate slug/email, `safeNext(next)`, call `begin`, `setOIDCCookie(w, cookieValue)`, answer `{url}`.
- `setOIDCCookie(w, value)` / `clearOIDCCookie(w)` (name `unicourse_oidc`, Path `/api/v1/auth/oidc`, 600 s, HttpOnly, Secure, Lax).
- `type Deps struct{ DB *sql.DB; Providers map[string]Provider; Secret []byte }` - ONLY the fields this sub-lane uses, in oidc.go. P1-L1c extends this struct (per design it lives in handler.go; move it there and add Sessions, Magic, WebBase, limiters).
- `func (d Deps) oidcCallback(w http.ResponseWriter, r *http.Request) (userID string, st state, err error)` - deviation from design 12: it does NOT create the session or redirect (landing is handler.go, P1-L1c). It reads `r.PathValue("provider")`, clears the state cookie, verifies state/provider/nonce/PKCE/id_token/claims, calls SignIn. Returns `errOIDC`-wrapped error for every verification failure (logged at Info with the cause; handler maps to `sign-in?error=unauthenticated`), `ErrNotAllowed` (-> `error=forbidden`), anything else is unexpected (500). On success the handler does `Sessions.Create`, `setSessionCookie`, 302 `landing(userID, st.Slug, st.Next)`. `st.Next` came from a signed cookie so it was validated by `safeNext` at start.
- `state{Provider, Slug, Next, Nonce, Verifier, Exp}`, `signState`, `parseState`, `checkClaims(provider string, c claims) error`.
- Microsoft rule: email non-empty, `xms_edov` true, `iss == https://login.microsoftonline.com/{tid}/v2.0`. Claim booleans accept `true`, `"true"`, `"1"` (`flexBool`).
- Test helpers: `insertInvitation(t, db, uni, email, role, deptIDs, expiresAt) (invitationID)` (returns the id, not a token: acceptance is by email), `insertSection`, `tcount`, `texec` (tenant-scoped SQL) in invite_test.go; `fakeIssuer` (`newFakeIssuer`, `mint`) and `newCB` in oidc_test.go.
- Modules added: `golang.org/x/oauth2 v0.37.0`, `github.com/coreos/go-oidc/v3 v3.21.0` (brings `go-jose/v4` indirect).
## Requests
- to P1-L1c: move/extend `Deps` as above; the HTTP-level OIDC cases (302 to `/a/me`, `next` landing) belong in handler tests since landing is yours.
## Blockers
None.
## Fix round (review of SignIn/OIDC)
- Tests added that kill mutations: removed membership/department_admins stay removed (drop `accepted_at is null`), `invited` -> `active` (drop `status = 'active'` from on-conflict), PKCE enforced by the fake `/token` (drop `VerifierOption`). Each mutation applied, new test failed, reverted.
- Ruling applied (deviates from design 7): `department_admins` rows are inserted only for ids that exist in the invitation's university; stale/foreign ids are skipped, no 23503.
- `verifyCallback` normalises the claim email; failure -> errOIDC (unauthenticated). `failRedirect` default branch logs and returns an explicit 500 `httpx.Error` (no 404/409 leak). `landing` runs `safeNext` on `next` (tested with a forged `//evil.com` state).
