package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"net/netip"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/app"
	"github.com/kapildev333/unicourse/apps/api/internal/auth"
	"github.com/kapildev333/unicourse/apps/api/internal/blob"
	"github.com/kapildev333/unicourse/apps/api/internal/config"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/outbox"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
)

// newHandler builds the routed, middleware-wrapped handler; ready backs /readyz and api (may be nil)
// serves everything under /api/v1/. web (may be nil) serves the built web app at /; nil answers the 404 envelope.
func newHandler(logger *slog.Logger, trusted []netip.Prefix, ready func(context.Context) error, api, web http.Handler) http.Handler {
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
	return httpx.ClientIPMiddleware(trusted, httpx.RequestIDMiddleware(httpx.LogMiddleware(logger, httpx.Recover(mux))))
}

// authDeps builds the auth wiring: providers whose client id is unset are absent. wake nudges the
// outbox sender after a magic link is queued.
func authDeps(ctx context.Context, cfg config.Config, db *sql.DB, wake func()) (auth.Deps, error) {
	providers, err := auth.NewProviders(ctx, cfg)
	if err != nil {
		return auth.Deps{}, err
	}
	web := cfg.WebOrigin
	if web == "" {
		web = cfg.BaseURL
	}
	return auth.Deps{
		DB: db, Sessions: &auth.SessionStore{DB: db},
		Magic:     &auth.MagicLinks{DB: db, Wake: wake, BaseURL: cfg.BaseURL},
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

	files, err := blob.New(cfg.FilesDir)
	if err != nil {
		logger.Error("files", "err", err)
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
	if err := pg.CheckAPIRole(ctx, db); err != nil {
		logger.Error("database role", "err", err)
		os.Exit(1)
	}
	if err := pg.CheckMigrated(ctx, db); err != nil {
		logger.Error("database not migrated", "err", err)
		os.Exit(1)
	}

	// without an SMTP URL mail is written to the log (dev and e2e)
	var mailer outbox.Mailer = outbox.LogMailer{L: logger}
	if cfg.SMTPURL != "" {
		if mailer, err = outbox.NewSMTPMailer(cfg.SMTPURL); err != nil {
			logger.Error("mail", "err", err)
			os.Exit(1)
		}
	}
	sender := outbox.NewSender(db, mailer, logger)
	go sender.Run(ctx)

	deps, err := authDeps(ctx, cfg, db, sender.Wake)
	if err != nil {
		logger.Error("auth", "err", err)
		os.Exit(1)
	}
	api := app.NewRouter(app.Deps{Auth: deps, BaseURL: cfg.BaseURL, Files: files})

	var web http.Handler
	if cfg.WebDir != "" {
		if web, err = httpx.Static(cfg.WebDir); err != nil {
			logger.Error("web", "err", err)
			os.Exit(1)
		}
	}

	srv := &http.Server{Addr: cfg.Addr, Handler: newHandler(logger, cfg.TrustedProxies, func(ctx context.Context) error { return pg.CheckMigrated(ctx, db) }, api, web), ReadHeaderTimeout: 5 * time.Second}
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
