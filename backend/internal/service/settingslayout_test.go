package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/models"
)

// TestHomepageLayoutNormalisesLegacyValues keeps V2.0 databases working: the
// stored `grid`/`list` values map onto the two V2.1 navigation layouts.
func TestHomepageLayoutNormalisesLegacyValues(t *testing.T) {
	st := newTestStore(t)
	svc := NewSettingsService(st)
	ctx := context.Background()

	cases := map[string]string{
		"grid":   "scroll",
		"list":   "tabs",
		"scroll": "scroll",
		"tabs":   "tabs",
		"":       "scroll", // falls back to the default
	}
	for stored, want := range cases {
		raw, err := json.Marshal(stored)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if err := st.UpsertSettings(ctx, map[string]string{keyHomepageLayout: string(raw)}); err != nil {
			t.Fatalf("UpsertSettings(%q) error = %v", stored, err)
		}
		settings, err := svc.Get(ctx)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if settings.HomepageLayout != want {
			t.Errorf("stored %q -> HomepageLayout %q, want %q", stored, settings.HomepageLayout, want)
		}
	}
}

func TestHomepageLayoutUpdateValidates(t *testing.T) {
	st := newTestStore(t)
	svc := NewSettingsService(st)
	ctx := context.Background()

	// The new values are accepted.
	for _, value := range []string{"scroll", "tabs"} {
		patch := map[string]json.RawMessage{keyHomepageLayout: json.RawMessage(`"` + value + `"`)}
		settings, err := svc.Update(ctx, patch)
		if err != nil {
			t.Fatalf("Update(%q) error = %v", value, err)
		}
		if settings.HomepageLayout != value {
			t.Errorf("Update(%q) -> %q", value, settings.HomepageLayout)
		}
	}

	// Legacy values are normalised on the way in as well.
	patch := map[string]json.RawMessage{keyHomepageLayout: json.RawMessage(`"grid"`)}
	settings, err := svc.Update(ctx, patch)
	if err != nil {
		t.Fatalf("Update(grid) error = %v", err)
	}
	if settings.HomepageLayout != "scroll" {
		t.Errorf("Update(grid) -> %q, want scroll", settings.HomepageLayout)
	}

	// Anything else is rejected.
	patch = map[string]json.RawMessage{keyHomepageLayout: json.RawMessage(`"carousel"`)}
	if _, err := svc.Update(ctx, patch); err == nil {
		t.Error("Update(carousel) must be rejected")
	}
}

// TestAutoIconValueIsPersisted covers the validation change that lets an
// automatically fetched icon carry a site-relative upload path.
func TestAutoIconValueIsPersisted(t *testing.T) {
	st := newTestStore(t)
	settings := NewSettingsService(st)
	catalog := NewCatalogService(st, settings)
	ctx := context.Background()

	cat, err := catalog.CreateCategory(ctx, CategoryInput{Name: "分类"})
	if err != nil {
		t.Fatalf("CreateCategory() error = %v", err)
	}

	// A local upload path is accepted for `auto`.
	created, err := catalog.CreateBookmark(ctx, BookmarkInput{
		CategoryID: &cat.ID,
		Title:      "自动图标",
		URL:        "https://example.com",
		IconType:   "auto",
		IconValue:  UploadURLPrefix + "abc.png",
	})
	if err != nil {
		t.Fatalf("CreateBookmark(auto) error = %v", err)
	}
	if created.IconType != "auto" || created.IconValue != UploadURLPrefix+"abc.png" {
		t.Errorf("created = (%s, %s), want the stored upload path", created.IconType, created.IconValue)
	}

	// An empty value stays valid ("nothing fetched yet").
	empty, err := catalog.CreateBookmark(ctx, BookmarkInput{
		CategoryID: &cat.ID, Title: "未获取", URL: "https://example.org", IconType: "auto",
	})
	if err != nil {
		t.Fatalf("CreateBookmark(auto, empty) error = %v", err)
	}
	if empty.IconValue != "" {
		t.Errorf("icon_value = %q, want empty", empty.IconValue)
	}

	// An external URL is not valid for `auto`: those belong to `url`.
	if _, err := catalog.CreateBookmark(ctx, BookmarkInput{
		CategoryID: &cat.ID, Title: "非法", URL: "https://example.net",
		IconType: "auto", IconValue: "https://cdn.example.com/logo.png",
	}); err == nil {
		t.Error("auto with an external URL must be rejected")
	}

	// The legacy behaviour of `none`/`favicon` is unchanged.
	none, err := catalog.CreateBookmark(ctx, BookmarkInput{
		CategoryID: &cat.ID, Title: "无", URL: "https://example.io", IconType: "none", IconValue: "ignored",
	})
	if err != nil {
		t.Fatalf("CreateBookmark(none) error = %v", err)
	}
	if none.IconValue != "" {
		t.Errorf("none icon_value = %q, want empty", none.IconValue)
	}

	_ = models.IconStatusOK
}
