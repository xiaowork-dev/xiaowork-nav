package store

import (
	"path/filepath"
	"testing"
)

func TestOpenAndPragmas(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	var fk int
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&fk); err != nil {
		t.Fatalf("foreign_keys pragma: %v", err)
	}
	if fk != 1 {
		t.Errorf("foreign_keys = %d, want 1", fk)
	}

	mode, err := JournalMode(db)
	if err != nil {
		t.Fatalf("JournalMode() error = %v", err)
	}
	if mode != "wal" {
		t.Logf("journal_mode = %q (WAL unavailable on this filesystem)", mode)
	}
}

func TestOpenInvalidPath(t *testing.T) {
	// A path inside a non-existent directory cannot be opened.
	_, err := Open(filepath.Join(t.TempDir(), "missing", "deep", "test.db"))
	if err == nil {
		t.Fatal("expected error opening database in missing directory")
	}
}
