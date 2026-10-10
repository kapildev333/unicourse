package calendar

import (
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/tenant"
)

const base = "/api/v1/universities/{slug}"

// Register mounts Upcoming and the feed token routes (any active member, through tenant.Wrap) and the public feed.
func Register(r *httpx.Router, d tenant.Deps) {
	r.Handle("GET", base+"/me/upcoming", tenant.Wrap(d, upcoming))
	r.Handle("GET", base+"/me/calendar-feed", tenant.Wrap(d, getFeed))
	r.Handle("POST", base+"/me/calendar-feed", tenant.Wrap(d, createFeed(d)))
	r.Handle("DELETE", base+"/me/calendar-feed", tenant.Wrap(d, deleteFeed))
	r.Handle("GET", base+"/calendar-feeds/{token}", feed(d)) // not tenant.Wrap: the token is the credential
}
