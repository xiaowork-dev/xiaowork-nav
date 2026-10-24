package migrations

import (
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

func TestExpectedTablesExist(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
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
