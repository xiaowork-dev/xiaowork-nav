package migrations

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/store"
)

func TestLoadIsSortedAndUnique(t *testing.T) {
	ms, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(ms) == 0 {
		t.Fatal("expected at least one migration")
	}
	for i := 1; i < len(ms); i++ {
		if ms[i].Version <= ms[i-1].Version {
			t.Fatalf("migrations not strictly increasing: %d after %d", ms[i].Version, ms[i-1].Version)
		}
	}
}

func TestApplyIsIdempotent(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	defer db.Close()

	latest, err := LatestVersion()
	if err != nil {
		t.Fatalf("LatestVersion() error = %v", err)
	}

	applied, err := Apply(db)
	if err != nil {
		t.Fatalf("first Apply() error = %v", err)
	}
	if len(applied) == 0 {
		t.Fatal("expected migrations to be applied on an empty database")
	}

	current, err := CurrentVersion(db)
	if err != nil {
		t.Fatalf("CurrentVersion() error = %v", err)
	}
	if current != latest {
		t.Fatalf("current = %d, want %d", current, latest)
	}

	// Second run must apply nothing.
	again, err := Apply(db)
	if err != nil {
		t.Fatalf("second Apply() error = %v", err)
	}
	if len(again) != 0 {
		t.Fatalf("second Apply() applied %v, want none", again)
	}
}

func TestExpectedTablesExist(t *testing.T) {	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	defer db.Close()

	if _, err := Apply(db); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	for _, table := range []string{"users", "sessions", "categories", "bookmarks", "site_settings", "uploads", "schema_migrations"} {
		var name string
		err := db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}

func TestForeignKeysCascade(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	defer db.Close()
	if _, err := Apply(db); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if _, err := db.Exec(`INSERT INTO categories (name, created_at, updated_at) VALUES ('c', 'now', 'now')`); err != nil {
		t.Fatalf("insert category: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO bookmarks (category_id, title, url, created_at, updated_at) VALUES (1, 'b', 'https://example.com', 'now', 'now')`); err != nil {
		t.Fatalf("insert bookmark: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM categories WHERE id = 1`); err != nil {
		t.Fatalf("delete category: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM bookmarks`).Scan(&count); err != nil {
		t.Fatalf("count bookmarks: %v", err)
	}
	if count != 0 {
		t.Errorf("expected cascade delete, got %d bookmarks", count)
	}
}

// TestExtendedColumnsExist verifies the 0002 migration added the icon/label
// columns to existing tables without recreating them.
func TestExtendedColumnsExist(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	defer db.Close()
	if _, err := Apply(db); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	want := map[string][]string{
		"categories": {"icon", "color"},
		"bookmarks":  {"tags", "color"},
	}
	for table, columns := range want {
		rows, err := db.Query(`SELECT name FROM pragma_table_info(?)`, table)
		if err != nil {
			t.Fatalf("pragma_table_info(%s): %v", table, err)
		}
		have := map[string]bool{}
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				rows.Close()
				t.Fatalf("scan column: %v", err)
			}
			have[name] = true
		}
		rows.Close()
		for _, column := range columns {
			if !have[column] {
				t.Errorf("table %s is missing column %q", table, column)
			}
		}
	}

	// The migration must be recorded and not re-applied.
	applied, err := Apply(db)
	if err != nil {
		t.Fatalf("second Apply() error = %v", err)
	}
	if len(applied) != 0 {
		t.Errorf("migrations re-applied: %v", applied)
	}
	latest, err := LatestVersion()
	if err != nil {
		t.Fatalf("LatestVersion() error = %v", err)
	}
	if latest < 2 {
		t.Errorf("latest migration version = %d, want >= 2", latest)
	}
}


// TestCheckCompatibleRejectsNewerSchema verifies that a database migrated by a
// newer build is detected, which is what makes rollbacks safe to reason about.
func TestCheckCompatibleRejectsNewerSchema(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	defer db.Close()
	if _, err := Apply(db); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	// Same version: compatible.
	current, latest, err := CheckCompatible(db)
	if err != nil {
		t.Fatalf("CheckCompatible() error = %v", err)
	}
	if current != latest {
		t.Fatalf("current=%d latest=%d, want equal", current, latest)
	}

	// Pretend a newer build applied a future migration.
	if _, err := db.Exec(
		`INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)`,
		latest+1, "future", "now",
	); err != nil {
		t.Fatalf("insert future migration: %v", err)
	}

	_, _, err = CheckCompatible(db)
	if err == nil {
		t.Fatal("CheckCompatible() = nil, want ErrSchemaTooNew")
	}
	if !errors.Is(err, ErrSchemaTooNew) {
		t.Errorf("error = %v, want ErrSchemaTooNew", err)
	}
}
