package auth

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
)

// Limiter is a fixed-window limit kept in rate_buckets, so every API process shares one count per key.
// ponytail: fixed window, up to 2x max across a window edge; sliding windows if abuse shows up.
// ponytail: window clock is this process's clock (l.now), not the database's; skew of milliseconds shifts windows by milliseconds.
// ponytail: keys (address, email) are stored in clear for at most two windows; hash them if the privacy notice says otherwise.
type Limiter struct {
	max    int
	window time.Duration
	now    func() time.Time // tests move the clock

	mu     sync.Mutex
	pruned time.Time // last prune by this instance
}

func NewLimiter(max int, window time.Duration) *Limiter {
	return &Limiter{max: max, window: window, now: time.Now}
}

// Check records one hit for (bucket, key) and returns nil while the window holds at most max hits,
// httpx.RateLimited() once it holds more, or the database error (fail closed: the caller returns it, a 500).
// Hits over the limit are still counted.
// ponytail: one extra statement per sign-in, expired rows deleted hourly per process; a cron-style job only if the delete shows in slow logs.
func (l *Limiter) Check(ctx context.Context, db *sql.DB, bucket, key string) error {
	now := l.now()
	l.mu.Lock()
	due := now.Sub(l.pruned) >= l.window
	if due {
		l.pruned = now
	}
	l.mu.Unlock()
	if due {
		if _, err := db.ExecContext(ctx, `delete from rate_buckets where expires_at < $1`, now); err != nil {
			return err
		}
	}
	var hits int
	err := db.QueryRowContext(ctx, `
insert into rate_buckets (bucket, client_key, hits, expires_at, updated_at)
values ($1, $2, 1, $4, $3)
on conflict (bucket, client_key) do update set
  hits       = case when rate_buckets.expires_at <= $3 then 1 else rate_buckets.hits + 1 end,
  expires_at = case when rate_buckets.expires_at <= $3 then excluded.expires_at else rate_buckets.expires_at end,
  updated_at = $3
returning hits`, bucket, key, now, now.Add(l.window)).Scan(&hits)
	if err != nil {
		return err
	}
	if hits > l.max {
		return httpx.RateLimited()
	}
	return nil
}
