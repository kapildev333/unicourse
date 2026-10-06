package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/config"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
	"github.com/kapildev333/unicourse/apps/api/internal/pg"
)

// newHandler builds the routed, middleware-wrapped handler; ready backs /readyz.
func newHandler(logger *slog.Logger, ready func(context.Context) error) http.Handler {
	mux := http.NewServeMux()
	health := httpx.Health(ready)
	mux.Handle("GET /healthz", health)
	mux.Handle("GET /readyz", health)
	mux.Handle("/", httpx.H(func(w http.ResponseWriter, r *http.Request) error { return httpx.NotFound("route") }))
	return httpx.RequestIDMiddleware(httpx.LogMiddleware(logger, httpx.Recover(mux)))
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

	srv := &http.Server{Addr: cfg.Addr, Handler: newHandler(logger, db.PingContext), ReadHeaderTimeout: 5 * time.Second}
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
