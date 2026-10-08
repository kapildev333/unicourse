package auth

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/testdb"
)

func isLimited(err error) bool {
	var he *httpx.Error
	return errors.As(err, &he) && he.Status == 429
}

func hitsOf(t *testing.T, db *sql.DB, bucket, key string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`select hits from rate_buckets where bucket=$1 and client_key=$2`, bucket, key).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestLimiterSharedAcrossInstances(t *testing.T) {
	db := testdb.Open(t)
	ctx := context.Background()
	l1, l2 := NewLimiter(3, time.Hour), NewLimiter(3, time.Hour)
	for _, c := range []struct {
		l   *Limiter
		key string
	}{{l1, "a"}, {l2, "a"}, {l1, "b"}} {
		if err := c.l.Check(ctx, db, "magic-link-ip", c.key); err != nil {
			t.Fatal(err)
		}
	}
	if err := l1.Check(ctx, db, "magic-link-ip", "a"); err != nil {
		t.Fatal(err)
	}
	if !isLimited(l2.Check(ctx, db, "magic-link-ip", "a")) {
		t.Fatal("a: 4th hit across instances not limited")
	}
	if err := l2.Check(ctx, db, "magic-link-ip", "b"); err != nil {
		t.Fatal(err)
	}
	if err := l1.Check(ctx, db, "magic-link-ip", "b"); err != nil {
		t.Fatal(err)
	}
	if !isLimited(l2.Check(ctx, db, "magic-link-ip", "b")) {
		t.Fatal("b: 4th hit not limited")
	}
}

func TestLimiterWindowResets(t *testing.T) {
	db := testdb.Open(t)
	ctx := context.Background()
	now := time.Unix(1_000_000, 0)
	l := NewLimiter(5, time.Hour)
	l.now = func() time.Time { return now }
	for i := 0; i < 5; i++ {
		if err := l.Check(ctx, db, "magic-link-email", "x@a.edu"); err != nil {
			t.Fatalf("hit %d: %v", i+1, err)
		}
	}
	if !isLimited(l.Check(ctx, db, "magic-link-email", "x@a.edu")) {
		t.Fatal("6th allowed")
	}
	now = now.Add(59 * time.Minute)
	if !isLimited(l.Check(ctx, db, "magic-link-email", "x@a.edu")) {
		t.Fatal("window reset early")
	}
	now = now.Add(time.Minute)
	if err := l.Check(ctx, db, "magic-link-email", "x@a.edu"); err != nil {
		t.Fatalf("window did not reset: %v", err)
	}
	if n := hitsOf(t, db, "magic-link-email", "x@a.edu"); n != 1 {
		t.Fatalf("hits after reset = %d", n)
	}
}

func TestLimiterKeysAndBucketsIndependent(t *testing.T) {
	db := testdb.Open(t)
	ctx := context.Background()
	l := NewLimiter(1, time.Hour)
	if err := l.Check(ctx, db, "magic-link-email", "x@a.edu"); err != nil {
		t.Fatal(err)
	}
	if !isLimited(l.Check(ctx, db, "magic-link-email", "x@a.edu")) {
		t.Fatal("x not limited")
	}
	if err := l.Check(ctx, db, "magic-link-email", "y@a.edu"); err != nil {
		t.Fatalf("other key affected: %v", err)
	}
	if err := l.Check(ctx, db, "magic-link-ip", "x@a.edu"); err != nil {
		t.Fatalf("other bucket affected: %v", err)
	}
}

func TestLimiterConcurrent(t *testing.T) {
	db := testdb.Open(t)
	ctx := context.Background()
	ls := []*Limiter{NewLimiter(10, time.Hour), NewLimiter(10, time.Hour)}
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok, limited := 0, 0
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func(l *Limiter) {
			defer wg.Done()
			err := l.Check(ctx, db, "magic-link-ip", "1.2.3.4")
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				ok++
			case isLimited(err):
				limited++
			default:
				t.Error(err)
			}
		}(ls[i%2])
	}
	wg.Wait()
	if ok != 10 || limited != 30 {
		t.Fatalf("ok=%d limited=%d", ok, limited)
	}
	if n := hitsOf(t, db, "magic-link-ip", "1.2.3.4"); n != 40 {
		t.Fatalf("hits = %d", n)
	}
}

func TestLimiterIPMax(t *testing.T) {
	db := testdb.Open(t)
	ctx := context.Background()
	l := NewLimiter(300, time.Hour)
	for i := 0; i < 300; i++ {
		if err := l.Check(ctx, db, "magic-link-ip", "1.2.3.4"); err != nil {
			t.Fatalf("hit %d: %v", i+1, err)
		}
	}
	if !isLimited(l.Check(ctx, db, "magic-link-ip", "1.2.3.4")) {
		t.Fatal("301st allowed")
	}
}

func TestLimiterPrunes(t *testing.T) {
	db := testdb.Open(t)
	ctx := context.Background()
	t0 := time.Unix(1_000_000, 0)
	now := t0
	l := NewLimiter(5, time.Hour)
	l.now = func() time.Time { return now }
	count := func(key string) int {
		var n int
		if err := db.QueryRow(`select count(*) from rate_buckets where client_key=$1`, key).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if err := l.Check(ctx, db, "b", "k1"); err != nil {
		t.Fatal(err)
	}
	now = t0.Add(2 * time.Hour)
	if err := l.Check(ctx, db, "b", "k2"); err != nil {
		t.Fatal(err)
	}
	if count("k1") != 0 || count("k2") != 1 {
		t.Fatalf("k1=%d k2=%d", count("k1"), count("k2"))
	}
	if _, err := db.Exec(`insert into rate_buckets (bucket, client_key, hits, expires_at) values ('b', 'old', 1, $1)`, t0); err != nil {
		t.Fatal(err)
	}
	now = t0.Add(2*time.Hour + time.Minute)
	if err := l.Check(ctx, db, "b", "k3"); err != nil {
		t.Fatal(err)
	}
	if count("old") != 1 {
		t.Fatal("pruned twice within one window")
	}
}

func TestLimiterFailsClosed(t *testing.T) {
	db := testdb.Open(t)
	db.Close()
	err := NewLimiter(5, time.Hour).Check(context.Background(), db, "magic-link-ip", "1.2.3.4")
	var he *httpx.Error
	if err == nil || errors.As(err, &he) {
		t.Fatalf("want a non-httpx error, got %v", err)
	}
}
