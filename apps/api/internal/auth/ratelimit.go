package auth

import (
	"sync"
	"time"
)

// ponytail: in-memory limiter, move to Postgres if more than one API process
// Fixed window per key; ceiling is 2x burst across a window edge.
type Limiter struct {
	mu     sync.Mutex
	max    int
	window time.Duration
	hits   map[string]*bucket
	now    func() time.Time
}

type bucket struct {
	start time.Time
	n     int
}

func NewLimiter(max int, window time.Duration) *Limiter {
	return &Limiter{max: max, window: window, hits: map[string]*bucket{}, now: time.Now}
}

// Allow records a hit for key and reports whether it is within the limit.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if len(l.hits) > 10000 {
		for k, b := range l.hits {
			if now.Sub(b.start) >= l.window {
				delete(l.hits, k)
			}
		}
	}
	b := l.hits[key]
	if b == nil || now.Sub(b.start) >= l.window {
		b = &bucket{start: now}
		l.hits[key] = b
	}
	if b.n >= l.max {
		return false
	}
	b.n++
	return true
}
