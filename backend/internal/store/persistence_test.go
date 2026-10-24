package store

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/migrations"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/models"
)

// TestDataPersistsAcrossReopen verifies that data written through the store
// survives closing and reopening the database file.
func TestDataPersistsAcrossReopen(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "app.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if _, err := migrations.Apply(db); err != nil {
		t.Fatalf("migrations.Apply() error = %v", err)
	}

	created, err := New(db).CreateCategory(ctx, &models.Category{Name: "Persisted", IsVisible: true})
	if err != nil {
		t.Fatalf("CreateCategory() error = %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// Reopen the same file; migrations must be a no-op and data must remain.
	db2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("reopen error = %v", err)
	}
	defer db2.Close()

	applied, err := migrations.Apply(db2)
	if err != nil {
		t.Fatalf("migrations.Apply() after reopen error = %v", err)
	}
	if len(applied) != 0 {
		t.Errorf("migrations re-applied on an existing database: %v", applied)
	}

	got, err := New(db2).GetCategory(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetCategory() after reopen error = %v", err)
	}
	if got.Name != "Persisted" {
		t.Errorf("name after reopen = %q, want Persisted", got.Name)
	}
}
