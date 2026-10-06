# P1-L1a - Auth primitives (apps/api/internal/auth)
Status: done
## Doing
Finished; awaiting orchestrator verification.
## Found
Exported surface as built (package `auth`):
- `type Actor struct{ UserID, Email string }`; `FromContext(ctx) (Actor, bool)`; `withActor(ctx, Actor) context.Context` (unexported)
- `NormalizeEmail(s string) (string, error)` (validation_failed, field `email`)
- `NewToken() (token string, hash []byte, err error)` (43 chars, raw 32-byte sha256); unexported `hashToken(string) []byte`, `validToken(string) bool`
- `const SessionCookie = "unicourse_session"`; `type SessionStore struct{ DB *sql.DB }`
  - `Create(ctx, userID) (token string, err error)`; `Lookup(ctx, token) (Actor, slid bool, err error)` (unknown/expired -> `httpx.Unauthenticated()`; expired row deleted); `Revoke(ctx, token) error`
- `Authenticate(w, r, *SessionStore) (Actor, error)` (cookie, slide + re-set cookie, `httpx.WithUserID`); `RequireSession(*SessionStore) func(http.Handler) http.Handler` (puts actor in ctx via `withActor`, 401 envelope)
- unexported `setSessionCookie(w, token)`, `clearSessionCookie(w)` (HttpOnly, Secure, Lax, Path /, 30 d)
- `Mailer` interface `Send(ctx, to, subject, text) error`; `LogMailer{L *slog.Logger}` (value receiver); `NewSMTPMailer(rawURL) (Mailer, error)` (requires host:port and ?from=; dial 10 s, whole dialogue 30 s or ctx deadline, STARTTLS when offered)
- `Limiter`, `NewLimiter(max int, window time.Duration)`, `(*Limiter).Allow(key) bool`; wiring numbers (5/email/h, 300/IP/h) are P1-L1c's job
- `MagicLinks{DB, Mailer, BaseURL}`; `Send(ctx, email, next string) error` (normalises; not-allowed -> nil, no row, no mail; `next` appended as `?next=` query-escaped, caller must `safeNext` it); `Consume(ctx, token) (email, err)` (400 validation_failed: "invalid link" / "link already used" / "link expired")
- `ErrNotAllowed`; unexported `allowed(ctx, tx, email) (bool, error)` (tx must be `pg.WithUserTx(ctx, db, "", email, ...)`). `SignIn`, `InvitationURL` are P1-L1b.
- Test helpers in package (reusable by b/c): `lastURL(t, *bytes.Buffer)` (mailer_test.go), `sqlExec`, `sqlCount`, `isUnauth` (session_test.go), `newMagic`, `tokenFromURL`, `wantLinkErr` (magic_test.go).
- Per section 16.7 `updated_at` is never set in SQL. Magic-link URL in mail is the GET URL; the D11 interstitial/POST is handler work (P1-L1c).
## Requests
- to P1-L1b/c: none beyond the surface above.
## Blockers
None.
