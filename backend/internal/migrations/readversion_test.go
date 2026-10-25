package migrations

import (
	"path/filepath"
	"testing"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/store"
)

// TestReadVersionIsNonDestructive verifies the read-only inspection path used
// by the deployment scripts: it must report the schema version, must not create
// anything on an empty database, and must work while another connection holds
// the file.
func TestReadVersionIsNonDestructive(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "app.db")

	// A database file with no schema_migrations table at all.
	empty, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	if _, err := empty.Exec(`CREATE TABLE placeholder (id INTEGER)`); err != nil {
		t.Fatalf("create placeholder: %v", err)
	}

	v, err := ReadVersion(empty)
	if err != nil {
		t.Fatalf("ReadVersion() error = %v", err)
	}
	if v != 0 {
		t.Errorf("ReadVersion() on a database without schema_migrations = %d, want 0", v)
	}

	// ReadVersion must not have created the table.
	var exists int
	if err := empty.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'`,
	).Scan(&exists); err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if exists != 0 {
		t.Error("ReadVersion() created schema_migrations; it must be read-only")
	}

	// Apply migrations, then read again while the first connection is still open.
	applied, err := Apply(empty)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if len(applied) == 0 {
		t.Fatal("expected migrations to be applied")
	}

	latest, err := LatestVersion()
	if err != nil {
		t.Fatalf("LatestVersion() error = %v", err)
	}

	inspector, err := store.OpenForInspection(dbPath)
	if err != nil {
		t.Fatalf("store.OpenForInspection() error = %v", err)
	}
	defer inspector.Close()

	v, err = ReadVersion(inspector)
	if err != nil {
		t.Fatalf("ReadVersion() error = %v", err)
	}
	if v != latest {
		t.Errorf("ReadVersion() = %d, want %d", v, latest)
	}

	_ = empty.Close()
}
