package calendar

import (
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

const base = "/api/v1/universities/{slug}"

// Register mounts Upcoming (any active member, through tenant.Wrap).
func Register(r *httpx.Router, d tenant.Deps) {
	r.Handle("GET", base+"/me/upcoming", tenant.Wrap(d, upcoming))
}
