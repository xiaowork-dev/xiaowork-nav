// Package models defines the domain entities shared by store, service and api.
//
// JSON tags mirror docs/openapi/openapi.yaml so that API responses stay
// consistent with the contract. Timestamps are RFC 3339 UTC strings.
package models

import "time"

// NowUTC returns the current UTC time formatted for storage and transport.
func NowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// FormatUTC formats a time as an RFC 3339 UTC string.
func FormatUTC(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// User is the single administrator account.
type User struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// Session is a server-side login session.
type Session struct {
	ID         string
	UserID     int64
	CreatedAt  string
	ExpiresAt  string
	LastSeenAt string
	UserAgent  string
	IP         string
}

// Category groups bookmarks.
type Category struct {
	ID            int64      `json:"id"`
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	Icon          string     `json:"icon"`
	Color         string     `json:"color"`
	SortOrder     int        `json:"sort_order"`
	IsVisible     bool       `json:"is_visible"`
	BookmarkCount int        `json:"bookmark_count,omitempty"`
	CreatedAt     string     `json:"created_at"`
	UpdatedAt     string     `json:"updated_at"`
	Bookmarks     []Bookmark `json:"bookmarks,omitempty"`
}

// Bookmark is a navigation entry.
//
// IconType values: none | url | upload | auto | favicon | simple | text.
type Bookmark struct {
	ID           int64  `json:"id"`
	CategoryID   int64  `json:"category_id"`
	Title        string `json:"title"`
	URL          string `json:"url"`
	Description  string `json:"description"`
	IconType     string `json:"icon_type"`
	IconValue    string `json:"icon_value"`
	Keywords     string `json:"keywords"`
	Tags         string `json:"tags"`
	Color        string `json:"color"`
	SortOrder    int    `json:"sort_order"`
	IsVisible    bool   `json:"is_visible"`
	OpenInNewTab bool   `json:"open_in_new_tab"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// SearchEngine is a configurable search provider.
type SearchEngine struct {
	Name        string `json:"name"`
	URLTemplate string `json:"url_template"`
}

// Settings is the full site configuration (admin view).
type Settings struct {
	SiteTitle           string         `json:"site_title"`
	SiteFavicon         string         `json:"site_favicon"`
	Theme               string         `json:"theme"`
	Background          map[string]any `json:"background"`
	SearchEngines       []SearchEngine `json:"search_engines"`
	DefaultSearchEngine string         `json:"default_search_engine"`
	OpenInNewTabDefault bool           `json:"open_in_new_tab_default"`
	HomepageLayout      string         `json:"homepage_layout"`
	AllowGuestAccess    bool           `json:"allow_guest_access"`
}

// PublicSettings is the visitor-visible subset of Settings.
type PublicSettings struct {
	SiteTitle           string         `json:"site_title"`
	SiteFavicon         string         `json:"site_favicon"`
	Theme               string         `json:"theme"`
	Background          map[string]any `json:"background"`
	SearchEngines       []SearchEngine `json:"search_engines"`
	DefaultSearchEngine string         `json:"default_search_engine"`
	OpenInNewTabDefault bool           `json:"open_in_new_tab_default"`
	HomepageLayout      string         `json:"homepage_layout"`
}

// Navigation is the aggregated payload for the public home page.
type Navigation struct {
	Categories []Category     `json:"categories"`
	Settings   PublicSettings `json:"settings"`
}

// Upload is metadata for an uploaded icon file.
type Upload struct {
	ID           int64  `json:"id"`
	Filename     string `json:"filename"`
	OriginalName string `json:"original_name"`
	Mime         string `json:"mime"`
	Size         int64  `json:"size"`
	CreatedAt    string `json:"created_at"`
}

// Backup describes a database snapshot on disk.
type Backup struct {
	ID            string `json:"id"`
	CreatedAt     string `json:"created_at"`
	AppVersion    string `json:"app_version"`
	SchemaVersion int    `json:"schema_version"`
	Size          int64  `json:"size"`
}

// ExportBundle is the JSON data package used by export/import.
type ExportBundle struct {
	Format        string           `json:"format"`
	SchemaVersion int              `json:"schema_version"`
	ExportedAt    string           `json:"exported_at,omitempty"`
	Checksum      string           `json:"checksum,omitempty"`
	Categories    []Category       `json:"categories,omitempty"`
	Bookmarks     []Bookmark       `json:"bookmarks,omitempty"`
	Settings      map[string]any   `json:"settings,omitempty"`
}

// ImportPreview summarises what an import would change without writing.
type ImportPreview struct {
	Format             string   `json:"format"`
	CategoriesNew      int      `json:"categories_new"`
	CategoriesConflict int      `json:"categories_conflict"`
	BookmarksNew       int      `json:"bookmarks_new"`
	BookmarksConflict  int      `json:"bookmarks_conflict"`
	BookmarksInvalid   int      `json:"bookmarks_invalid"`
	IconsKept          int      `json:"icons_kept"`
	IconsConverted     int      `json:"icons_converted"`
	IconsSimple        int      `json:"icons_simple"`
	IconsText          int      `json:"icons_text"`
	IconsFavicon       int      `json:"icons_favicon"`
	IconsDropped       int      `json:"icons_dropped"`
	Warnings           []string `json:"warnings,omitempty"`
}

// ImportResult reports the outcome of an applied import.
type ImportResult struct {
	OK                 bool `json:"ok"`
	CategoriesImported int  `json:"categories_imported"`
	BookmarksImported  int  `json:"bookmarks_imported"`
	Skipped            int  `json:"skipped"`
}

// OrderItem is one entry of a reorder request.
type OrderItem struct {
	ID        int64 `json:"id"`
	SortOrder int   `json:"sort_order"`
}
