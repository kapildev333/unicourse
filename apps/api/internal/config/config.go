// Package config loads UNICOURSE_* environment variables.
package config

import (
	"fmt"
	"strings"
)

type Config struct {
	Addr, BaseURL, DatabaseURL, SessionSecret                                    string
	GoogleClientID, GoogleClientSecret, MicrosoftClientID, MicrosoftClientSecret string
	SMTPURL, WebOrigin                                                           string
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
		Addr:                  opt("ADDR"),
		GoogleClientID:        opt("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:    opt("GOOGLE_CLIENT_SECRET"),
		MicrosoftClientID:     opt("MICROSOFT_CLIENT_ID"),
		MicrosoftClientSecret: opt("MICROSOFT_CLIENT_SECRET"),
		SMTPURL:               opt("SMTP_URL"),
		WebOrigin:             opt("WEB_ORIGIN"),
	}
	if c.Addr == "" {
		c.Addr = ":8080"
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing env: %s", strings.Join(missing, ", "))
	}
	return c, nil
}
