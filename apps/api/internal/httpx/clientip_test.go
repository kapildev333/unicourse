package httpx

import (
	"net/http/httptest"
	"net/netip"
	"testing"
)

func TestClientIP(t *testing.T) {
	tr := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8"), netip.MustParsePrefix("2001:db8:ffff::/48"), netip.MustParsePrefix("fe80::/10")}
	for i, c := range []struct {
		remote string
		xff    []string
		none   bool
		want   string
	}{
		{"203.0.113.7:5000", []string{"1.2.3.4"}, true, "203.0.113.7"},
		{"198.51.100.9:1", []string{"1.2.3.4"}, false, "198.51.100.9"},
		{"10.0.0.2:1", []string{"203.0.113.7"}, false, "203.0.113.7"},
		{"10.0.0.2:1", []string{"1.2.3.4, 203.0.113.7"}, false, "203.0.113.7"},
		{"10.0.0.2:1", []string{"1.2.3.4", "203.0.113.7"}, false, "203.0.113.7"},
		{"10.0.0.2:1", []string{"203.0.113.7, 10.0.0.5"}, false, "203.0.113.7"},
		{"10.0.0.2:1", []string{"10.0.0.9, 10.0.0.5"}, false, "10.0.0.9"},
		{"10.0.0.2:1", nil, false, "10.0.0.2"},
		{"10.0.0.2:1", []string{" , "}, false, "10.0.0.2"},
		{"10.0.0.2:1", []string{"203.0.113.7, garbage"}, false, "10.0.0.2"},
		{"10.0.0.2:1", []string{"garbage, 203.0.113.7"}, false, "203.0.113.7"},
		{"10.0.0.2:1", []string{"unknown"}, false, "10.0.0.2"},
		{"[2001:db8:ffff::1]:443", []string{"2001:DB8::7"}, false, "2001:db8::7"},
		{"10.0.0.2:1", []string{"[2001:db8::7]:1234"}, false, "2001:db8::7"},
		{"10.0.0.2:1", []string{"[2001:db8::7]"}, false, "2001:db8::7"},
		{"10.0.0.2:1", []string{"203.0.113.7:8080"}, false, "203.0.113.7"},
		{"[::ffff:10.0.0.2]:1", []string{"203.0.113.7"}, false, "203.0.113.7"},
		{"10.0.0.2:1", []string{"::ffff:203.0.113.7"}, false, "203.0.113.7"},
		{"[fe80::1%eth0]:1", []string{"203.0.113.7"}, false, "203.0.113.7"},
		{"[2001:db8::7]:1", nil, true, "2001:db8::7"},
		{"pipe", []string{"203.0.113.7"}, false, "pipe"},
		{"", nil, false, ""},
	} {
		t2 := tr
		if c.none {
			t2 = nil
		}
		if got := clientIP(c.remote, c.xff, t2); got != c.want {
			t.Errorf("row %d: clientIP(%q, %q) = %q, want %q", i+1, c.remote, c.xff, got, c.want)
		}
	}
}

func TestClientIPWithoutMiddleware(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-For", "1.2.3.4")
	if got := ClientIP(r); got != "192.0.2.1" {
		t.Fatalf("got %q", got)
	}
}
