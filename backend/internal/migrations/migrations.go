// Package migrations applies versioned SQL migrations embedded in the binary.
//
// Design (see docs/database-design.md and docs/decisions/ADR-0003):
//   - migrations are numbered `NNNN_<name>.up.sql`;
//   - applied versions are recorded in `schema_migrations`;
//   - each migration runs inside its own transaction;
//   - a failure aborts startup so the service never runs half-migrated.
package migrations

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed sql/*.sql
var sqlFS embed.FS

var upFileRe = regexp.MustCompile(`^(\d+)_(.+)\.up\.sql$`)

// Migration is a single up migration.
type Migration struct {
	Version int
	Name    string
	UpSQL   string
}

// Load reads and sorts all embedded up migrations.
func Load() ([]Migration, error) {
	entries, err := fs.ReadDir(sqlFS, "sql")
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}

	var out []Migration
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := upFileRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		version, err := strconv.Atoi(m[1])
		if err != nil {
			return nil, fmt.Errorf("parse migration version %q: %w", e.Name(), err)
		}
		content, err := sqlFS.ReadFile("sql/" + e.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration %q: %w", e.Name(), err)
		}
		out = append(out, Migration{Version: version, Name: m[2], UpSQL: string(content)})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })

	for i := 1; i < len(out); i++ {
		if out[i].Version == out[i-1].Version {
			return nil, fmt.Errorf("duplicate migration version %d", out[i].Version)
		}
	}
	return out, nil
}

// LatestVersion returns the highest available migration version.
func LatestVersion() (int, error) {
	ms, err := Load()
	if err != nil {
		return 0, err
	}
	if len(ms) == 0 {
		return 0, nil
	}
	return ms[len(ms)-1].Version, nil
}

// CurrentVersion returns the highest applied migration version (0 if none).
func CurrentVersion(db *sql.DB) (int, error) {
	if err := ensureTable(db); err != nil {
		return 0, err
	}
	var v sql.NullInt64
	if err := db.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&v); err != nil {
		return 0, fmt.Errorf("read schema version: %w", err)
	}
	if !v.Valid {
		return 0, nil
	}
	return int(v.Int64), nil
}

// ErrSchemaTooNew is returned when the database was migrated by a newer build
// than the one currently running. Starting would risk corrupting data, so the
// service must refuse to start (see docs/updating.md).
var ErrSchemaTooNew = errors.New("database schema is newer than this build")

// CheckCompatible verifies that the on-disk schema is not newer than the
// migrations bundled with this build, and returns the current/latest versions.
func CheckCompatible(db *sql.DB) (current, latest int, err error) {
	if current, err = CurrentVersion(db); err != nil {
		return 0, 0, err
	}
	if latest, err = LatestVersion(); err != nil {
		return 0, 0, err
	}
	if current > latest {
		return current, latest, fmt.Errorf("%w: database=%d, build=%d", ErrSchemaTooNew, current, latest)
	}
	return current, latest, nil
}

// ReadVersion returns the applied schema version without creating anything.
//
// Unlike CurrentVersion it never writes, so it is safe to call against a
// database the running application still owns, and it returns 0 when the
// schema_migrations table does not exist yet.
func ReadVersion(db *sql.DB) (int, error) {
	var exists int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'schema_migrations'`,
	).Scan(&exists); err != nil {
		return 0, fmt.Errorf("look up schema_migrations: %w", err)
	}
	if exists == 0 {
		return 0, nil
	}

	var v sql.NullInt64
	if err := db.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&v); err != nil {
		return 0, fmt.Errorf("read schema version: %w", err)
	}
	if !v.Valid {
		return 0, nil
	}
	return int(v.Int64), nil
}

// Apply runs all pending migrations and returns the versions it applied.
func Apply(db *sql.DB) ([]int, error) {
	if err := ensureTable(db); err != nil {
		return nil, err
	}

	ms, err := Load()
	if err != nil {
		return nil, err
	}

	applied, err := appliedVersions(db)
	if err != nil {
		return nil, err
	}

	var done []int
	for _, m := range ms {
		if applied[m.Version] {
			continue
		}
		if err := applyOne(db, m); err != nil {
			return done, err
		}
		done = append(done, m.Version)
	}
	return done, nil
}

func applyOne(db *sql.DB, m Migration) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin migration %d: %w", m.Version, err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(m.UpSQL); err != nil {
		return fmt.Errorf("migration %d (%s) failed: %w", m.Version, m.Name, err)
	}
	if _, err := tx.Exec(
		`INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)`,
		m.Version, m.Name, time.Now().UTC().Format(time.RFC3339),
	); err != nil {
		return fmt.Errorf("record migration %d: %w", m.Version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %d: %w", m.Version, err)
	}
	return nil
}

func ensureTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
        version    INTEGER PRIMARY KEY,
        name       TEXT    NOT NULL,
        applied_at TEXT    NOT NULL
    )`)
	if err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}
	return nil
}

func appliedVersions(db *sql.DB) (map[int]bool, error) {
	rows, err := db.Query(`SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("list applied migrations: %w", err)
	}
	defer rows.Close()

	out := map[int]bool{}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = true
	}
	return out, rows.Err()
}

// Status returns the current and latest schema versions.
func Status(db *sql.DB) (current, latest int, err error) {
	if current, err = CurrentVersion(db); err != nil {
		return 0, 0, err
	}
	if latest, err = LatestVersion(); err != nil {
		return 0, 0, err
	}
	return current, latest, nil
}

// Names returns "NNNN_name" identifiers for logging.
func Names() ([]string, error) {
	ms, err := Load()
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(ms))
	for _, m := range ms {
		out = append(out, fmt.Sprintf("%04d_%s", m.Version, m.Name))
	}
	return out, nil
}

// FormatVersions renders applied versions for logs.
func FormatVersions(vs []int) string {
	if len(vs) == 0 {
		return "none"
	}
	parts := make([]string, len(vs))
	for i, v := range vs {
		parts[i] = fmt.Sprintf("%04d", v)
	}
	return strings.Join(parts, ",")
}
