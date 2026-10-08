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

// forwardedFor returns the for= value of every element of the Forwarded header lines (RFC 7239), in order.
// A blank element yields "" (skipped, like an empty X-Forwarded-For entry). An element with no for=, a
// for= that is not a valid token or quoted string, or a quoted string with a backslash yields "unknown"
// (not an address, so the walk stops there with the connection address).
// ponytail: naive comma and semicolon splitting; a quoted value containing either splits wrongly, which can
// only make the sender's own request fall back to the proxy address. Add a real parser if that ever matters.
func forwardedFor(lines []string) []string {
	var out []string
	for _, el := range strings.Split(strings.Join(lines, ","), ",") {
		if strings.TrimSpace(el) == "" {
			out = append(out, "")
			continue
		}
		v := "unknown"
		for _, pair := range strings.Split(el, ";") {
			name, val, ok := strings.Cut(pair, "=")
			if !ok || !strings.EqualFold(strings.TrimSpace(name), "for") {
				continue
			}
			val = strings.TrimSpace(val)
			if strings.HasPrefix(val, `"`) {
				if len(val) < 2 || !strings.HasSuffix(val, `"`) || strings.ContainsAny(val[1:len(val)-1], `"\`) {
					break
				}
				val = val[1 : len(val)-1]
			}
			v = val
			break
		}
		out = append(out, v)
	}
	return out
}

// walk returns the rightmost hop that is not trusted, the leftmost hop when all are trusted, remote when
// there are no hops, and remote when a malformed hop is reached first. Blank hops are skipped.
func walk(remote netip.Addr, hops []string, trusted []netip.Prefix) netip.Addr {
	client := remote
	for i := len(hops) - 1; i >= 0; i-- {
		if strings.TrimSpace(hops[i]) == "" {
			continue
		}
		a, ok := parseIP(hops[i])
		if !ok {
			return remote
		}
		client = a
		if !trustedAddr(a, trusted) {
			break
		}
	}
	return client
}

func present(lines []string) bool {
	return strings.Trim(strings.Join(lines, ""), " ,\t") != ""
}

// ponytail: only X-Forwarded-For and Forwarded (not CF-Connecting-IP, X-Real-IP); IPv6 keyed per address, not
// per /64; clients inside a trusted range can pick their own key. Add headers or /64 masking when a deployment needs it.
// clientIP: the connection address unless it is a trusted proxy; then the rightmost hop of
// X-Forwarded-For or Forwarded (for=) that is not itself trusted (a client can only prepend entries,
// so it cannot choose that hop). All hops trusted: the leftmost hop. Malformed hop reached during the
// walk: the connection address. Both headers present: their walks must agree, else the connection
// address (a proxy chain sets one; the other is the client's). Never fails.
func clientIP(remoteAddr string, xff, fwd []string, trusted []netip.Prefix) string {
	remote, ok := parseIP(remoteAddr)
	if !ok {
		return remoteAddr // unix socket or test oddity: same as today
	}
	if !trustedAddr(remote, trusted) {
		return remote.String()
	}
	byXFF := walk(remote, strings.Split(strings.Join(xff, ","), ","), trusted)
	if !present(fwd) {
		return byXFF.String()
	}
	byFwd := walk(remote, forwardedFor(fwd), trusted)
	if !present(xff) || byXFF == byFwd {
		return byFwd.String()
	}
	return remote.String()
}

// ClientIPMiddleware resolves the client address once per request (see clientIP) and
// stores it for ClientIP. r.RemoteAddr is left as the socket peer.
func ClientIPMiddleware(trusted []netip.Prefix, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r.RemoteAddr, r.Header.Values("X-Forwarded-For"), r.Header.Values("Forwarded"), trusted)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), clientIPKey, ip)))
	})
}

// ClientIP returns the resolved client address; without the middleware it is the
// connection address. It never fails.
func ClientIP(r *http.Request) string {
	if ip, ok := r.Context().Value(clientIPKey).(string); ok {
		return ip
	}
	return clientIP(r.RemoteAddr, nil, nil, nil)
}
