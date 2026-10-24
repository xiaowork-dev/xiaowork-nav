package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/models"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/security"
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
	icons *IconService
}

// NewTransferService builds a TransferService.
func NewTransferService(st *store.Store, icons *IconService) *TransferService {
	return &TransferService{store: st, icons: icons}
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

// DecodeBundle parses a bundle in native or legacy form and normalises legacy
// payloads into the native shape.
//
// Supported shapes:
//   - native object with `format: xiaowork-nav-v1`
//   - legacy object `{ "format": "legacy-v1", "categories": [...] }`
//   - legacy top-level array `[ { "category": ..., "items": [...] } ]`
func DecodeBundle(raw []byte) (*models.ExportBundle, []string, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, nil, Validation("validation_failed", "导入数据为空")
	}

	// Top-level array => legacy category list.
	if trimmed[0] == '[' {
		return normalizeLegacy(trimmed)
	}

	var probe struct {
		Format     string              `json:"format"`
		Categories []json.RawMessage   `json:"categories"`
		Bookmarks  []json.RawMessage   `json:"bookmarks"`
		Items      []json.RawMessage   `json:"items"`
	}
	if err := json.Unmarshal(trimmed, &probe); err != nil {
		return nil, nil, Validation("validation_failed", "导入数据不是合法 JSON")
	}

	switch strings.ToLower(strings.TrimSpace(probe.Format)) {
	case FormatNative:
		return decodeNative(trimmed)
	case FormatLegacy:
		return normalizeLegacy(trimmed)
	case "":
		// Detect by shape: native categories carry `name`/`bookmarks`/`sort_order`.
		if len(probe.Bookmarks) > 0 || looksNative(probe.Categories) {
			return decodeNative(trimmed)
		}
		return normalizeLegacy(trimmed)
	default:
		return nil, nil, Validation("validation_failed", "不支持的导入格式："+probe.Format)
	}
}

func decodeNative(raw []byte) (*models.ExportBundle, []string, error) {
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
}

func looksNative(categories []json.RawMessage) bool {
	for _, raw := range categories {
		var probe map[string]json.RawMessage
		if err := json.Unmarshal(raw, &probe); err != nil {
			continue
		}
		if _, ok := probe["name"]; ok {
			return true
		}
		if _, ok := probe["bookmarks"]; ok {
			return true
		}
		if _, ok := probe["sort_order"]; ok {
			return true
		}
	}
	return false
}

// --- legacy normalisation ---

type legacyDocument struct {
	Categories []legacyCategory `json:"categories"`
}

type legacyCategory struct {
	Category  string       `json:"category"`
	Name      string       `json:"name"`
	Title     string       `json:"title"`
	Icon      string       `json:"icon"`
	Items     []legacyItem `json:"items"`
	Links     []legacyItem `json:"links"`
	Bookmarks []legacyItem `json:"bookmarks"`
	Sites     []legacyItem `json:"sites"`
}

type legacyItem struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Link        string `json:"link"`
	Href        string `json:"href"`
	Description string `json:"description"`
	Desc        string `json:"desc"`
	Logo        string `json:"logo"`
	Icon        string `json:"icon"`
	IconURL     string `json:"iconUrl"`
	Favicon     string `json:"favicon"`
	LogoType    string `json:"logoType"`
	IconType    string `json:"iconType"`
	SI          string `json:"si"`
	Tag         string `json:"tag"`
	Keywords    string `json:"keywords"`
	Tags        string `json:"tags"`
	Color       string `json:"color"`
}

type legacyStats struct {
	categories           int
	items                int
	siMapped             int
	tagMapped            int
	colorUnmapped        int
	categoryIconUnmapped int
	textIcons            int
	dataURIs             int
	externalIcons        int
	missingName          int
	missingURL           int
}

func normalizeLegacy(raw []byte) (*models.ExportBundle, []string, error) {
	// Accept both a top-level array and an object wrapper.
	var categories []legacyCategory
	if trimmed := bytes.TrimSpace(raw); len(trimmed) > 0 && trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, &categories); err != nil {
			return nil, nil, Validation("validation_failed", "无法解析旧站数据："+err.Error())
		}
	} else {
		var doc legacyDocument
		if err := json.Unmarshal(trimmed, &doc); err != nil {
			return nil, nil, Validation("validation_failed", "无法解析旧站数据："+err.Error())
		}
		categories = doc.Categories
	}

	bundle := &models.ExportBundle{Format: FormatLegacy, SchemaVersion: currentSchemaVersion}
	stats := &legacyStats{}
	var nextID int64 = 1

	for _, lc := range categories {
		name := strings.TrimSpace(firstNonEmpty(lc.Category, lc.Name, lc.Title))
		if name == "" {
			continue
		}
		stats.categories++
		if strings.TrimSpace(lc.Icon) != "" {
			stats.categoryIconUnmapped++
		}

		cat := models.Category{
			ID:        nextID,
			Name:      name,
			SortOrder: int(nextID) * 10,
			IsVisible: true,
		}
		nextID++

		items := lc.Items
		if len(items) == 0 {
			items = lc.Links
		}
		if len(items) == 0 {
			items = lc.Bookmarks
		}
		if len(items) == 0 {
			items = lc.Sites
		}

		position := 0
		for _, item := range items {
			stats.items++

			title := strings.TrimSpace(firstNonEmpty(item.Title, item.Name))
			url := strings.TrimSpace(firstNonEmpty(item.URL, item.Link, item.Href))
			if title == "" {
				stats.missingName++
				continue
			}
			if url == "" {
				stats.missingURL++
				continue
			}

			position++
			iconType, iconValue := mapLegacyIcon(item, stats)

			bookmark := models.Bookmark{
				CategoryID:   cat.ID,
				Title:        title,
				URL:          url,
				Description:  strings.TrimSpace(firstNonEmpty(item.Description, item.Desc)),
				IconType:     iconType,
				IconValue:    iconValue,
				Keywords:     buildLegacyKeywords(item),
				SortOrder:    position * 10,
				IsVisible:    true,
				OpenInNewTab: true,
			}
			if strings.TrimSpace(item.SI) != "" {
				stats.siMapped++
			}
			if strings.TrimSpace(item.Tag) != "" {
				stats.tagMapped++
			}
			if strings.TrimSpace(item.Color) != "" {
				stats.colorUnmapped++
			}
			bundle.Bookmarks = append(bundle.Bookmarks, bookmark)
		}

		bundle.Categories = append(bundle.Categories, cat)
	}

	if len(bundle.Categories) == 0 {
		return nil, nil, Validation("validation_failed", "未在导入数据中找到任何分类")
	}
	return bundle, legacyWarnings(stats), nil
}

// mapLegacyIcon converts a legacy icon field into our icon model.
//
// Rules (see docs/legacy-data-migration-report.md):
//   - http(s) URL          -> icon_type=url, value kept verbatim (never replaced)
//   - data: URI image      -> kept verbatim here; converted to an uploaded file
//     by sanitizeIcons so the exact icon is preserved without inflating payloads
//   - short text (e.g. 延迟) -> no image; the text is preserved in keywords
//   - iconType=favicon     -> treated as "no icon" (we do not substitute a favicon)
func mapLegacyIcon(item legacyItem, stats *legacyStats) (string, string) {
	value := strings.TrimSpace(firstNonEmpty(item.Logo, item.Icon, item.IconURL, item.Favicon))
	if value == "" {
		return "none", ""
	}
	if IsDataURI(value) {
		stats.dataURIs++
		return "url", value // sanitizeIcons converts this into an upload
	}
	if security.ValidateHTTPURL(value) == nil {
		stats.externalIcons++
		return "url", value
	}
	// Not an image reference (e.g. a short text glyph).
	stats.textIcons++
	return "none", ""
}

func buildLegacyKeywords(item legacyItem) string {
	parts := []string{}
	for _, v := range []string{item.Keywords, item.Tags, item.SI, item.Tag} {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	// Preserve a text glyph so the original label is not lost.
	value := strings.TrimSpace(firstNonEmpty(item.Logo, item.Icon, item.IconURL, item.Favicon))
	if value != "" && !IsDataURI(value) && security.ValidateHTTPURL(value) != nil {
		parts = append(parts, value)
	}
	return strings.Join(parts, ",")
}

func legacyWarnings(stats *legacyStats) []string {
	warnings := []string{}
	if stats.dataURIs > 0 {
		warnings = append(warnings, fmt.Sprintf(
			"%d 个内嵌 data URI 图标将转换为本地上传文件（图标内容原样保留）", stats.dataURIs))
	}
	if stats.externalIcons > 0 {
		warnings = append(warnings, fmt.Sprintf(
			"%d 个外部图标地址已原样保留，未替换为网站默认 Favicon", stats.externalIcons))
	}
	if stats.textIcons > 0 {
		warnings = append(warnings, fmt.Sprintf(
			"%d 个文本型图标（如「延迟」）无法作为图片，已设为无图标并把文本保留到关键词", stats.textIcons))
	}
	if stats.siMapped > 0 {
		warnings = append(warnings, fmt.Sprintf(
			"%d 项旧站字段 si（图标标识）已合并到关键词", stats.siMapped))
	}
	if stats.tagMapped > 0 {
		warnings = append(warnings, fmt.Sprintf(
			"%d 项旧站字段 tag 已合并到关键词", stats.tagMapped))
	}
	if stats.colorUnmapped > 0 {
		warnings = append(warnings, fmt.Sprintf(
			"%d 项旧站字段 color 未映射（当前数据模型没有对应字段）", stats.colorUnmapped))
	}
	if stats.categoryIconUnmapped > 0 {
		warnings = append(warnings, fmt.Sprintf(
			"%d 个分类图标（emoji）未映射（当前数据模型没有对应字段）", stats.categoryIconUnmapped))
	}
	if stats.missingName > 0 {
		warnings = append(warnings, fmt.Sprintf("%d 个条目缺少名称，已忽略", stats.missingName))
	}
	if stats.missingURL > 0 {
		warnings = append(warnings, fmt.Sprintf("%d 个条目缺少网址，已忽略", stats.missingURL))
	}
	return warnings
}

// --- icon preparation ---

type iconStats struct {
	kept      int
	converted int
	dropped   int
}

// sanitizeIcons validates icon values and, when convert is true, materialises
// embedded data: URIs into uploaded files.
func (t *TransferService) sanitizeIcons(ctx context.Context, bundle *models.ExportBundle, convert bool) (iconStats, []string, error) {
	stats := iconStats{}
	warnings := []string{}
	failed := 0

	for i := range bundle.Bookmarks {
		b := &bundle.Bookmarks[i]
		value := strings.TrimSpace(b.IconValue)

		if value == "" || b.IconType == "none" {
			b.IconType, b.IconValue = "none", ""
			continue
		}

		if IsDataURI(value) {
			if !convert {
				stats.converted++
				b.IconType, b.IconValue = "upload", "(pending)"
				continue
			}
			if t.icons == nil {
				b.IconType, b.IconValue = "none", ""
				stats.dropped++
				failed++
				continue
			}
			upload, err := t.icons.SaveDataURI(ctx, value)
			if err != nil {
				b.IconType, b.IconValue = "none", ""
				stats.dropped++
				failed++
				continue
			}
			b.IconType = "upload"
			b.IconValue = UploadURL(upload.Filename)
			stats.converted++
			continue
		}

		if security.ValidateHTTPURL(value) == nil {
			b.IconType, b.IconValue = "url", value
			stats.kept++
			continue
		}

		b.IconType, b.IconValue = "none", ""
		stats.dropped++
	}

	if stats.kept > 0 {
		warnings = append(warnings, fmt.Sprintf("%d 个外部图标地址原样保留（icon_type=url）", stats.kept))
	}
	if stats.converted > 0 {
		warnings = append(warnings, fmt.Sprintf("%d 个内嵌图标已转换为本地上传（icon_type=upload）", stats.converted))
	}
	if failed > 0 {
		warnings = append(warnings, fmt.Sprintf("%d 个内嵌图标转换失败，已设为无图标", failed))
	}
	if stats.dropped > 0 {
		warnings = append(warnings, fmt.Sprintf("%d 个图标值无效，已设为无图标（原值保留在导入数据中）", stats.dropped))
	}
	return stats, warnings, nil
}

// filterValidBookmarks removes bookmarks whose URL is not an allowed http(s)
// URL, returning the valid set and the number dropped.
func filterValidBookmarks(bundle *models.ExportBundle) int {
	valid := bundle.Bookmarks[:0]
	dropped := 0
	for _, b := range bundle.Bookmarks {
		if security.ValidateHTTPURL(strings.TrimSpace(b.URL)) != nil {
			dropped++
			continue
		}
		valid = append(valid, b)
	}
	bundle.Bookmarks = valid
	return dropped
}

// Preview validates a bundle and reports what an import would change.
func (t *TransferService) Preview(ctx context.Context, raw []byte) (*models.ImportPreview, error) {
	bundle, warnings, err := DecodeBundle(raw)
	if err != nil {
		return nil, err
	}

	invalid := filterValidBookmarks(bundle)
	iconStats, iconWarnings, err := t.sanitizeIcons(ctx, bundle, false)
	if err != nil {
		return nil, err
	}
	warnings = append(warnings, iconWarnings...)

	catNew, catConflict, bmNew, bmConflict, err := t.store.PreviewImport(ctx, bundle.Categories, bundle.Bookmarks)
	if err != nil {
		return nil, err
	}

	if invalid > 0 {
		warnings = append(warnings, fmt.Sprintf("%d 个条目的网址不合法（仅允许 http/https），导入时会被跳过", invalid))
	}

	return &models.ImportPreview{
		Format:             bundle.Format,
		CategoriesNew:      catNew,
		CategoriesConflict: catConflict,
		BookmarksNew:       bmNew,
		BookmarksConflict:  bmConflict,
		BookmarksInvalid:   invalid,
		IconsKept:          iconStats.kept,
		IconsConverted:     iconStats.converted,
		IconsDropped:       iconStats.dropped,
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

	invalid := filterValidBookmarks(bundle)
	if _, _, err := t.sanitizeIcons(ctx, bundle, true); err != nil {
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
	result.Skipped += invalid
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
