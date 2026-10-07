// Package app wires every domain package's routes into one httpx.Router.
package app

import (
	"github.com/kapildev333/unicourse/apps/api/internal/auth"
	"github.com/kapildev333/unicourse/apps/api/internal/blob"
	"github.com/kapildev333/unicourse/apps/api/internal/content"
	"github.com/kapildev333/unicourse/apps/api/internal/course"
	"github.com/kapildev333/unicourse/apps/api/internal/grade"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

// Deps holds what domain packages need; later lanes add fields. Zero value is valid for route
// registration (the contract test), handlers only dereference at request time.
type Deps struct {
	Auth    auth.Deps
	BaseURL string   // used for invitation URLs
	Files   blob.Dir // UNICOURSE_FILES_DIR
}

// NewRouter registers every domain package's routes. Add each package's
// Register(r, deps) call here as its lane lands.
func NewRouter(deps Deps) *httpx.Router {
	r := httpx.NewRouter()
	auth.Register(r, deps.Auth)
	td := tenant.Deps{DB: deps.Auth.DB, Sessions: deps.Auth.Sessions, BaseURL: deps.BaseURL, Files: deps.Files}
	tenant.Register(r, td)
	course.Register(r, td)
	grade.Register(r, td)
	content.Register(r, td)
	return r
}
