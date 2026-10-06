package config

import (
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
