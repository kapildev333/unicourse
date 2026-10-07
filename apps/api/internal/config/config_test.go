package config

import (
	"net/netip"
	"strings"
	"testing"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoad(t *testing.T) {
	c, err := Load(env(map[string]string{
		"UNICOURSE_DATABASE_URL":   "postgres://x",
		"UNICOURSE_BASE_URL":       "http://localhost:8080",
		"UNICOURSE_SESSION_SECRET": strings.Repeat("s", 32),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if c.Addr != ":8080" || c.DatabaseURL != "postgres://x" || c.BaseURL != "http://localhost:8080" || c.SessionSecret != strings.Repeat("s", 32) || c.SMTPURL != "" {
		t.Fatalf("unexpected config %+v", c)
	}
}

func TestLoadMissing(t *testing.T) {
	_, err := Load(env(map[string]string{"UNICOURSE_BASE_URL": "b"}))
	if err == nil || !strings.Contains(err.Error(), "UNICOURSE_DATABASE_URL") || !strings.Contains(err.Error(), "UNICOURSE_SESSION_SECRET") || strings.Contains(err.Error(), "UNICOURSE_BASE_URL") {
		t.Fatalf("got %v", err)
	}
}

func TestLoadShortSecret(t *testing.T) {
	_, err := Load(env(map[string]string{
		"UNICOURSE_DATABASE_URL":   "postgres://x",
		"UNICOURSE_BASE_URL":       "http://localhost:8080",
		"UNICOURSE_SESSION_SECRET": strings.Repeat("s", 31),
	}))
	if err == nil || !strings.Contains(err.Error(), "UNICOURSE_SESSION_SECRET") || !strings.Contains(err.Error(), "32") {
		t.Fatalf("got %v", err)
	}
}

func TestLoadWebDir(t *testing.T) {
	m := map[string]string{
		"UNICOURSE_DATABASE_URL":   "postgres://x",
		"UNICOURSE_BASE_URL":       "http://localhost:8080",
		"UNICOURSE_SESSION_SECRET": strings.Repeat("s", 32),
	}
	c, err := Load(env(m))
	if err != nil || c.WebDir != "" {
		t.Fatal(c.WebDir, err)
	}
	m["UNICOURSE_WEB_DIR"] = "../web/dist"
	if c, err = Load(env(m)); err != nil || c.WebDir != "../web/dist" {
		t.Fatal(c.WebDir, err)
	}
}

func TestLoadTrustedProxies(t *testing.T) {
	load := func(v string) (Config, error) {
		m := map[string]string{
			"UNICOURSE_DATABASE_URL":   "postgres://x",
			"UNICOURSE_BASE_URL":       "http://localhost:8080",
			"UNICOURSE_SESSION_SECRET": strings.Repeat("s", 32),
		}
		if v != "" {
			m["UNICOURSE_TRUSTED_PROXIES"] = v
		}
		return Load(env(m))
	}
	if c, err := load(""); err != nil || c.TrustedProxies != nil {
		t.Fatalf("unset: %v %v", c.TrustedProxies, err)
	}
	for in, want := range map[string][]string{
		"10.0.0.0/8, 192.168.1.1 ,2001:db8::/32,": {"10.0.0.0/8", "192.168.1.1/32", "2001:db8::/32"},
		"::1":        {"::1/128"},
		"10.0.0.1/8": {"10.0.0.0/8"},
	} {
		c, err := load(in)
		if err != nil || len(c.TrustedProxies) != len(want) {
			t.Fatalf("%q: %v %v", in, c.TrustedProxies, err)
		}
		for i, w := range want {
			if c.TrustedProxies[i] != netip.MustParsePrefix(w) {
				t.Errorf("%q[%d] = %v, want %s", in, i, c.TrustedProxies[i], w)
			}
		}
	}
	for _, bad := range []string{"nope", "10.0.0.0/33", "fe80::1%eth0", "0.0.0.0/0", "::/0"} {
		_, err := load("10.0.0.1," + bad)
		if err == nil || !strings.Contains(err.Error(), "UNICOURSE_TRUSTED_PROXIES") || !strings.Contains(err.Error(), bad) {
			t.Errorf("%q: %v", bad, err)
		}
	}
}
