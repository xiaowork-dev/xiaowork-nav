package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/models"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/store"
)

// Supported export/import format identifiers.
const (
	FormatNative = "xiaowork-nav-v1"
	FormatLegacy = "legacy-v1"

	currentSchemaVersion = 1
)

// TransferService implements JSON export and import (including legacy data).
type TransferService struct {
	store *store.Store
}

// NewTransferService builds a TransferService.
func NewTransferService(st *store.Store) *TransferService {
	return &TransferService{store: st}
}

// Export produces a bundle containing all categories, bookmarks and settings.
func (t *TransferService) Export(ctx context.Context) (*models.ExportBundle, error) {
	categories, err := t.store.ListCategories(ctx, true, false)
	if err != nil {
		return nil, err
	}
	bookmarks, _, err := t.store.ListBookmarks(ctx, store.BookmarkFilter{})
	if err != nil {
		return nil, err
	}
	rawSettings, err := t.store.ListSettings(ctx)
	if err != nil {
		return nil, err
	}

	settings := make(map[string]any, len(rawSettings))
	for k, v := range rawSettings {
		var decoded any
		if err := json.Unmarshal([]byte(v), &decoded); err != nil {
			decoded = v
		}
		settings[k] = decoded
	}

	bundle := &models.ExportBundle{
		Format:        FormatNative,
		SchemaVersion: currentSchemaVersion,
		ExportedAt:    models.NowUTC(),
		Categories:    categories,
		Bookmarks:     bookmarks,
		Settings:      settings,
	}
	checksum, err := bundleChecksum(bundle)
	if err != nil {
		return nil, err
	}
	bundle.Checksum = checksum
	return bundle, nil
}

func bundleChecksum(b *models.ExportBundle) (string, error) {
	payload := struct {
		Categories []models.Category `json:"categories"`
		Bookmarks  []models.Bookmark `json:"bookmarks"`
		Settings   map[string]any    `json:"settings"`
	}{b.Categories, b.Bookmarks, b.Settings}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("checksum marshal: %w", err)
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// DecodeBundle parses a bundle in either native or legacy format, normalising
// legacy payloads into the native shape. It returns any non-fatal warnings.
func DecodeBundle(raw []byte) (*models.ExportBundle, []string, error) {
	var probe struct {
		Format string `json:"format"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, nil, Validation("validation_failed", "导入数据不是合法 JSON")
	}

	switch strings.ToLower(strings.TrimSpace(probe.Format)) {
	case FormatNative:
		var b models.ExportBundle
		if err := json.Unmarshal(raw, &b); err != nil {
			return nil, nil, Validation("validation_failed", "导入数据格式错误："+err.Error())
		}
		if b.SchemaVersion > currentSchemaVersion {
			return nil, nil, Validation("validation_failed",
				fmt.Sprintf("导入数据版本(%d)高于当前支持版本(%d)", b.SchemaVersion, currentSchemaVersion))
		}
		b.Format = FormatNative
		return &b, nil, nil
	case FormatLegacy:
		return normalizeLegacy(raw)
	case "":
		// No format marker: try native first, fall back to legacy.
		var b models.ExportBundle
		if err := json.Unmarshal(raw, &b); err == nil && (len(b.Categories) > 0 || len(b.Bookmarks) > 0) {
			b.Format = FormatNative
			return &b, nil, nil
		}
		return normalizeLegacy(raw)
	default:
		return nil, nil, Validation("validation_failed", "不支持的导入格式："+probe.Format)
	}
}

type legacyBundle struct {
	Categories []legacyCategory `json:"categories"`
}

type legacyCategory struct {
	Name      string       `json:"name"`
	Title     string       `json:"title"`
	Icon      string       `json:"icon"`
	Links     []legacyLink `json:"links"`
	Bookmarks []legacyLink `json:"bookmarks"`
	Sites     []legacyLink `json:"sites"`
}

type legacyLink struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Link        string `json:"link"`
	Href        string `json:"href"`
	Description string `json:"description"`
	Desc        string `json:"desc"`
	Icon        string `json:"icon"`
	IconURL     string `json:"iconUrl"`
	Favicon     string `json:"favicon"`
	Keywords    string `json:"keywords"`
	Tags        string `json:"tags"`
}

func normalizeLegacy(raw []byte) (*models.ExportBundle, []string, error) {
	var legacy legacyBundle
	if err := json.Unmarshal(raw, &legacy); err != nil {
		return nil, nil, Validation("validation_failed", "无法解析旧站数据："+err.Error())
	}

	var warnings []string
	bundle := &models.ExportBundle{Format: FormatLegacy, SchemaVersion: currentSchemaVersion}
	var nextID int64 = 1

	for _, lc := range legacy.Categories {
		name := firstNonEmpty(lc.Name, lc.Title)
		if strings.TrimSpace(name) == "" {
			warnings = append(warnings, "忽略了一个没有名称的分类")
			continue
		}
		cat := models.Category{
			ID:        nextID,
			Name:      strings.TrimSpace(name),
			IsVisible: true,
		}
		nextID++

		links := lc.Links
		if len(links) == 0 {
			links = lc.Bookmarks
		}
		if len(links) == 0 {
			links = lc.Sites
		}

		for _, ll := range links {
			title := firstNonEmpty(ll.Title, ll.Name)
			url := firstNonEmpty(ll.URL, ll.Link, ll.Href)
			if strings.TrimSpace(title) == "" || strings.TrimSpace(url) == "" {
				warnings = append(warnings, fmt.Sprintf("分类 %q 中存在缺少名称或链接的条目，已忽略", name))
				continue
			}
			iconValue := firstNonEmpty(ll.Icon, ll.IconURL, ll.Favicon)
			iconType := "none"
			if iconValue != "" {
				iconType = "url"
			}
			bundle.Bookmarks = append(bundle.Bookmarks, models.Bookmark{
				CategoryID:   cat.ID,
				Title:        strings.TrimSpace(title),
				URL:          strings.TrimSpace(url),
				Description:  firstNonEmpty(ll.Description, ll.Desc),
				IconType:     iconType,
				IconValue:    iconValue,
				Keywords:     firstNonEmpty(ll.Keywords, ll.Tags),
				IsVisible:    true,
				OpenInNewTab: true,
			})
		}
		bundle.Categories = append(bundle.Categories, cat)
	}

	if len(bundle.Categories) == 0 {
		return nil, nil, Validation("validation_failed", "未在导入数据中找到任何分类")
	}
	warnings = append(warnings, "已按通用规则解析旧站数据；字段映射可能需根据真实数据调整")
	return bundle, warnings, nil
}

// Preview validates a bundle and reports what an import would change.
func (t *TransferService) Preview(ctx context.Context, raw []byte) (*models.ImportPreview, error) {
	bundle, warnings, err := DecodeBundle(raw)
	if err != nil {
		return nil, err
	}
	catNew, catConflict, bmNew, bmConflict, err := t.store.PreviewImport(ctx, bundle.Categories, bundle.Bookmarks)
	if err != nil {
		return nil, err
	}
	return &models.ImportPreview{
		Format:             bundle.Format,
		CategoriesNew:      catNew,
		CategoriesConflict: catConflict,
		BookmarksNew:       bmNew,
		BookmarksConflict:  bmConflict,
		Warnings:           warnings,
	}, nil
}

// Import applies a bundle using the given conflict strategy.
func (t *TransferService) Import(ctx context.Context, raw []byte, strategy string) (*models.ImportResult, error) {
	switch strategy {
	case "", store.ImportSkip, store.ImportOverwrite, store.ImportRename:
	default:
		return nil, Validation("validation_failed", "冲突策略非法",
			FieldError{Field: "conflict_strategy", Issue: "仅允许 skip/overwrite/rename"})
	}

	bundle, _, err := DecodeBundle(raw)
	if err != nil {
		return nil, err
	}

	settings := make(map[string]string)
	for k, v := range bundle.Settings {
		if !allowedSettingKeys[k] {
			continue
		}
		encoded, err := json.Marshal(v)
		if err != nil {
			continue
		}
		settings[k] = string(encoded)
	}

	result, err := t.store.ImportData(ctx, bundle.Categories, bundle.Bookmarks, settings, strategy)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
