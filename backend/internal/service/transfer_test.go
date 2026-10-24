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

	// Category order and trimmed names are preserved.
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

	// The invalid-URL entry is dropped during decoding? No: it is kept here and
	// filtered later, so the bundle should still contain it.
	if len(bundle.Bookmarks) != 9 {
		t.Fatalf("bookmarks = %d, want 9", len(bundle.Bookmarks))
	}

	byTitle := map[string]models.Bookmark{}
	for _, b := range bundle.Bookmarks {
		byTitle[b.Title] = b
	}

	// Item order within a category is preserved via increasing sort_order.
	first := byTitle["Example Site"]
	second := byTitle["Embedded SVG Icon"]
	if !(first.SortOrder < second.SortOrder) {
		t.Errorf("bookmark sort_order not increasing: %d, %d", first.SortOrder, second.SortOrder)
	}
	if first.CategoryID != bundle.Categories[0].ID {
		t.Errorf("first bookmark category = %d, want %d", first.CategoryID, bundle.Categories[0].ID)
	}

	// Remote icon URL kept verbatim.
	remote := byTitle["Remote SVG Icon"]
	if remote.IconType != "url" || remote.IconValue != "https://icons.example.com/logo.svg" {
		t.Errorf("remote icon = (%q, %q), want (url, original URL)", remote.IconType, remote.IconValue)
	}

	// Embedded data URI is carried through for conversion.
	embedded := byTitle["Embedded SVG Icon"]
	if !IsDataURI(embedded.IconValue) {
		t.Errorf("embedded icon value should still be a data URI, got %q", embedded.IconValue)
	}

	// Text glyph becomes no icon, with the text preserved in keywords.
	glyph := byTitle["Text Glyph Icon"]
	if glyph.IconType != "none" || glyph.IconValue != "" {
		t.Errorf("text glyph icon = (%q, %q), want (none, '')", glyph.IconType, glyph.IconValue)
	}
	if glyph.Keywords == "" {
		t.Error("text glyph should be preserved in keywords")
	}

	// si is merged into keywords.
	if byTitle["Example Site"].Keywords != "example" {
		t.Errorf("si keyword = %q, want example", byTitle["Example Site"].Keywords)
	}

	// iconType=favicon must not substitute a favicon URL.
	favicon := byTitle["Favicon Type"]
	if favicon.IconType != "none" {
		t.Errorf("iconType=favicon mapped to %q, want none", favicon.IconType)
	}

	if len(warnings) == 0 {
		t.Error("expected mapping warnings for unmapped legacy fields")
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
	// The text glyph is mapped to "no icon" during normalisation (with its own
	// warning) rather than counted as a dropped invalid icon value.
	if preview.IconsDropped != 0 {
		t.Errorf("icons_dropped = %d, want 0", preview.IconsDropped)
	}
	if !containsWarning(preview.Warnings, "文本型图标") {
		t.Errorf("expected a text-glyph warning, got %v", preview.Warnings)
	}
}

func containsWarning(warnings []string, needle string) bool {
	for _, w := range warnings {
		if strings.Contains(w, needle) {
			return true
		}
	}
	return false
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
	if !result.OK {
		t.Error("result.OK = false")
	}
	if result.CategoriesImported != 2 {
		t.Errorf("categories_imported = %d, want 2", result.CategoriesImported)
	}
	// 9 bundle bookmarks minus the invalid-URL one.
	if result.BookmarksImported != 8 {
		t.Errorf("bookmarks_imported = %d, want 8", result.BookmarksImported)
	}
	if result.Skipped != 1 {
		t.Errorf("skipped = %d, want 1", result.Skipped)
	}

	// Embedded icons became local uploads on disk.
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
	// Order preserved: the first legacy category keeps the lower sort_order.
	if categories[0].Name != "Sample 常用" || categories[1].Name != "Sample 域名" {
		t.Errorf("category order = %q, %q", categories[0].Name, categories[1].Name)
	}
	if !(categories[0].SortOrder < categories[1].SortOrder) {
		t.Errorf("category sort_order not preserved: %d, %d", categories[0].SortOrder, categories[1].SortOrder)
	}

	bookmarks, total, err := st.ListBookmarks(ctx, bookmarkFilterAll())
	if err != nil {
		t.Fatalf("ListBookmarks() error = %v", err)
	}
	if total != 8 {
		t.Fatalf("bookmarks in db = %d, want 8", total)
	}

	var uploaded, remote, none int
	for _, b := range bookmarks {
		switch b.IconType {
		case "upload":
			uploaded++
			if b.IconValue == "" || b.IconValue[:1] != "/" {
				t.Errorf("upload icon value = %q, want a site-relative upload URL", b.IconValue)
			}
		case "url":
			remote++
		case "none":
			none++
		}
	}
	if uploaded != 2 {
		t.Errorf("uploaded icons in db = %d, want 2", uploaded)
	}
	if remote != 2 {
		t.Errorf("remote icons in db = %d, want 2", remote)
	}
	if none != 4 {
		t.Errorf("no-icon bookmarks = %d, want 4", none)
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
	if total != 8 {
		t.Errorf("bookmarks after re-import = %d, want 8 (no duplicates)", total)
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
