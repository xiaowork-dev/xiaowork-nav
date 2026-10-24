package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/models"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/store"
)

const sampleLegacyPath = "../../../docs/samples/legacy-data.sample.json"

func bookmarkFilterAll() store.BookmarkFilter { return store.BookmarkFilter{} }

func loadSample(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.FromSlash(sampleLegacyPath))
	if err != nil {
		t.Fatalf("read sample legacy data: %v", err)
	}
	return raw
}

func containsWarning(warnings []string, needle string) bool {
	for _, w := range warnings {
		if strings.Contains(w, needle) {
			return true
		}
	}
	return false
}

func TestDecodeLegacyArrayPreservesOrderAndFields(t *testing.T) {
	bundle, warnings, err := DecodeBundle(loadSample(t))
	if err != nil {
		t.Fatalf("DecodeBundle() error = %v", err)
	}
	if bundle.Format != FormatLegacy {
		t.Errorf("format = %q, want %q", bundle.Format, FormatLegacy)
	}
	if len(bundle.Categories) != 2 {
		t.Fatalf("categories = %d, want 2", len(bundle.Categories))
	}
	if bundle.Categories[0].Name != "Sample 常用" {
		t.Errorf("category[0] = %q", bundle.Categories[0].Name)
	}
	if bundle.Categories[1].Name != "Sample 域名" {
		t.Errorf("category[1] name not trimmed: %q", bundle.Categories[1].Name)
	}
	if !(bundle.Categories[0].SortOrder < bundle.Categories[1].SortOrder) {
		t.Errorf("category sort_order not increasing: %d, %d",
			bundle.Categories[0].SortOrder, bundle.Categories[1].SortOrder)
	}
	// Category emoji icons are migrated.
	if bundle.Categories[0].Icon != "✨" || bundle.Categories[1].Icon != "🔗" {
		t.Errorf("category icons = %q, %q", bundle.Categories[0].Icon, bundle.Categories[1].Icon)
	}

	if len(bundle.Bookmarks) != 10 {
		t.Fatalf("bookmarks = %d, want 10", len(bundle.Bookmarks))
	}

	byTitle := map[string]models.Bookmark{}
	for _, b := range bundle.Bookmarks {
		byTitle[b.Title] = b
	}

	if !(byTitle["Example Site"].SortOrder < byTitle["Simple Icon Site"].SortOrder) {
		t.Error("bookmark sort_order not increasing")
	}
	if byTitle["Example Site"].CategoryID != bundle.Categories[0].ID {
		t.Errorf("first bookmark category = %d", byTitle["Example Site"].CategoryID)
	}

	// Remote icon URL kept verbatim.
	remote := byTitle["Remote SVG Icon"]
	if remote.IconType != "url" || remote.IconValue != "https://icons.example.com/logo.svg" {
		t.Errorf("remote icon = (%q, %q)", remote.IconType, remote.IconValue)
	}

	// Embedded data URI is carried through for conversion.
	if !IsDataURI(byTitle["Embedded SVG Icon"].IconValue) {
		t.Error("embedded icon value should still be a data URI")
	}

	// Text glyph keeps its own type now.
	glyph := byTitle["Text Glyph Icon"]
	if glyph.IconType != "text" || glyph.IconValue != "延迟" {
		t.Errorf("text glyph = (%q, %q), want (text, 延迟)", glyph.IconType, glyph.IconValue)
	}

	// A verified `si` slug becomes a Simple Icons icon.
	simple := byTitle["Simple Icon Site"]
	if simple.IconType != "simple" || simple.IconValue != "github" {
		t.Errorf("simple icon = (%q, %q), want (simple, github)", simple.IconType, simple.IconValue)
	}

	// An unmatched `si` stays searchable instead of becoming a CDN URL.
	example := byTitle["Example Site"]
	if example.IconType != "none" {
		t.Errorf("unmatched si produced icon type %q, want none", example.IconType)
	}
	if !strings.Contains(example.Keywords, "not-a-real-icon-slug") {
		t.Errorf("unmatched si lost from keywords: %q", example.Keywords)
	}

	// iconType=favicon is preserved as a display-time fallback.
	if byTitle["Favicon Type"].IconType != "favicon" {
		t.Errorf("favicon type = %q", byTitle["Favicon Type"].IconType)
	}

	// Legacy tag and colour are migrated to their own fields.
	favicon := byTitle["Favicon Type"]
	if favicon.Tags != "Info" {
		t.Errorf("tags = %q, want Info", favicon.Tags)
	}
	if favicon.Color != "#2D72D9" {
		t.Errorf("color = %q, want #2D72D9", favicon.Color)
	}

	if len(warnings) == 0 {
		t.Error("expected mapping warnings")
	}
}

func TestPreviewLegacySample(t *testing.T) {
	st := newTestStore(t)
	svc := NewTransferService(st, NewIconService(st, t.TempDir()))

	preview, err := svc.Preview(context.Background(), loadSample(t))
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}
	if preview.Format != FormatLegacy {
		t.Errorf("format = %q", preview.Format)
	}
	if preview.CategoriesNew != 2 {
		t.Errorf("categories_new = %d, want 2", preview.CategoriesNew)
	}
	if preview.BookmarksInvalid != 1 {
		t.Errorf("bookmarks_invalid = %d, want 1", preview.BookmarksInvalid)
	}
	// 2 remote URLs (svg + webp) kept, 2 data URIs convertible.
	if preview.IconsKept != 2 {
		t.Errorf("icons_kept = %d, want 2", preview.IconsKept)
	}
	if preview.IconsConverted != 2 {
		t.Errorf("icons_converted = %d, want 2", preview.IconsConverted)
	}
	if preview.IconsSimple != 1 {
		t.Errorf("icons_simple = %d, want 1", preview.IconsSimple)
	}
	if preview.IconsText != 1 {
		t.Errorf("icons_text = %d, want 1", preview.IconsText)
	}
	if preview.IconsFavicon != 1 {
		t.Errorf("icons_favicon = %d, want 1", preview.IconsFavicon)
	}
	if preview.IconsDropped != 0 {
		t.Errorf("icons_dropped = %d, want 0", preview.IconsDropped)
	}
}

func TestImportLegacyConvertsEmbeddedIcons(t *testing.T) {
	st := newTestStore(t)
	uploads := t.TempDir()
	svc := NewTransferService(st, NewIconService(st, uploads))
	ctx := context.Background()

	result, err := svc.Import(ctx, loadSample(t), "skip")
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result.CategoriesImported != 2 {
		t.Errorf("categories_imported = %d, want 2", result.CategoriesImported)
	}
	// 10 bundle bookmarks minus the invalid-URL one.
	if result.BookmarksImported != 9 {
		t.Errorf("bookmarks_imported = %d, want 9", result.BookmarksImported)
	}
	if result.Skipped != 1 {
		t.Errorf("skipped = %d, want 1", result.Skipped)
	}

	files, err := os.ReadDir(uploads)
	if err != nil {
		t.Fatalf("read uploads dir: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("uploaded icon files = %d, want 2", len(files))
	}

	categories, err := st.ListCategories(ctx, true, false)
	if err != nil {
		t.Fatalf("ListCategories() error = %v", err)
	}
	if len(categories) != 2 {
		t.Fatalf("categories in db = %d, want 2", len(categories))
	}
	if categories[0].Name != "Sample 常用" || categories[1].Name != "Sample 域名" {
		t.Errorf("category order = %q, %q", categories[0].Name, categories[1].Name)
	}
	if !(categories[0].SortOrder < categories[1].SortOrder) {
		t.Errorf("category sort_order not preserved: %d, %d", categories[0].SortOrder, categories[1].SortOrder)
	}
	if categories[0].Icon != "✨" {
		t.Errorf("category icon not stored: %q", categories[0].Icon)
	}

	bookmarks, total, err := st.ListBookmarks(ctx, bookmarkFilterAll())
	if err != nil {
		t.Fatalf("ListBookmarks() error = %v", err)
	}
	if total != 9 {
		t.Fatalf("bookmarks in db = %d, want 9", total)
	}

	counts := map[string]int{}
	for _, b := range bookmarks {
		counts[b.IconType]++
		if b.IconType == "upload" && (b.IconValue == "" || b.IconValue[0] != '/') {
			t.Errorf("upload icon value = %q, want a site-relative path", b.IconValue)
		}
	}
	want := map[string]int{"upload": 2, "url": 2, "simple": 1, "text": 1, "favicon": 1, "none": 2}
	for k, v := range want {
		if counts[k] != v {
			t.Errorf("icon type %q count = %d, want %d (all: %v)", k, counts[k], v, counts)
		}
	}
}

func TestImportIsIdempotentWithSkipStrategy(t *testing.T) {
	st := newTestStore(t)
	svc := NewTransferService(st, NewIconService(st, t.TempDir()))
	ctx := context.Background()

	if _, err := svc.Import(ctx, loadSample(t), "skip"); err != nil {
		t.Fatalf("first Import() error = %v", err)
	}
	second, err := svc.Import(ctx, loadSample(t), "skip")
	if err != nil {
		t.Fatalf("second Import() error = %v", err)
	}
	if second.CategoriesImported != 0 || second.BookmarksImported != 0 {
		t.Errorf("second import wrote data: %+v", second)
	}

	_, total, err := st.ListBookmarks(ctx, bookmarkFilterAll())
	if err != nil {
		t.Fatalf("ListBookmarks() error = %v", err)
	}
	if total != 9 {
		t.Errorf("bookmarks after re-import = %d, want 9", total)
	}
}

func TestDecodeDataURI(t *testing.T) {
	percent := "data:image/svg+xml,%3Csvg%3E%3C%2Fsvg%3E"
	data, mediaType, err := decodeDataURI(percent)
	if err != nil {
		t.Fatalf("decodeDataURI() error = %v", err)
	}
	if mediaType != "image/svg+xml" || string(data) != "<svg></svg>" {
		t.Errorf("decoded = (%q, %q)", mediaType, string(data))
	}

	base64URI := "data:image/png;base64,aGVsbG8="
	data, mediaType, err = decodeDataURI(base64URI)
	if err != nil {
		t.Fatalf("decodeDataURI(base64) error = %v", err)
	}
	if mediaType != "image/png" || string(data) != "hello" {
		t.Errorf("decoded base64 = (%q, %q)", mediaType, string(data))
	}

	if _, _, err := decodeDataURI("not-a-data-uri"); err == nil {
		t.Error("expected an error for a non data URI")
	}
	if _, _, err := decodeDataURI("data:image/png;base64,!!!not-base64!!!"); err == nil {
		t.Error("expected an error for invalid base64")
	}
}

func TestSimpleIconSlugValidation(t *testing.T) {
	if SimpleIconCount() < 1000 {
		t.Fatalf("bundled Simple Icons slug list looks too small: %d", SimpleIconCount())
	}
	valid := []string{"github", "vercel", "stackoverflow", "artstation", "GITHUB"}
	for _, v := range valid {
		if !IsSimpleIconSlug(v) {
			t.Errorf("IsSimpleIconSlug(%q) = false, want true", v)
		}
	}
	// Values from the real legacy export that are NOT slugs.
	invalid := []string{"", "微软", "yongge", "speed", "88完美邮箱", "not-a-real-icon-slug", "a b"}
	for _, v := range invalid {
		if IsSimpleIconSlug(v) {
			t.Errorf("IsSimpleIconSlug(%q) = true, want false", v)
		}
	}
	if got := SimpleIconURL("github"); got != "https://cdn.simpleicons.org/github" {
		t.Errorf("SimpleIconURL(github) = %q", got)
	}
	if got := SimpleIconURL("微软"); got != "" {
		t.Errorf("SimpleIconURL(微软) = %q, want empty", got)
	}
}

// TestImportRollsBackAndRemovesUploadsOnFailure forces a database error during
// the import transaction and verifies that no rows and no icon files remain.
func TestImportRollsBackAndRemovesUploadsOnFailure(t *testing.T) {
	st := newTestStore(t)
	uploads := t.TempDir()
	svc := NewTransferService(st, NewIconService(st, uploads))
	ctx := context.Background()

	// Fail any insert of a bookmark titled "BOOM".
	if _, err := st.DB().Exec(`
        CREATE TRIGGER fail_import BEFORE INSERT ON bookmarks
        WHEN NEW.title = 'BOOM'
        BEGIN SELECT RAISE(ABORT, 'forced failure'); END;`); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	raw := []byte(`[
      {"category": "Rollback", "icon": "🧪", "items": [
        {"name": "First", "url": "https://first.example/"},
        {"name": "BOOM", "url": "https://boom.example/",
         "logo": "data:image/svg+xml,%3Csvg%3E%3C%2Fsvg%3E"}
      ]}
    ]`)

	if _, err := svc.Import(ctx, raw, "skip"); err == nil {
		t.Fatal("expected the import to fail")
	}

	// No partial data.
	categories, err := st.ListCategories(ctx, true, false)
	if err != nil {
		t.Fatalf("ListCategories() error = %v", err)
	}
	if len(categories) != 0 {
		t.Errorf("categories after failed import = %d, want 0 (rollback)", len(categories))
	}
	_, total, err := st.ListBookmarks(ctx, bookmarkFilterAll())
	if err != nil {
		t.Fatalf("ListBookmarks() error = %v", err)
	}
	if total != 0 {
		t.Errorf("bookmarks after failed import = %d, want 0 (rollback)", total)
	}

	// No orphan icon files or upload rows.
	files, err := os.ReadDir(uploads)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("read uploads dir: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("orphan icon files after failed import = %d, want 0", len(files))
	}
	uploadsMeta, err := st.ListUploads(ctx)
	if err != nil {
		t.Fatalf("ListUploads() error = %v", err)
	}
	if len(uploadsMeta) != 0 {
		t.Errorf("orphan upload rows after failed import = %d, want 0", len(uploadsMeta))
	}
}
