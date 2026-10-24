package store

import (
	"database/sql"
	"errors"
	"strings"
)

// ErrNotFound is returned when a requested row does not exist.
var ErrNotFound = errors.New("store: not found")

// Store provides typed data access over a SQLite connection.
type Store struct {
	db *sql.DB
}

// New wraps an open database connection.
func New(db *sql.DB) *Store {
	return &Store{db: db}
}

// DB exposes the underlying connection (used by migrations and health checks).
func (s *Store) DB() *sql.DB {
	return s.db
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func intToBool(i int) bool {
	return i != 0
}

// isUniqueViolation reports whether err is a SQLite UNIQUE constraint failure.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "constraint failed: UNIQUE")
}
