// Package store owns the SQLite connection and data access.
//
// The rest of the application depends on this package through the *sql.DB it
// exposes; domain services must not open their own connections.
package store

import (
	"database/sql"
	"fmt"
	"net/url"

	_ "modernc.org/sqlite" // pure-Go SQLite driver (no cgo)
)

// Open opens (creating if needed) the SQLite database at path and applies the
// pragmas required by the architecture: foreign keys, WAL journal, a busy
// timeout, and NORMAL synchronous mode.
func Open(path string) (*sql.DB, error) {
	dsn := "file:" + path + "?" + url.Values{
		"_pragma": {
			"foreign_keys(1)",
			"journal_mode(WAL)",
			"busy_timeout(5000)",
			"synchronous(NORMAL)",
		},
	}.Encode()

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", path, err)
	}

	// SQLite allows a single writer. Serialising connections keeps writes
	// simple and correct for this single-instance deployment.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	if err := verifyPragmas(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// OpenForInspection opens the database for read-only inspection.
//
// It is used by the `--db-info` CLI mode, which the deployment scripts call to
// determine the schema version and row counts even when the application cannot
// start (for example after a failed migration) and in Compose modes that
// publish no host port. Only SELECT statements are issued.
//
// The connection is opened read-write because SQLite needs to read the WAL;
// the caller must not write. The file must already exist.
func OpenForInspection(path string) (*sql.DB, error) {
	dsn := "file:" + path + "?" + url.Values{
		"_pragma": {"busy_timeout(5000)"},
	}.Encode()

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", path, err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return db, nil
}

func verifyPragmas(db *sql.DB) error {
	var fk int
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&fk); err != nil {
		return fmt.Errorf("read foreign_keys pragma: %w", err)
	}
	if fk != 1 {
		return fmt.Errorf("foreign_keys pragma is not enabled")
	}
	return nil
}

// JournalMode reports the current journal mode (informational; WAL is expected
// on regular filesystems but is not treated as fatal).
func JournalMode(db *sql.DB) (string, error) {
	var mode string
	if err := db.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil {
		return "", fmt.Errorf("read journal_mode pragma: %w", err)
	}
	return mode, nil
}
