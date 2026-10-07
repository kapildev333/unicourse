// Package config loads UNICOURSE_* environment variables.
package config

import (
	"fmt"
	"net/netip"
	"strings"
)

type Config struct {
	Addr, BaseURL, DatabaseURL, SessionSecret, FilesDir                          string
	GoogleClientID, GoogleClientSecret, MicrosoftClientID, MicrosoftClientSecret string
	SMTPURL, WebOrigin, WebDir                                                   string
	// TrustedProxies are the reverse proxies whose X-Forwarded-For is believed; empty trusts none.
	TrustedProxies []netip.Prefix
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
	if c.TrustedProxies, err = parseProxies(opt("TRUSTED_PROXIES")); err != nil {
		return Config{}, err
	}
	return c, nil
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
		if p.Bits() == 0 {
			return nil, fmt.Errorf("UNICOURSE_TRUSTED_PROXIES: %s trusts every address; list only the proxies' addresses", item)
		}
		out = append(out, p)
	}
	return out, nil
}
