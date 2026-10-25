package migrations

import (
	"path/filepath"
	"testing"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/store"
)

func openMigrated(t *testing.T) *store.Store {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := Apply(db); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	return store.New(db)
}

func columnExists(t *testing.T, st *store.Store, table, column string) bool {
	t.Helper()
	rows, err := st.DB().Query(`SELECT name FROM pragma_table_info(?)`, table)
	if err != nil {
		t.Fatalf("pragma_table_info(%s): %v", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan column name: %v", err)
		}
		if name == column {
			return true
		}
	}
	return false
}

// TestIconMetadataMigration checks that migration 0003 adds the icon
// bookkeeping columns without breaking existing rows.
func TestIconMetadataMigration(t *testing.T) {
	st := openMigrated(t)

	version, err := CurrentVersion(st.DB())
	if err != nil {
		t.Fatalf("CurrentVersion() error = %v", err)
	}
	if version < 3 {
		t.Fatalf("schema version = %d, want >= 3 after the icon metadata migration", version)
	}

	for _, col := range []string{"icon_source", "icon_status", "icon_error", "icon_checked_at"} {
		if !columnExists(t, st, "bookmarks", col) {
			t.Errorf("bookmarks.%s is missing", col)
		}
	}
	for _, col := range []string{"source_url", "origin"} {
		if !columnExists(t, st, "uploads", col) {
			t.Errorf("uploads.%s is missing", col)
		}
	}
}

// TestIconMetadataDefaultsAreBackwardsCompatible inserts rows the way the
// pre-2.1 code did and checks the new columns fall back to safe defaults.
func TestIconMetadataDefaultsAreBackwardsCompatible(t *testing.T) {
	st := openMigrated(t)
	ctx := t.Context()

	// A bookmark created without any icon bookkeeping.
	if _, err := st.DB().ExecContext(ctx,
		`INSERT INTO categories (name, description, icon, color, sort_order, is_visible, created_at, updated_at)
         VALUES ('旧分类', '', '', '', 0, 1, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatalf("insert category: %v", err)
	}
	if _, err := st.DB().ExecContext(ctx,
		`INSERT INTO bookmarks (category_id, title, url, description, icon_type, icon_value, keywords, tags, color,
             sort_order, is_visible, open_in_new_tab, created_at, updated_at)
         VALUES (1, '旧站点', 'https://example.com', '', 'none', '', '', '', '', 0, 1, 1,
             '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatalf("insert bookmark: %v", err)
	}

	bookmark, err := st.GetBookmark(ctx, 1)
	if err != nil {
		t.Fatalf("GetBookmark() error = %v", err)
	}
	if bookmark.IconSource != "" || bookmark.IconStatus != "" || bookmark.IconError != "" || bookmark.IconCheckedAt != "" {
		t.Errorf("icon bookkeeping = (%q, %q, %q, %q), want empty defaults",
			bookmark.IconSource, bookmark.IconStatus, bookmark.IconError, bookmark.IconCheckedAt)
	}

	// An upload inserted the old way is treated as a user upload.
	if _, err := st.DB().ExecContext(ctx,
		`INSERT INTO uploads (filename, original_name, mime, size, created_at)
         VALUES ('abc.png', 'abc.png', 'image/png', 10, '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatalf("insert upload: %v", err)
	}
	upload, err := st.GetUploadByFilename(ctx, "abc.png")
	if err != nil {
		t.Fatalf("GetUploadByFilename() error = %v", err)
	}
	if upload.Origin != "manual" {
		t.Errorf("upload origin = %q, want manual", upload.Origin)
	}
	if upload.SourceURL != "" {
		t.Errorf("upload source_url = %q, want empty", upload.SourceURL)
	}
}
