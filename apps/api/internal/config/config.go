// Package config loads UNICOURSE_* environment variables.
package config

import (
	"fmt"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
)

type Config struct {
	Addr, BaseURL, DatabaseURL, SessionSecret, FilesDir                          string
	GoogleClientID, GoogleClientSecret, MicrosoftClientID, MicrosoftClientSecret string
	SMTPURL, WebOrigin, WebDir                                                   string
	// TrustedProxies are the reverse proxies whose X-Forwarded-For is believed; empty trusts none.
	TrustedProxies []netip.Prefix
	OutboxSenders  int   // mail sender goroutines per process, 1 to 10
	MaxUploadBytes int64 // largest material file, 1 to 10 GiB
	SignInIPLimit  int   // sign-in link requests per client address per hour, 1 to 1000000
}

// Load reads configuration through getenv; the error lists every missing required variable.
func Load(getenv func(string) string) (Config, error) {
	var missing []string
	req := func(name string) string {
		v := getenv("UNICOURSE_" + name)
		if v == "" {
			missing = append(missing, "UNICOURSE_"+name)
		}
		return v
	}
	opt := func(name string) string { return getenv("UNICOURSE_" + name) }
	c := Config{
		DatabaseURL:           req("DATABASE_URL"),
		BaseURL:               req("BASE_URL"),
		SessionSecret:         req("SESSION_SECRET"),
		FilesDir:              req("FILES_DIR"),
		Addr:                  opt("ADDR"),
		GoogleClientID:        opt("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:    opt("GOOGLE_CLIENT_SECRET"),
		MicrosoftClientID:     opt("MICROSOFT_CLIENT_ID"),
		MicrosoftClientSecret: opt("MICROSOFT_CLIENT_SECRET"),
		SMTPURL:               opt("SMTP_URL"),
		WebOrigin:             opt("WEB_ORIGIN"),
		WebDir:                opt("WEB_DIR"),
	}
	if c.Addr == "" {
		c.Addr = ":8080"
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing env: %s", strings.Join(missing, ", "))
	}
	if len(c.SessionSecret) < 32 {
		return Config{}, fmt.Errorf("UNICOURSE_SESSION_SECRET must be at least 32 bytes, got %d", len(c.SessionSecret))
	}
	var err error
	var senders int64
	if senders, err = wholeInRange(opt("OUTBOX_SENDERS"), 1, 1, 10, "UNICOURSE_OUTBOX_SENDERS must be a whole number from 1 to 10"); err != nil {
		return Config{}, err
	}
	c.OutboxSenders = int(senders)
	if c.MaxUploadBytes, err = wholeInRange(opt("MAX_UPLOAD_BYTES"), 100<<20, 1, 10<<30, "UNICOURSE_MAX_UPLOAD_BYTES must be a whole number of bytes from 1 to 10737418240"); err != nil {
		return Config{}, err
	}
	var ipLimit int64
	if ipLimit, err = wholeInRange(opt("SIGNIN_IP_LIMIT"), 300, 1, 1000000, "UNICOURSE_SIGNIN_IP_LIMIT must be a whole number from 1 to 1000000"); err != nil {
		return Config{}, err
	}
	c.SignInIPLimit = int(ipLimit)
	if c.WebOrigin, err = webOrigin(c.WebOrigin); err != nil {
		return Config{}, err
	}
	if c.TrustedProxies, err = parseProxies(opt("TRUSTED_PROXIES")); err != nil {
		return Config{}, err
	}
	return c, nil
}

// wholeInRange parses s as a base-10 integer in [lo, hi]; empty gives def.
func wholeInRange(s string, def, lo, hi int64, msg string) (int64, error) {
	if s == "" {
		return def, nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n < lo || n > hi {
		return 0, fmt.Errorf("%s, got %q", msg, s)
	}
	return n, nil
}

// webOrigin accepts scheme://host[:port] with one optional trailing slash and returns it lowercased without the slash.
func webOrigin(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil ||
		u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" || (u.Path != "" && u.Path != "/") {
		return "", fmt.Errorf("UNICOURSE_WEB_ORIGIN must be scheme://host[:port] with no path, e.g. http://localhost:5173, got %q", s)
	}
	if port := u.Port(); strings.HasSuffix(u.Host, ":") || (port != "" && !validPort(port)) {
		return "", fmt.Errorf("UNICOURSE_WEB_ORIGIN must be scheme://host[:port] with no path, e.g. http://localhost:5173, got %q", s)
	}
	return u.Scheme + "://" + strings.ToLower(u.Host), nil
}

func validPort(p string) bool {
	n, err := strconv.Atoi(p)
	return err == nil && n >= 1 && n <= 65535
}

func parseProxies(s string) ([]netip.Prefix, error) {
	var out []netip.Prefix
	for _, item := range strings.Split(s, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		var p netip.Prefix
		var err error
		if strings.Contains(item, "/") {
			if p, err = netip.ParsePrefix(item); err == nil {
				p = p.Masked()
			}
		} else {
			var a netip.Addr
			if a, err = netip.ParseAddr(item); err == nil {
				if a.Zone() != "" {
					err = fmt.Errorf("zone")
				} else {
					a = a.Unmap()
					p = netip.PrefixFrom(a, a.BitLen())
				}
			}
		}
		if err != nil {
			return nil, fmt.Errorf("UNICOURSE_TRUSTED_PROXIES: %q is not a CIDR or IP address", item)
		}
		if p.Addr().Is4In6() && p.Bits() >= 96 {
			p = netip.PrefixFrom(p.Addr().Unmap(), p.Bits()-96)
		}
		if p.Bits() == 0 {
			return nil, fmt.Errorf("UNICOURSE_TRUSTED_PROXIES: %s trusts every address; list only the proxies' addresses", item)
		}
		out = append(out, p)
	}
	return out, nil
}
