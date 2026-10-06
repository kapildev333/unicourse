// Package app wires every domain package's routes into one httpx.Router.
package app

import "github.com/kapildev333/unicourse/apps/api/internal/httpx"

// Deps holds what domain packages need; later lanes add fields.
type Deps struct{}

// NewRouter registers every domain package's routes. Add each package's
// Register(r, deps) call here as its lane lands.
func NewRouter(deps Deps) *httpx.Router {
	return httpx.NewRouter()
}
