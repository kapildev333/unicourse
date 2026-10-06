// Package app wires every domain package's routes into one httpx.Router.
package app

import (
	"github.com/kapildev333/unicourse/apps/api/internal/auth"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

// Deps holds what domain packages need; later lanes add fields. Zero value is valid for route
// registration (the contract test), handlers only dereference at request time.
type Deps struct {
	Auth    auth.Deps
	Mailer  auth.Mailer
	BaseURL string // used for invitation URLs
}

// NewRouter registers every domain package's routes. Add each package's
// Register(r, deps) call here as its lane lands.
func NewRouter(deps Deps) *httpx.Router {
	r := httpx.NewRouter()
	auth.Register(r, deps.Auth)
	tenant.Register(r, tenant.Deps{DB: deps.Auth.DB, Sessions: deps.Auth.Sessions, Mailer: deps.Mailer, BaseURL: deps.BaseURL})
	return r
}
