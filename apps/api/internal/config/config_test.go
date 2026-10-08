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
		"UNICOURSE_FILES_DIR":      "/tmp/files",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if c.Addr != ":8080" || c.DatabaseURL != "postgres://x" || c.BaseURL != "http://localhost:8080" || c.SessionSecret != strings.Repeat("s", 32) || c.SMTPURL != "" || c.FilesDir != "/tmp/files" {
		t.Fatalf("unexpected config %+v", c)
	}
}

func TestLoadMissing(t *testing.T) {
	_, err := Load(env(map[string]string{"UNICOURSE_BASE_URL": "b"}))
	if err == nil || !strings.Contains(err.Error(), "UNICOURSE_DATABASE_URL") || !strings.Contains(err.Error(), "UNICOURSE_SESSION_SECRET") || !strings.Contains(err.Error(), "UNICOURSE_FILES_DIR") || strings.Contains(err.Error(), "UNICOURSE_BASE_URL") {
		t.Fatalf("got %v", err)
	}
}

func TestLoadShortSecret(t *testing.T) {
	_, err := Load(env(map[string]string{
		"UNICOURSE_DATABASE_URL":   "postgres://x",
		"UNICOURSE_BASE_URL":       "http://localhost:8080",
		"UNICOURSE_SESSION_SECRET": strings.Repeat("s", 31),
		"UNICOURSE_FILES_DIR":      "/tmp/files",
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
		"UNICOURSE_FILES_DIR":      "/tmp/files",
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
			"UNICOURSE_FILES_DIR":      "/tmp/files",
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
		"::1":                 {"::1/128"},
		"10.0.0.1/8":          {"10.0.0.0/8"},
		"::ffff:10.0.0.0/104": {"10.0.0.0/8"},
		"::ffff:10.1.2.3/120": {"10.1.2.0/24"},
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
	for _, bad := range []string{"nope", "10.0.0.0/33", "fe80::1%eth0", "0.0.0.0/0", "::/0", "::ffff:0.0.0.0/96"} {
		_, err := load("10.0.0.1," + bad)
		if err == nil || !strings.Contains(err.Error(), "UNICOURSE_TRUSTED_PROXIES") || !strings.Contains(err.Error(), bad) {
			t.Errorf("%q: %v", bad, err)
		}
	}
}

func loadWith(extra map[string]string) (Config, error) {
	m := map[string]string{
		"UNICOURSE_DATABASE_URL":   "postgres://x",
		"UNICOURSE_BASE_URL":       "http://localhost:8080",
		"UNICOURSE_SESSION_SECRET": strings.Repeat("s", 32),
		"UNICOURSE_FILES_DIR":      "/tmp/files",
	}
	for k, v := range extra {
		m["UNICOURSE_"+k] = v
	}
	return Load(env(m))
}

func TestLoadOutboxSenders(t *testing.T) {
	for in, want := range map[string]int{"": 1, "1": 1, "4": 4, "10": 10} {
		c, err := loadWith(map[string]string{"OUTBOX_SENDERS": in})
		if err != nil || c.OutboxSenders != want {
			t.Errorf("%q: %d %v", in, c.OutboxSenders, err)
		}
	}
	for _, bad := range []string{"0", "11", "x", "-1", "2.5"} {
		_, err := loadWith(map[string]string{"OUTBOX_SENDERS": bad})
		if want := `UNICOURSE_OUTBOX_SENDERS must be a whole number from 1 to 10, got "` + bad + `"`; err == nil || err.Error() != want {
			t.Errorf("%q: %v", bad, err)
		}
	}
}

func TestLoadMaxUploadBytes(t *testing.T) {
	for in, want := range map[string]int64{"": 104857600, "1": 1, "10737418240": 10737418240} {
		c, err := loadWith(map[string]string{"MAX_UPLOAD_BYTES": in})
		if err != nil || c.MaxUploadBytes != want {
			t.Errorf("%q: %d %v", in, c.MaxUploadBytes, err)
		}
	}
	for _, bad := range []string{"0", "10737418241", "x", "-1", "1.5", "100MB"} {
		_, err := loadWith(map[string]string{"MAX_UPLOAD_BYTES": bad})
		if want := `UNICOURSE_MAX_UPLOAD_BYTES must be a whole number of bytes from 1 to 10737418240, got "` + bad + `"`; err == nil || err.Error() != want {
			t.Errorf("%q: %v", bad, err)
		}
	}
}

func TestLoadWebOrigin(t *testing.T) {
	for in, want := range map[string]string{
		"": "", "http://localhost:5173": "http://localhost:5173", "https://App.Example.edu": "https://app.example.edu",
		"http://localhost:5173/": "http://localhost:5173",
	} {
		c, err := loadWith(map[string]string{"WEB_ORIGIN": in})
		if err != nil || c.WebOrigin != want {
			t.Errorf("%q: %q %v", in, c.WebOrigin, err)
		}
	}
	for _, bad := range []string{"localhost:5173", "http://localhost:5173/app", "ftp://x", "http://", "http://u:p@x", "http://x?y=1", "http://x#f", " http://x", "http://x//", "http://x:", "http://x:0", "http://x:99999", "http://x:+80"} {
		_, err := loadWith(map[string]string{"WEB_ORIGIN": bad})
		if want := `UNICOURSE_WEB_ORIGIN must be scheme://host[:port] with no path, e.g. http://localhost:5173, got "` + bad + `"`; err == nil || err.Error() != want {
			t.Errorf("%q: %v", bad, err)
		}
	}
}

func TestLoadSignInIPLimit(t *testing.T) {
	for in, want := range map[string]int{"": 300, "1": 1, "100000": 100000, "1000000": 1000000} {
		c, err := loadWith(map[string]string{"SIGNIN_IP_LIMIT": in})
		if err != nil || c.SignInIPLimit != want {
			t.Errorf("%q: %d %v", in, c.SignInIPLimit, err)
		}
	}
	for _, bad := range []string{"0", "1000001", "x", "-1", "2.5"} {
		_, err := loadWith(map[string]string{"SIGNIN_IP_LIMIT": bad})
		if want := `UNICOURSE_SIGNIN_IP_LIMIT must be a whole number from 1 to 1000000, got "` + bad + `"`; err == nil || err.Error() != want {
			t.Errorf("%q: %v", bad, err)
		}
	}
}
