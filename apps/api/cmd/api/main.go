package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/app"
	"github.com/kapildev333/unicourse/apps/api/internal/auth"
	"github.com/kapildev333/unicourse/apps/api/internal/config"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
)

// newHandler builds the routed, middleware-wrapped handler; ready backs /readyz and api (may be nil)
// serves everything under /api/v1/. web (may be nil) serves the built web app at /; nil answers the 404 envelope.
func newHandler(logger *slog.Logger, ready func(context.Context) error, api, web http.Handler) http.Handler {
	mux := http.NewServeMux()
	health := httpx.Health(ready)
	mux.Handle("GET /healthz", health)
	mux.Handle("GET /readyz", health)
	if api != nil {
		mux.Handle("/api/v1/", api)
	}
	if web == nil {
		web = httpx.H(func(w http.ResponseWriter, r *http.Request) error { return httpx.NotFound("route") })
	}
	mux.Handle("/", web)
	return httpx.RequestIDMiddleware(httpx.LogMiddleware(logger, httpx.Recover(mux)))
}

// authDeps builds the auth wiring: providers whose client id is unset are absent; without an SMTP URL
// the magic-link mail is written to the log (dev and e2e).
func authDeps(ctx context.Context, cfg config.Config, db *sql.DB, logger *slog.Logger) (auth.Deps, error) {
	providers, err := auth.NewProviders(ctx, cfg)
	if err != nil {
		return auth.Deps{}, err
	}
	var mailer auth.Mailer = auth.LogMailer{L: logger}
	if cfg.SMTPURL != "" {
		if mailer, err = auth.NewSMTPMailer(cfg.SMTPURL); err != nil {
			return auth.Deps{}, err
		}
	}
	web := cfg.WebOrigin
	if web == "" {
		web = cfg.BaseURL
	}
	return auth.Deps{
		DB: db, Sessions: &auth.SessionStore{DB: db},
		Magic:     &auth.MagicLinks{DB: db, Mailer: mailer, BaseURL: cfg.BaseURL},
		Providers: providers, Secret: []byte(cfg.SessionSecret), WebBase: web,
		EmailLimit: auth.NewLimiter(5, time.Hour), IPLimit: auth.NewLimiter(300, time.Hour),
	}, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load(os.Getenv)
	if err != nil {
		logger.Error("config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := pg.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database", "err", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := pg.Migrate(ctx, db); err != nil {
		logger.Error("migrate", "err", err)
		os.Exit(1)
	}

	deps, err := authDeps(ctx, cfg, db, logger)
	if err != nil {
		logger.Error("auth", "err", err)
		os.Exit(1)
	}
	api := app.NewRouter(app.Deps{Auth: deps, Mailer: deps.Magic.Mailer, BaseURL: cfg.BaseURL})

	var web http.Handler
	if cfg.WebDir != "" {
		if web, err = httpx.Static(cfg.WebDir); err != nil {
			logger.Error("web", "err", err)
			os.Exit(1)
		}
	}

	srv := &http.Server{Addr: cfg.Addr, Handler: newHandler(logger, db.PingContext, api, web), ReadHeaderTimeout: 5 * time.Second}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	logger.Info("listening", "addr", cfg.Addr)

	select {
	case err := <-errc:
		logger.Error("server", "err", err)
		os.Exit(1)
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown", "err", err)
		os.Exit(1)
	}
}
