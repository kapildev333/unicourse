package httpx

import (
	"context"
	"net/http"
	"net/netip"
	"strings"
)

// parseIP accepts 1.2.3.4, 1.2.3.4:80, ::1, [::1], [::1]:80, fe80::1%eth0. It unmaps
// IPv4-mapped IPv6 and drops the zone, so one host has one canonical key and
// netip.Prefix.Contains works (Contains is false for any address with a zone).
func parseIP(s string) (netip.Addr, bool) {
	s = strings.TrimSpace(s)
	if ap, err := netip.ParseAddrPort(s); err == nil {
		return ap.Addr().Unmap().WithZone(""), true
	}
	if len(s) > 2 && s[0] == '[' && s[len(s)-1] == ']' {
		s = s[1 : len(s)-1]
	}
	a, err := netip.ParseAddr(s)
	if err != nil {
		return netip.Addr{}, false
	}
	return a.Unmap().WithZone(""), true
}

func trustedAddr(a netip.Addr, trusted []netip.Prefix) bool {
	for _, p := range trusted {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

// clientIP: the connection address unless it is a trusted proxy; then the rightmost
// X-Forwarded-For hop that is not itself trusted (a client can only prepend entries,
// so it cannot choose that hop). All hops trusted: the leftmost hop. Malformed hop
// reached during the walk: the connection address. Never fails.
func clientIP(remoteAddr string, xff []string, trusted []netip.Prefix) string {
	remote, ok := parseIP(remoteAddr)
	if !ok {
		return remoteAddr // unix socket or test oddity: same as today
	}
	if !trustedAddr(remote, trusted) {
		return remote.String()
	}
	hops := strings.Split(strings.Join(xff, ","), ",")
	client := remote
	for i := len(hops) - 1; i >= 0; i-- {
		if strings.TrimSpace(hops[i]) == "" {
			continue
		}
		a, ok := parseIP(hops[i])
		if !ok {
			return remote.String()
		}
		client = a
		if !trustedAddr(a, trusted) {
			break
		}
	}
	return client.String()
}

// ClientIPMiddleware resolves the client address once per request (see clientIP) and
// stores it for ClientIP. r.RemoteAddr is left as the socket peer.
func ClientIPMiddleware(trusted []netip.Prefix, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r.RemoteAddr, r.Header.Values("X-Forwarded-For"), trusted)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), clientIPKey, ip)))
	})
}

// ClientIP returns the resolved client address; without the middleware it is the
// connection address. It never fails.
func ClientIP(r *http.Request) string {
	if ip, ok := r.Context().Value(clientIPKey).(string); ok {
		return ip
	}
	return clientIP(r.RemoteAddr, nil, nil)
}
