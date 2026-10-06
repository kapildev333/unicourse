package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kapildev333/unicourse/apps/api/internal/config"
	"github.com/kapildev333/unicourse/apps/api/internal/httpx"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load(os.Getenv)
	if err != nil {
		logger.Error("config", "err", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	// db wired in P0-L2
	mux.Handle("/", httpx.Health(func(context.Context) error { return errors.New("database not configured") }))

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpx.Recover(httpx.RequestIDMiddleware(httpx.LogMiddleware(logger, mux))),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown", "err", err)
		}
	}()

	logger.Info("listening", "addr", cfg.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server", "err", err)
		os.Exit(1)
	}
}
