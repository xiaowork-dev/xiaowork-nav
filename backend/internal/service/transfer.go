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

	// Format version of the export bundle itself (not the database schema).
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
		Format     string            `json:"format"`
		Categories []json.RawMessage `json:"categories"`
		Bookmarks  []json.RawMessage `json:"bookmarks"`
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
		for _, key := range []string{"name", "bookmarks", "sort_order"} {
			if _, ok := probe[key]; ok {
				return true
			}
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
	Color     string       `json:"color"`
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
	Tags        string `json:"tags"`
	Keywords    string `json:"keywords"`
	Color       string `json:"color"`
}

type legacyStats struct {
	categories       int
	items            int
	siAsIcon         int
	siUnmatched      int
	tagsMapped       int
	colorsMapped     int
	colorsUnmapped   int
	categoryIcons    int
	textIcons        int
	faviconIcons     int
	dataURIs         int
	externalIcons    int
	missingName      int
	missingURL       int
}

func normalizeLegacy(raw []byte) (*models.ExportBundle, []string, error) {
	var categories []legacyCategory
	if trimmed := bytes.TrimSpace(raw); len(trimmed) > 0 && trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, &categories); err != nil {
			return nil, nil, Validation("validation_failed", "无法解析旧站数据："+err.Error())
		}
	} else {
		var doc legacyDocument
		if err := json.Unmarshal(bytes.TrimSpace(raw), &doc); err != nil {
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

		categoryIcon := strings.TrimSpace(lc.Icon)
		if categoryIcon != "" {
			stats.categoryIcons++
		}

		cat := models.Category{
			ID:        nextID,
			Name:      name,
			Icon:      categoryIcon,
			Color:     normalizeLegacyColor(lc.Color, stats),
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
			iconType, iconValue, siUsedAsIcon := mapLegacyIcon(item, stats)
			tags := buildLegacyTags(item)
			if tags != "" {
				stats.tagsMapped++
			}

			bookmark := models.Bookmark{
				CategoryID:   cat.ID,
				Title:        title,
				URL:          url,
				Description:  strings.TrimSpace(firstNonEmpty(item.Description, item.Desc)),
				IconType:     iconType,
				IconValue:    iconValue,
				Keywords:     buildLegacyKeywords(item, siUsedAsIcon),
				Tags:         tags,
				Color:        normalizeLegacyColor(item.Color, stats),
				SortOrder:    position * 10,
				IsVisible:    true,
				OpenInNewTab: true,
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
// Priority (a user-supplied icon always wins):
//  1. `logo` http(s) URL      -> icon_type=url, value kept verbatim
//  2. `logo` data: URI        -> kept verbatim; converted to an upload later
//  3. `logo` short text       -> icon_type=text (e.g. 「延迟」)
//  4. `si` verified slug      -> icon_type=simple (Simple Icons CDN)
//  5. `iconType=favicon`      -> icon_type=favicon (display-time fallback)
//  6. otherwise               -> icon_type=none (frontend tries favicon, then letter)
func mapLegacyIcon(item legacyItem, stats *legacyStats) (string, string, bool) {
	value := strings.TrimSpace(firstNonEmpty(item.Logo, item.Icon, item.IconURL, item.Favicon))
	if value != "" {
		if IsDataURI(value) {
			stats.dataURIs++
			return "url", value, false
		}
		if security.ValidateHTTPURL(value) == nil {
			stats.externalIcons++
			return "url", value, false
		}
		// A short text glyph used as an icon.
		stats.textIcons++
		return "text", value, false
	}

	si := strings.TrimSpace(item.SI)
	if si != "" {
		if IsSimpleIconSlug(si) {
			stats.siAsIcon++
			return "simple", strings.ToLower(si), true
		}
		stats.siUnmatched++
	}

	if strings.EqualFold(strings.TrimSpace(item.IconType), "favicon") {
		stats.faviconIcons++
		return "favicon", "", false
	}
	return "none", "", false
}

func buildLegacyKeywords(item legacyItem, siUsedAsIcon bool) string {
	parts := []string{}
	for _, v := range []string{item.Keywords} {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	// Keep an unmatched `si` searchable; a matched one becomes the icon.
	if si := strings.TrimSpace(item.SI); si != "" && !siUsedAsIcon {
		parts = append(parts, si)
	}
	return strings.Join(parts, ",")
}

func buildLegacyTags(item legacyItem) string {
	parts := []string{}
	for _, v := range []string{item.Tag, item.Tags} {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return strings.Join(parts, ",")
}

// normalizeLegacyColor accepts a legacy hex colour, dropping anything invalid.
func normalizeLegacyColor(value string, stats *legacyStats) string {
	v := strings.TrimSpace(value)
	if v == "" {
		return ""
	}
	if hexColorPattern.MatchString(v) {
		stats.colorsMapped++
		return v
	}
	stats.colorsUnmapped++
	return ""
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
			"%d 个文字型图标（如「延迟」）已保留为 icon_type=text", stats.textIcons))
	}
	if stats.siAsIcon > 0 {
		warnings = append(warnings, fmt.Sprintf(
			"%d 项旧站字段 si 已匹配到可用的 Simple Icons 图标（icon_type=simple）", stats.siAsIcon))
	}
	if stats.siUnmatched > 0 {
		warnings = append(warnings, fmt.Sprintf(
			"%d 项旧站字段 si 不是可用的 Simple Icons 标识，已保留到关键词", stats.siUnmatched))
	}
	if stats.faviconIcons > 0 {
		warnings = append(warnings, fmt.Sprintf(
			"%d 项标记为 favicon 的图标改为显示时回退（不抓取、不覆盖）", stats.faviconIcons))
	}
	if stats.tagsMapped > 0 {
		warnings = append(warnings, fmt.Sprintf("%d 项旧站 tag 已迁移到 tags 字段", stats.tagsMapped))
	}
	if stats.categoryIcons > 0 {
		warnings = append(warnings, fmt.Sprintf("%d 个分类 emoji 图标已迁移到 categories.icon", stats.categoryIcons))
	}
	if stats.colorsMapped > 0 {
		warnings = append(warnings, fmt.Sprintf("%d 项颜色已迁移到 color 字段", stats.colorsMapped))
	}
	if stats.colorsUnmapped > 0 {
		warnings = append(warnings, fmt.Sprintf("%d 项颜色格式非法，已忽略", stats.colorsUnmapped))
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
	kept           int
	converted      int
	simple         int
	text           int
	favicon        int
	dropped        int
	createdUploads []string
}

// sanitizeIcons validates icon values and, when convert is true, materialises
// embedded data: URIs into uploaded files. Uploads created here are tracked so
// they can be removed if the surrounding import fails.
func (t *TransferService) sanitizeIcons(ctx context.Context, bundle *models.ExportBundle, convert bool) (iconStats, []string, error) {
	stats := iconStats{}
	warnings := []string{}
	failed := 0

	for i := range bundle.Bookmarks {
		b := &bundle.Bookmarks[i]

		// Types that intentionally carry no stored value.
		switch b.IconType {
		case "", "none", "auto":
			b.IconType, b.IconValue = "none", ""
			continue
		case "favicon":
			b.IconType, b.IconValue = "favicon", ""
			stats.favicon++
			continue
		}

		value := strings.TrimSpace(b.IconValue)
		if value == "" {
			b.IconType, b.IconValue = "none", ""
			continue
		}

		switch b.IconType {
		case "text":
			runes := []rune(value)
			if len(runes) > maxIconTextRunes {
				runes = runes[:maxIconTextRunes]
			}
			b.IconValue = string(runes)
			stats.text++
			continue
		case "simple":
			if IsSimpleIconSlug(value) {
				b.IconValue = strings.ToLower(value)
				stats.simple++
				continue
			}
			b.IconType, b.IconValue = "none", ""
			stats.dropped++
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
			stats.createdUploads = append(stats.createdUploads, upload.Filename)
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
	if stats.simple > 0 {
		warnings = append(warnings, fmt.Sprintf("%d 个图标来自 Simple Icons（icon_type=simple）", stats.simple))
	}
	if stats.text > 0 {
		warnings = append(warnings, fmt.Sprintf("%d 个文字型图标（icon_type=text）", stats.text))
	}
	if stats.favicon > 0 {
		warnings = append(warnings, fmt.Sprintf("%d 个图标使用网站 Favicon 回退（icon_type=favicon）", stats.favicon))
	}
	if failed > 0 {
		warnings = append(warnings, fmt.Sprintf("%d 个内嵌图标转换失败，已设为无图标", failed))
	}
	if stats.dropped > 0 {
		warnings = append(warnings, fmt.Sprintf("%d 个图标值无效，已设为无图标", stats.dropped))
	}
	return stats, warnings, nil
}

// cleanupUploads removes icon files (and their metadata) created during a
// failed import, so a rolled-back import leaves nothing behind.
func (t *TransferService) cleanupUploads(ctx context.Context, filenames []string) {
	if t.icons == nil {
		return
	}
	for _, name := range filenames {
		_ = t.icons.DeleteUpload(ctx, name)
	}
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
		IconsSimple:        iconStats.simple,
		IconsText:          iconStats.text,
		IconsFavicon:       iconStats.favicon,
		IconsDropped:       iconStats.dropped,
		Warnings:           warnings,
	}, nil
}

// Import applies a bundle using the given conflict strategy.
//
// The database write happens in a single transaction. Icon files created for
// embedded data URIs are removed again if the transaction fails, so a failed
// import leaves neither partial data nor orphan files.
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
	iconStats, _, err := t.sanitizeIcons(ctx, bundle, true)
	if err != nil {
		t.cleanupUploads(ctx, iconStats.createdUploads)
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
		t.cleanupUploads(ctx, iconStats.createdUploads)
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
