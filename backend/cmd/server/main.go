// Command server is the XiaoWork Nav backend entrypoint.
//
// It loads configuration from the environment, opens SQLite, applies pending
// migrations (aborting on failure), and serves the REST API plus the embedded
// frontend with graceful shutdown.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/api"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/config"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/migrations"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/static"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/store"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/version"
)

func main() {
	os.Exit(run())
}

func run() int {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		return 1
	}

	logger := newLogger(cfg.LogLevel)

	if err := cfg.EnsureDirs(); err != nil {
		logger.Error("prepare data directories", "error", err)
		return 1
	}

	db, err := store.Open(cfg.DBPath)
	if err != nil {
		logger.Error("open database", "error", err, "path", cfg.DBPath)
		return 1
	}
	defer func() { _ = db.Close() }()

	applied, err := migrations.Apply(db)
	if err != nil {
		// A failed migration must stop startup: never run half-migrated.
		logger.Error("database migration failed", "error", err)
		return 1
	}
	if len(applied) > 0 {
		logger.Info("applied migrations", "versions", migrations.FormatVersions(applied))
	}

	current, latest, err := migrations.Status(db)
	if err != nil {
		logger.Error("read schema status", "error", err)
		return 1
	}
	if current < latest {
		logger.Error("schema is behind", "current", current, "latest", latest)
		return 1
	}

	logger.Info("starting xiaowork-nav",
		"version", version.Version,
		"commit", version.Commit,
		"addr", cfg.Addr(),
		"data_dir", cfg.DataDir,
		"schema_version", current,
	)

	handler := api.New(api.Deps{
		Config: cfg,
		DB:     db,
		Logger: logger,
		Static: static.Handler(),
	}).Handler()

	srv := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		logger.Error("http server error", "error", err)
		return 1
	case sig := <-stop:
		logger.Info("shutting down", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		return 1
	}
	logger.Info("stopped")
	return 0
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}
