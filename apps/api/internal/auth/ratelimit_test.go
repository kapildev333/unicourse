package auth

import (
	"testing"
	"time"
)

func TestLimiter(t *testing.T) {
	now := time.Unix(1000, 0)
	l := NewLimiter(5, time.Hour)
	l.now = func() time.Time { return now }
	for i := 0; i < 5; i++ {
		if !l.Allow("email:x") {
			t.Fatalf("hit %d denied", i+1)
		}
	}
	if l.Allow("email:x") {
		t.Fatal("6th allowed")
	}
	if !l.Allow("email:y") {
		t.Fatal("other key affected")
	}
	now = now.Add(59 * time.Minute)
	if l.Allow("email:x") {
		t.Fatal("window reset early")
	}
	now = now.Add(time.Minute)
	if !l.Allow("email:x") {
		t.Fatal("window did not reset after 1h")
	}
}

func TestLimiterIPMax(t *testing.T) {
	l := NewLimiter(300, time.Hour)
	for i := 0; i < 300; i++ {
		if !l.Allow("ip:1.2.3.4") {
			t.Fatalf("hit %d denied", i+1)
		}
	}
	if l.Allow("ip:1.2.3.4") {
		t.Fatal("301st allowed")
	}
}

func TestLimiterSweep(t *testing.T) {
	now := time.Unix(0, 0)
	l := NewLimiter(1, time.Hour)
	l.now = func() time.Time { return now }
	for i := 0; i < 10001; i++ {
		l.Allow(string(rune('a'+i%26)) + time.Duration(i).String())
	}
	now = now.Add(2 * time.Hour)
	l.Allow("fresh")
	if len(l.hits) != 1 {
		t.Fatalf("expired buckets not swept: %d", len(l.hits))
	}
}
