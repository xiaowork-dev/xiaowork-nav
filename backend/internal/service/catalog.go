package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/models"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/security"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/store"
)

// Field limits enforced by validation.
const (
	maxCategoryNameLen = 64
	maxTitleLen        = 128
	maxDescriptionLen  = 512
	maxKeywordsLen     = 512
	maxTagsLen         = 256
	maxIconValueLen    = 2048
	maxIconTextRunes   = 8
	maxCategoryIconLen = 8
	maxPageSize        = 100
	defaultPageSize    = 20
)

// Icon types accepted by the API.
//
//   - none    : no icon; the frontend falls back to the site favicon then a letter
//   - url     : external image URL (user supplied)
//   - upload  : icon stored locally under /api/v1/public/uploads
//   - auto    : reserved for server-side fetching
//   - favicon : derive `<origin>/favicon.ico` at display time
//   - simple  : verified Simple Icons slug rendered from the CDN
//   - text    : short text glyph (e.g. 「延迟」) rendered as a label
var allowedIconTypes = map[string]bool{
	"none":    true,
	"url":     true,
	"upload":  true,
	"auto":    true,
	"favicon": true,
	"simple":  true,
	"text":    true,
}

// CatalogService implements category and bookmark business rules.
type CatalogService struct {
	store    *store.Store
	settings *SettingsService
}

// NewCatalogService builds a CatalogService.
func NewCatalogService(st *store.Store, settings *SettingsService) *CatalogService {
	return &CatalogService{store: st, settings: settings}
}

// CategoryInput is the create payload for a category.
type CategoryInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Color       string `json:"color"`
	SortOrder   *int   `json:"sort_order"`
	IsVisible   *bool  `json:"is_visible"`
}

// BookmarkInput is the create payload for a bookmark.
type BookmarkInput struct {
	CategoryID   *int64 `json:"category_id"`
	Title        string `json:"title"`
	URL          string `json:"url"`
	Description  string `json:"description"`
	IconType     string `json:"icon_type"`
	IconValue    string `json:"icon_value"`
	Keywords     string `json:"keywords"`
	Tags         string `json:"tags"`
	Color        string `json:"color"`
	SortOrder    *int   `json:"sort_order"`
	IsVisible    *bool  `json:"is_visible"`
	OpenInNewTab *bool  `json:"open_in_new_tab"`
}

// --- Categories ---

// ListCategories returns categories for the admin view.
func (c *CatalogService) ListCategories(ctx context.Context) ([]models.Category, error) {
	return c.store.ListCategories(ctx, true, false)
}

// PublicCategories returns visible categories only.
func (c *CatalogService) PublicCategories(ctx context.Context) ([]models.Category, error) {
	return c.store.ListCategories(ctx, false, true)
}

// GetCategory returns a category by ID.
func (c *CatalogService) GetCategory(ctx context.Context, id int64) (*models.Category, error) {
	cat, err := c.store.GetCategory(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, NotFound("分类不存在")
		}
		return nil, err
	}
	return cat, nil
}

// CreateCategory validates and inserts a category.
func (c *CatalogService) CreateCategory(ctx context.Context, in CategoryInput) (*models.Category, error) {
	name, err := validateName(in.Name, maxCategoryNameLen)
	if err != nil {
		return nil, err
	}
	description, err := validateOptionalText(in.Description, maxDescriptionLen, "description")
	if err != nil {
		return nil, err
	}
	icon, err := validateCategoryIcon(in.Icon)
	if err != nil {
		return nil, err
	}
	color, err := validateColor(in.Color)
	if err != nil {
		return nil, err
	}
	cat := &models.Category{
		Name:        name,
		Description: description,
		Icon:        icon,
		Color:       color,
		SortOrder:   derefInt(in.SortOrder, 0),
		IsVisible:   derefBool(in.IsVisible, true),
	}
	return c.store.CreateCategory(ctx, cat)
}

// UpdateCategory applies a partial update to a category.
func (c *CatalogService) UpdateCategory(ctx context.Context, id int64, patch map[string]json.RawMessage) (*models.Category, error) {
	cat, err := c.GetCategory(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := applyCategoryPatch(cat, patch); err != nil {
		return nil, err
	}
	return c.store.UpdateCategory(ctx, cat)
}

// DeleteCategory removes a category and its bookmarks.
func (c *CatalogService) DeleteCategory(ctx context.Context, id int64) error {
	if err := c.store.DeleteCategory(ctx, id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return NotFound("分类不存在")
		}
		return err
	}
	return nil
}

// ReorderCategories applies category ordering.
func (c *CatalogService) ReorderCategories(ctx context.Context, items []models.OrderItem) (int, error) {
	if len(items) == 0 {
		return 0, Validation("validation_failed", "排序列表不能为空",
			FieldError{Field: "items", Issue: "至少需要一项"})
	}
	return c.store.ReorderCategories(ctx, items)
}

func applyCategoryPatch(cat *models.Category, patch map[string]json.RawMessage) error {
	for key, raw := range patch {
		switch key {
		case "name":
			var v string
			if err := json.Unmarshal(raw, &v); err != nil {
				return fieldErr("name", "格式错误")
			}
			name, err := validateName(v, maxCategoryNameLen)
			if err != nil {
				return err
			}
			cat.Name = name
		case "description":
			var v string
			if err := json.Unmarshal(raw, &v); err != nil {
				return fieldErr("description", "格式错误")
			}
			desc, err := validateOptionalText(v, maxDescriptionLen, "description")
			if err != nil {
				return err
			}
			cat.Description = desc
		case "icon":
			var v string
			if err := json.Unmarshal(raw, &v); err != nil {
				return fieldErr("icon", "格式错误")
			}
			icon, err := validateCategoryIcon(v)
			if err != nil {
				return err
			}
			cat.Icon = icon
		case "color":
			var v string
			if err := json.Unmarshal(raw, &v); err != nil {
				return fieldErr("color", "格式错误")
			}
			color, err := validateColor(v)
			if err != nil {
				return err
			}
			cat.Color = color
		case "sort_order":
			var v int
			if err := json.Unmarshal(raw, &v); err != nil {
				return fieldErr("sort_order", "格式错误")
			}
			cat.SortOrder = v
		case "is_visible":
			var v bool
			if err := json.Unmarshal(raw, &v); err != nil {
				return fieldErr("is_visible", "格式错误")
			}
			cat.IsVisible = v
		default:
			return fieldErr(key, "不支持的字段")
		}
	}
	return nil
}

// --- Bookmarks ---

// ListBookmarks returns paginated bookmarks for the admin view.
func (c *CatalogService) ListBookmarks(ctx context.Context, categoryID *int64, query string, isVisible *bool, page, pageSize int) ([]models.Bookmark, int, error) {
	page, pageSize, offset := paginate(page, pageSize)
	items, total, err := c.store.ListBookmarks(ctx, store.BookmarkFilter{
		CategoryID: categoryID,
		Query:      query,
		IsVisible:  isVisible,
		Limit:      pageSize,
		Offset:     offset,
	})
	_ = page
	return items, total, err
}

// GetBookmark returns a bookmark by ID.
func (c *CatalogService) GetBookmark(ctx context.Context, id int64) (*models.Bookmark, error) {
	b, err := c.store.GetBookmark(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, NotFound("导航项不存在")
		}
		return nil, err
	}
	return b, nil
}

// CreateBookmark validates and inserts a bookmark.
func (c *CatalogService) CreateBookmark(ctx context.Context, in BookmarkInput) (*models.Bookmark, error) {
	if in.CategoryID == nil {
		return nil, fieldErr("category_id", "必填")
	}
	ok, err := c.store.CategoryExists(ctx, *in.CategoryID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fieldErr("category_id", "分类不存在")
	}

	title, err := validateName(in.Title, maxTitleLen)
	if err != nil {
		return nil, fieldErr("title", "长度需为 1-128 个字符")
	}
	url, err := validateURL(in.URL)
	if err != nil {
		return nil, err
	}
	description, err := validateOptionalText(in.Description, maxDescriptionLen, "description")
	if err != nil {
		return nil, err
	}
	keywords, err := validateOptionalText(in.Keywords, maxKeywordsLen, "keywords")
	if err != nil {
		return nil, err
	}
	iconType, iconValue, err := validateIcon(in.IconType, in.IconValue)
	if err != nil {
		return nil, err
	}
	tags, err := validateOptionalText(in.Tags, maxTagsLen, "tags")
	if err != nil {
		return nil, err
	}
	color, err := validateColor(in.Color)
	if err != nil {
		return nil, err
	}

	b := &models.Bookmark{
		CategoryID:   *in.CategoryID,
		Title:        title,
		URL:          url,
		Description:  description,
		IconType:     iconType,
		IconValue:    iconValue,
		Keywords:     keywords,
		Tags:         tags,
		Color:        color,
		SortOrder:    derefInt(in.SortOrder, 0),
		IsVisible:    derefBool(in.IsVisible, true),
		OpenInNewTab: derefBool(in.OpenInNewTab, true),
	}
	return c.store.CreateBookmark(ctx, b)
}

// UpdateBookmark applies a partial update to a bookmark.
func (c *CatalogService) UpdateBookmark(ctx context.Context, id int64, patch map[string]json.RawMessage) (*models.Bookmark, error) {
	b, err := c.GetBookmark(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := applyBookmarkPatch(b, patch); err != nil {
		return nil, err
	}
	if ok, err := c.store.CategoryExists(ctx, b.CategoryID); err != nil {
		return nil, err
	} else if !ok {
		return nil, fieldErr("category_id", "分类不存在")
	}
	return c.store.UpdateBookmark(ctx, b)
}

// DeleteBookmark removes a bookmark.
func (c *CatalogService) DeleteBookmark(ctx context.Context, id int64) error {
	if err := c.store.DeleteBookmark(ctx, id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return NotFound("导航项不存在")
		}
		return err
	}
	return nil
}

// ReorderBookmarks applies bookmark ordering.
func (c *CatalogService) ReorderBookmarks(ctx context.Context, items []models.OrderItem) (int, error) {
	if len(items) == 0 {
		return 0, Validation("validation_failed", "排序列表不能为空",
			FieldError{Field: "items", Issue: "至少需要一项"})
	}
	return c.store.ReorderBookmarks(ctx, items)
}

// MoveBookmark moves a bookmark to another category.
func (c *CatalogService) MoveBookmark(ctx context.Context, id, categoryID int64, sortOrder *int) (*models.Bookmark, error) {
	if _, err := c.GetBookmark(ctx, id); err != nil {
		return nil, err
	}
	ok, err := c.store.CategoryExists(ctx, categoryID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fieldErr("category_id", "分类不存在")
	}
	return c.store.MoveBookmark(ctx, id, categoryID, sortOrder)
}

func applyBookmarkPatch(b *models.Bookmark, patch map[string]json.RawMessage) error {
	for key, raw := range patch {
		switch key {
		case "category_id":
			var v int64
			if err := json.Unmarshal(raw, &v); err != nil {
				return fieldErr("category_id", "格式错误")
			}
			b.CategoryID = v
		case "title":
			var v string
			if err := json.Unmarshal(raw, &v); err != nil {
				return fieldErr("title", "格式错误")
			}
			title, err := validateName(v, maxTitleLen)
			if err != nil {
				return fieldErr("title", "长度需为 1-128 个字符")
			}
			b.Title = title
		case "url":
			var v string
			if err := json.Unmarshal(raw, &v); err != nil {
				return fieldErr("url", "格式错误")
			}
			u, err := validateURL(v)
			if err != nil {
				return err
			}
			b.URL = u
		case "description":
			var v string
			if err := json.Unmarshal(raw, &v); err != nil {
				return fieldErr("description", "格式错误")
			}
			text, err := validateOptionalText(v, maxDescriptionLen, "description")
			if err != nil {
				return err
			}
			b.Description = text
		case "keywords":
			var v string
			if err := json.Unmarshal(raw, &v); err != nil {
				return fieldErr("keywords", "格式错误")
			}
			text, err := validateOptionalText(v, maxKeywordsLen, "keywords")
			if err != nil {
				return err
			}
			b.Keywords = text
		case "tags":
			var v string
			if err := json.Unmarshal(raw, &v); err != nil {
				return fieldErr("tags", "格式错误")
			}
			text, err := validateOptionalText(v, maxTagsLen, "tags")
			if err != nil {
				return err
			}
			b.Tags = text
		case "color":
			var v string
			if err := json.Unmarshal(raw, &v); err != nil {
				return fieldErr("color", "格式错误")
			}
			color, err := validateColor(v)
			if err != nil {
				return err
			}
			b.Color = color
		case "icon_type", "icon_value":
			// Applied together after the loop to keep them consistent.
			continue
		case "sort_order":
			var v int
			if err := json.Unmarshal(raw, &v); err != nil {
				return fieldErr("sort_order", "格式错误")
			}
			b.SortOrder = v
		case "is_visible":
			var v bool
			if err := json.Unmarshal(raw, &v); err != nil {
				return fieldErr("is_visible", "格式错误")
			}
			b.IsVisible = v
		case "open_in_new_tab":
			var v bool
			if err := json.Unmarshal(raw, &v); err != nil {
				return fieldErr("open_in_new_tab", "格式错误")
			}
			b.OpenInNewTab = v
		default:
			return fieldErr(key, "不支持的字段")
		}
	}

	if _, ok := patch["icon_type"]; ok {
		iconType := b.IconType
		if err := json.Unmarshal(patch["icon_type"], &iconType); err != nil {
			return fieldErr("icon_type", "格式错误")
		}
		iconValue := b.IconValue
		if raw, ok := patch["icon_value"]; ok {
			if err := json.Unmarshal(raw, &iconValue); err != nil {
				return fieldErr("icon_value", "格式错误")
			}
		}
		t, v, err := validateIcon(iconType, iconValue)
		if err != nil {
			return err
		}
		b.IconType, b.IconValue = t, v
	} else if raw, ok := patch["icon_value"]; ok {
		iconValue := b.IconValue
		if err := json.Unmarshal(raw, &iconValue); err != nil {
			return fieldErr("icon_value", "格式错误")
		}
		t, v, err := validateIcon(b.IconType, iconValue)
		if err != nil {
			return err
		}
		b.IconType, b.IconValue = t, v
	}
	return nil
}

// --- Public ---

// Navigation returns the public home-page payload.
func (c *CatalogService) Navigation(ctx context.Context) (*models.Navigation, error) {
	categories, err := c.store.ListNavigation(ctx)
	if err != nil {
		return nil, err
	}
	settings, err := c.settings.Public(ctx)
	if err != nil {
		return nil, err
	}
	return &models.Navigation{Categories: categories, Settings: settings}, nil
}

// PublicBookmarksByCategory returns visible bookmarks of a visible category.
func (c *CatalogService) PublicBookmarksByCategory(ctx context.Context, categoryID int64) ([]models.Bookmark, error) {
	cat, err := c.store.GetCategory(ctx, categoryID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, NotFound("分类不存在")
		}
		return nil, err
	}
	if !cat.IsVisible {
		return nil, NotFound("分类不存在")
	}
	items, _, err := c.store.ListBookmarks(ctx, store.BookmarkFilter{
		CategoryID:  &categoryID,
		VisibleOnly: true,
	})
	return items, err
}

// Search performs a public search over visible bookmarks.
func (c *CatalogService) Search(ctx context.Context, query string, page, pageSize int) ([]models.Bookmark, int, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, 0, Validation("validation_failed", "搜索关键词不能为空",
			FieldError{Field: "q", Issue: "必填"})
	}
	_, pageSize, offset := paginate(page, pageSize)
	items, total, err := c.store.ListBookmarks(ctx, store.BookmarkFilter{
		Query:                 query,
		VisibleOnly:           true,
		VisibleCategoriesOnly: true,
		Limit:                 pageSize,
		Offset:                offset,
	})
	return items, total, err
}

// --- validation helpers ---

func validateName(value string, max int) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", fieldErr("name", "不能为空")
	}
	if len([]rune(v)) > max {
		return "", fieldErr("name", fmt.Sprintf("最多 %d 个字符", max))
	}
	return v, nil
}

func validateOptionalText(value string, max int, field string) (string, error) {
	v := strings.TrimSpace(value)
	if len([]rune(v)) > max {
		return "", fieldErr(field, fmt.Sprintf("最多 %d 个字符", max))
	}
	return v, nil
}

func validateURL(raw string) (string, error) {
	v := security.NormalizeURL(raw)
	if err := security.ValidateHTTPURL(v); err != nil {
		return "", New(KindUnsupported, "invalid_url_scheme", "URL 非法：仅允许 http/https",
			FieldError{Field: "url", Issue: err.Error()})
	}
	return v, nil
}

func validateIcon(iconType, iconValue string) (string, string, error) {
	if iconType == "" {
		iconType = "none"
	}
	if !allowedIconTypes[iconType] {
		return "", "", fieldErr("icon_type", "仅允许 none/url/upload/auto/favicon/simple/text")
	}
	iconValue = strings.TrimSpace(iconValue)
	if len(iconValue) > maxIconValueLen {
		return "", "", fieldErr("icon_value", fmt.Sprintf("最多 %d 个字符", maxIconValueLen))
	}

	switch iconType {
	case "none", "auto", "favicon":
		// These types carry no stored value.
		iconValue = ""
	case "url":
		if iconValue == "" {
			return "", "", fieldErr("icon_value", "图标 URL 不能为空")
		}
		if err := security.ValidateHTTPURL(iconValue); err != nil {
			return "", "", fieldErr("icon_value", "图标 URL 非法（仅允许 http/https）")
		}
	case "upload":
		if iconValue == "" || !strings.HasPrefix(iconValue, "/") {
			return "", "", fieldErr("icon_value", "上传图标必须是站内路径")
		}
	case "simple":
		if !IsSimpleIconSlug(iconValue) {
			return "", "", fieldErr("icon_value", "不是可用的 Simple Icons 图标标识")
		}
		iconValue = strings.ToLower(iconValue)
	case "text":
		if iconValue == "" {
			return "", "", fieldErr("icon_value", "文字图标不能为空")
		}
		if len([]rune(iconValue)) > maxIconTextRunes {
			return "", "", fieldErr("icon_value", fmt.Sprintf("文字图标最多 %d 个字符", maxIconTextRunes))
		}
	}
	return iconType, iconValue, nil
}

// validateCategoryIcon accepts an emoji or a very short text glyph.
func validateCategoryIcon(value string) (string, error) {
	v := strings.TrimSpace(value)
	if len([]rune(v)) > maxCategoryIconLen {
		return "", fieldErr("icon", fmt.Sprintf("最多 %d 个字符", maxCategoryIconLen))
	}
	return v, nil
}

var hexColorPattern = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

// validateColor accepts an empty value or a #RGB / #RRGGBB hex colour.
func validateColor(value string) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", nil
	}
	if !hexColorPattern.MatchString(v) {
		return "", fieldErr("color", "颜色需为 #RGB 或 #RRGGBB")
	}
	return v, nil
}

func fieldErr(field, issue string) *Error {
	return Validation("validation_failed", "请求参数校验失败", FieldError{Field: field, Issue: issue})
}

func derefInt(v *int, fallback int) int {
	if v == nil {
		return fallback
	}
	return *v
}

func derefBool(v *bool, fallback bool) bool {
	if v == nil {
		return fallback
	}
	return *v
}

func paginate(page, pageSize int) (int, int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize, (page - 1) * pageSize
}
