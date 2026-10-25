package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/models"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/security"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/store"
)

// Setting keys persisted in site_settings.
const (
	keySiteTitle           = "site_title"
	keySiteFavicon         = "site_favicon"
	keyTheme               = "theme"
	keyBackground          = "background"
	keySearchEngines       = "search_engines"
	keyDefaultSearchEngine = "default_search_engine"
	keyOpenInNewTabDefault = "open_in_new_tab_default"
	keyHomepageLayout      = "homepage_layout"
	keyAllowGuestAccess    = "allow_guest_access"
)

var allowedSettingKeys = map[string]bool{
	keySiteTitle:           true,
	keySiteFavicon:         true,
	keyTheme:               true,
	keyBackground:          true,
	keySearchEngines:       true,
	keyDefaultSearchEngine: true,
	keyOpenInNewTabDefault: true,
	keyHomepageLayout:      true,
	keyAllowGuestAccess:    true,
}

// SettingsService reads and writes site configuration.
type SettingsService struct {
	store *store.Store
}

// NewSettingsService builds a SettingsService.
func NewSettingsService(st *store.Store) *SettingsService {
	return &SettingsService{store: st}
}

// Defaults returns the built-in default configuration.
func Defaults() models.Settings {
	return models.Settings{
		SiteTitle:   "XiaoWork Nav",
		SiteFavicon: "",
		Theme:       "auto",
		Background:  map[string]any{},
		SearchEngines: []models.SearchEngine{
			{Name: "Google", URLTemplate: "https://www.google.com/search?q={query}"},
			{Name: "Bing", URLTemplate: "https://www.bing.com/search?q={query}"},
			{Name: "百度", URLTemplate: "https://www.baidu.com/s?wd={query}"},
		},
		DefaultSearchEngine: "Google",
		OpenInNewTabDefault: true,
		HomepageLayout:      "scroll",
		AllowGuestAccess:    true,
	}
}

// normalizeHomepageLayout maps legacy values onto the V2.1 names.
//
// V2.0 stored `grid`/`list` for a setting the public page never read; V2.1
// replaces them with the two real navigation layouts. Normalising on read
// keeps existing databases working without rewriting user data.
func normalizeHomepageLayout(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "grid", "scroll":
		return "scroll"
	case "list", "tabs":
		return "tabs"
	default:
		return value
	}
}

// Get returns the effective settings (defaults merged with stored values).
func (s *SettingsService) Get(ctx context.Context) (models.Settings, error) {
	stored, err := s.store.ListSettings(ctx)
	if err != nil {
		return models.Settings{}, err
	}
	settings := Defaults()
	for key, raw := range stored {
		if err := applySetting(&settings, key, raw); err != nil {
			return models.Settings{}, fmt.Errorf("apply setting %q: %w", key, err)
		}
	}
	settings.HomepageLayout = normalizeHomepageLayout(settings.HomepageLayout)
	return settings, nil
}

// Public returns the visitor-visible subset.
func (s *SettingsService) Public(ctx context.Context) (models.PublicSettings, error) {
	settings, err := s.Get(ctx)
	if err != nil {
		return models.PublicSettings{}, err
	}
	return models.PublicSettings{
		SiteTitle:           settings.SiteTitle,
		SiteFavicon:         settings.SiteFavicon,
		Theme:               settings.Theme,
		Background:          settings.Background,
		SearchEngines:       settings.SearchEngines,
		DefaultSearchEngine: settings.DefaultSearchEngine,
		OpenInNewTabDefault: settings.OpenInNewTabDefault,
		HomepageLayout:      settings.HomepageLayout,
	}, nil
}

// AllowGuestAccess reports whether anonymous visitors may read public data.
func (s *SettingsService) AllowGuestAccess(ctx context.Context) (bool, error) {
	settings, err := s.Get(ctx)
	if err != nil {
		return false, err
	}
	return settings.AllowGuestAccess, nil
}

// Update applies a partial settings patch and returns the effective settings.
func (s *SettingsService) Update(ctx context.Context, patch map[string]json.RawMessage) (models.Settings, error) {
	if len(patch) == 0 {
		return s.Get(ctx)
	}

	current, err := s.Get(ctx)
	if err != nil {
		return models.Settings{}, err
	}

	values := make(map[string]string, len(patch))
	for key, raw := range patch {
		if !allowedSettingKeys[key] {
			return models.Settings{}, Validation("validation_failed", "存在未知的设置项",
				FieldError{Field: key, Issue: "未知的设置键"})
		}
		if err := applySetting(&current, key, string(raw)); err != nil {
			return models.Settings{}, Validation("validation_failed", "设置项格式错误",
				FieldError{Field: key, Issue: err.Error()})
		}
		values[key] = string(raw)
	}

	if err := validateSettings(current); err != nil {
		return models.Settings{}, err
	}
	if err := s.store.UpsertSettings(ctx, values); err != nil {
		return models.Settings{}, err
	}
	return current, nil
}

func applySetting(settings *models.Settings, key, raw string) error {
	switch key {
	case keySiteTitle:
		return json.Unmarshal([]byte(raw), &settings.SiteTitle)
	case keySiteFavicon:
		return json.Unmarshal([]byte(raw), &settings.SiteFavicon)
	case keyTheme:
		return json.Unmarshal([]byte(raw), &settings.Theme)
	case keyBackground:
		return json.Unmarshal([]byte(raw), &settings.Background)
	case keySearchEngines:
		return json.Unmarshal([]byte(raw), &settings.SearchEngines)
	case keyDefaultSearchEngine:
		return json.Unmarshal([]byte(raw), &settings.DefaultSearchEngine)
	case keyOpenInNewTabDefault:
		return json.Unmarshal([]byte(raw), &settings.OpenInNewTabDefault)
	case keyHomepageLayout:
		if err := json.Unmarshal([]byte(raw), &settings.HomepageLayout); err != nil {
			return err
		}
		settings.HomepageLayout = normalizeHomepageLayout(settings.HomepageLayout)
		return nil
	case keyAllowGuestAccess:
		return json.Unmarshal([]byte(raw), &settings.AllowGuestAccess)
	default:
		return nil
	}
}

func validateSettings(s models.Settings) error {
	if strings.TrimSpace(s.SiteTitle) == "" {
		return Validation("validation_failed", "站点标题不能为空",
			FieldError{Field: keySiteTitle, Issue: "不能为空"})
	}
	if len([]rune(s.SiteTitle)) > 128 {
		return Validation("validation_failed", "站点标题过长",
			FieldError{Field: keySiteTitle, Issue: "最多 128 字符"})
	}
	switch s.Theme {
	case "light", "dark", "auto":
	default:
		return Validation("validation_failed", "主题取值非法",
			FieldError{Field: keyTheme, Issue: "仅允许 light/dark/auto"})
	}
	switch s.HomepageLayout {
	case "scroll", "tabs":
	default:
		return Validation("validation_failed", "导航布局取值非法",
			FieldError{Field: keyHomepageLayout, Issue: "仅允许 scroll/tabs"})
	}
	if s.SiteFavicon != "" && !isAssetRef(s.SiteFavicon) {
		return Validation("validation_failed", "Favicon 地址非法",
			FieldError{Field: keySiteFavicon, Issue: "需为 http(s) URL 或 /uploads 路径"})
	}
	if len(s.SearchEngines) == 0 {
		return Validation("validation_failed", "至少需要一个搜索引擎",
			FieldError{Field: keySearchEngines, Issue: "不能为空"})
	}
	names := map[string]bool{}
	for i, e := range s.SearchEngines {
		if strings.TrimSpace(e.Name) == "" {
			return Validation("validation_failed", "搜索引擎名称不能为空",
				FieldError{Field: keySearchEngines, Issue: fmt.Sprintf("第 %d 项缺少名称", i+1)})
		}
		if !strings.Contains(e.URLTemplate, "{query}") {
			return Validation("validation_failed", "搜索引擎模板缺少 {query} 占位符",
				FieldError{Field: keySearchEngines, Issue: fmt.Sprintf("第 %d 项模板非法", i+1)})
		}
		probe := strings.ReplaceAll(e.URLTemplate, "{query}", "test")
		if err := security.ValidateHTTPURL(probe); err != nil {
			return Validation("validation_failed", "搜索引擎模板不是合法 URL",
				FieldError{Field: keySearchEngines, Issue: fmt.Sprintf("第 %d 项模板非法", i+1)})
		}
		names[e.Name] = true
	}
	if !names[s.DefaultSearchEngine] {
		return Validation("validation_failed", "默认搜索引擎必须存在于引擎列表中",
			FieldError{Field: keyDefaultSearchEngine, Issue: "未找到对应引擎"})
	}
	return nil
}

// isAssetRef accepts an absolute http(s) URL or a site-relative asset path.
func isAssetRef(v string) bool {
	if strings.HasPrefix(v, "/") {
		return true
	}
	return security.ValidateHTTPURL(v) == nil
}
