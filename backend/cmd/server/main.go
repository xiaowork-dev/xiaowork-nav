// Command server is the XiaoWork Nav backend entrypoint.
//
// It loads configuration from the environment, opens SQLite, applies pending
// migrations (aborting on failure), and serves the REST API plus the embedded
// frontend with graceful shutdown.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
	// Deployment helper mode: report database state and exit. The update and
	// restore scripts use this because it works in every Compose mode (including
	// ones that publish no host port) and while the application is stopped, so a
	// failed update can still be assessed before any rollback decision.
	if hasFlag(os.Args[1:], "--db-info") {
		return runDBInfo(os.Args[1:])
	}

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

	// Refuse to run against a database created by a newer build: opening it
	// with older code could corrupt data. This is what makes "roll back to the
	// previous image" safe to detect (see docs/updating.md).
	if current, latest, err := migrations.CheckCompatible(db); err != nil {
		if errors.Is(err, migrations.ErrSchemaTooNew) {
			logger.Error("database schema is newer than this build; refusing to start",
				"database_schema", current, "build_schema", latest,
				"hint", "restore the pre-update backup or deploy a matching image version")
			return 1
		}
		logger.Error("check schema compatibility", "error", err)
		return 1
	}

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

	app := api.New(api.Deps{
		Config: cfg,
		DB:     db,
		Logger: logger,
		Static: static.Handler(),
	})
	handler := app.Handler()

	if !cfg.SecureCookie {
		logger.Warn("session cookie is not marked Secure (XN_SECURE_COOKIE=false); enable it in production")
	}

	maintenanceCtx, stopMaintenance := context.WithCancel(context.Background())
	defer stopMaintenance()
	go app.RunMaintenance(maintenanceCtx, time.Hour)

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

// runDBInfo implements the `--db-info` deployment helper mode.
func runDBInfo(args []string) int {
	dbPath := flagValue(args, "--db")
	if dbPath == "" {
		if cfg, err := config.Load(); err == nil {
			dbPath = cfg.DBPath
		}
	}
	if dbPath == "" {
		dbPath = "/data/app.db"
	}
	fmt.Printf("db_path=%s\n", dbPath)

	if _, err := os.Stat(dbPath); err != nil {
		fmt.Println("exists=false")
		fmt.Println("schema_version=0")
		return 0
	}
	fmt.Println("exists=true")

	db, err := store.OpenForInspection(dbPath)
	if err != nil {
		fmt.Printf("error=%s\n", err)
		fmt.Println("schema_version=0")
		return 0
	}
	defer func() { _ = db.Close() }()

	version, err := migrations.ReadVersion(db)
	if err != nil {
		fmt.Printf("error=%s\n", err)
		fmt.Println("schema_version=0")
		return 0
	}
	fmt.Printf("schema_version=%d\n", version)
	fmt.Printf("categories=%d\n", countRows(db, "categories"))
	fmt.Printf("bookmarks=%d\n", countRows(db, "bookmarks"))
	fmt.Printf("uploads=%d\n", countRows(db, "uploads"))
	return 0
}

// countRows returns the row count, or -1 when the table is missing or unreadable.
func countRows(db *sql.DB, table string) int {
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
		return -1
	}
	return n
}

func hasFlag(args []string, name string) bool {
	for _, a := range args {
		if a == name {
			return true
		}
	}
	return false
}

// flagValue reads "--name value" or "--name=value".
func flagValue(args []string, name string) string {
	for i, a := range args {
		if a == name && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(a, name+"=") {
			return strings.TrimPrefix(a, name+"=")
		}
	}
	return ""
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
