package store

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/migrations"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/models"
)

func openStore(t *testing.T) *Store {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := migrations.Apply(db); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	return New(db)
}

func seedIconBookmarks(t *testing.T, st *Store) (categoryID int64, ids []int64) {
	t.Helper()
	ctx := context.Background()
	cat, err := st.CreateCategory(ctx, &models.Category{Name: "分类", IsVisible: true})
	if err != nil {
		t.Fatalf("CreateCategory() error = %v", err)
	}
	specs := []struct {
		title     string
		iconType  string
		iconValue string
	}{
		{"手动上传", "upload", "/api/v1/public/uploads/a.png"},
		{"手动文字", "text", "文"},
		{"无图标", "none", ""},
		{"站点图标", "favicon", ""},
		{"自动图标", "auto", "/api/v1/public/uploads/b.png"},
	}
	for _, spec := range specs {
		b, err := st.CreateBookmark(ctx, &models.Bookmark{
			CategoryID: cat.ID, Title: spec.title, URL: "https://example.com",
			IconType: spec.iconType, IconValue: spec.iconValue, IsVisible: true,
		})
		if err != nil {
			t.Fatalf("CreateBookmark(%s) error = %v", spec.title, err)
		}
		ids = append(ids, b.ID)
	}
	return cat.ID, ids
}

// TestListIconFillCandidatesExcludesManualIcons is the store-level guarantee
// behind "the automatic fetcher never touches a user-set icon".
func TestListIconFillCandidatesExcludesManualIcons(t *testing.T) {
	st := openStore(t)
	_, _ = seedIconBookmarks(t, st)

	items, err := st.ListIconFillCandidates(context.Background(), false, "", 0)
	if err != nil {
		t.Fatalf("ListIconFillCandidates() error = %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("got %d candidates, want 3 (none/favicon/auto)", len(items))
	}
	for _, item := range items {
		if item.IconType == "upload" || item.IconType == "text" {
			t.Errorf("manual icon type %q must not be a candidate", item.IconType)
		}
	}
}

func TestListIconFillCandidatesOnlyMissingHonoursRefreshWindow(t *testing.T) {
	st := openStore(t)
	_, ids := seedIconBookmarks(t, st)
	ctx := context.Background()

	// Mark the "none" bookmark as recently and successfully fetched.
	if _, err := st.SetBookmarkIcon(ctx, ids[2], IconBookkeeping{
		IconType: "auto", IconValue: "/api/v1/public/uploads/c.png",
		IconStatus: models.IconStatusOK, IconCheckedAt: "2026-10-25T10:00:00Z",
	}); err != nil {
		t.Fatalf("SetBookmarkIcon() error = %v", err)
	}

	// A refresh window that starts after that timestamp skips it.
	items, err := st.ListIconFillCandidates(ctx, true, "2026-10-20T00:00:00Z", 0)
	if err != nil {
		t.Fatalf("ListIconFillCandidates() error = %v", err)
	}
	for _, item := range items {
		if item.ID == ids[2] {
			t.Error("a recently succeeded bookmark must be skipped in only_missing mode")
		}
	}

	// A cutoff after the last check makes it eligible again (it is now stale).
	items, err = st.ListIconFillCandidates(ctx, true, "2026-10-26T00:00:00Z", 0)
	if err != nil {
		t.Fatalf("ListIconFillCandidates() error = %v", err)
	}
	found := false
	for _, item := range items {
		if item.ID == ids[2] {
			found = true
		}
	}
	if !found {
		t.Error("a bookmark whose last check is older than the refresh window must be retried")
	}

	// The limit is honoured.
	items, err = st.ListIconFillCandidates(ctx, false, "", 1)
	if err != nil {
		t.Fatalf("ListIconFillCandidates(limit) error = %v", err)
	}
	if len(items) != 1 {
		t.Errorf("limit=1 returned %d items", len(items))
	}
}

func TestCountIconSummaryAndUploadReferences(t *testing.T) {
	st := openStore(t)
	_, ids := seedIconBookmarks(t, st)
	ctx := context.Background()

	summary, err := st.CountIconSummary(ctx)
	if err != nil {
		t.Fatalf("CountIconSummary() error = %v", err)
	}
	if summary.Total != 5 {
		t.Errorf("total = %d, want 5", summary.Total)
	}
	if summary.Manual != 2 {
		t.Errorf("manual = %d, want 2 (upload + text)", summary.Manual)
	}
	if summary.Missing != 1 {
		t.Errorf("missing = %d, want 1 (none)", summary.Missing)
	}
	if summary.Favicon != 1 {
		t.Errorf("favicon = %d, want 1", summary.Favicon)
	}
	if summary.Auto != 1 {
		t.Errorf("auto = %d, want 1", summary.Auto)
	}

	// Reference counting protects shared cached icons.
	count, err := st.CountBookmarksUsingUpload(ctx, "/api/v1/public/uploads/b.png")
	if err != nil {
		t.Fatalf("CountBookmarksUsingUpload() error = %v", err)
	}
	if count != 1 {
		t.Errorf("references = %d, want 1", count)
	}

	if _, err := st.SetBookmarkIcon(ctx, ids[4], IconBookkeeping{IconType: "none"}); err != nil {
		t.Fatalf("SetBookmarkIcon() error = %v", err)
	}
	count, err = st.CountBookmarksUsingUpload(ctx, "/api/v1/public/uploads/b.png")
	if err != nil {
		t.Fatalf("CountBookmarksUsingUpload() error = %v", err)
	}
	if count != 0 {
		t.Errorf("references after clearing = %d, want 0", count)
	}
}

func TestFindUploadBySourceURLDeduplicatesCache(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()

	upload := &models.Upload{
		Filename: "cached.png", Mime: "image/png", Size: 12,
		SourceURL: "https://example.com/favicon.ico", Origin: models.UploadOriginAuto,
	}
	if err := st.CreateUpload(ctx, upload); err != nil {
		t.Fatalf("CreateUpload() error = %v", err)
	}

	found, err := st.FindUploadBySourceURL(ctx, "https://example.com/favicon.ico")
	if err != nil {
		t.Fatalf("FindUploadBySourceURL() error = %v", err)
	}
	if found.Filename != "cached.png" || found.Origin != models.UploadOriginAuto {
		t.Errorf("found = %+v", found)
	}

	if _, err := st.FindUploadBySourceURL(ctx, "https://example.com/other.ico"); err == nil {
		t.Error("an unknown source URL must not match a cached upload")
	}
	if _, err := st.FindUploadBySourceURL(ctx, ""); err == nil {
		t.Error("an empty source URL must not match a cached upload")
	}
}
