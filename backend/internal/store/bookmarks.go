package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/models"
)

const bookmarkColumns = `b.id, b.category_id, b.title, b.url, b.description, b.icon_type, b.icon_value,
    b.keywords, b.tags, b.color, b.sort_order, b.is_visible, b.open_in_new_tab,
    b.icon_source, b.icon_status, b.icon_error, b.icon_checked_at,
    b.created_at, b.updated_at`

// BookmarkFilter selects and paginates bookmarks.
type BookmarkFilter struct {
	CategoryID *int64
	Query      string
	// VisibleOnly restricts results to is_visible = 1.
	VisibleOnly bool
	// IsVisible applies an explicit admin filter when non-nil.
	IsVisible *bool
	// VisibleCategoriesOnly restricts results to bookmarks in visible categories.
	VisibleCategoriesOnly bool
	Limit                 int
	Offset                int
}

func (f BookmarkFilter) where() (string, []any) {
	var clauses []string
	var args []any

	if f.CategoryID != nil {
		clauses = append(clauses, "b.category_id = ?")
		args = append(args, *f.CategoryID)
	}
	if f.VisibleOnly {
		clauses = append(clauses, "b.is_visible = 1")
	}
	if f.IsVisible != nil {
		clauses = append(clauses, "b.is_visible = ?")
		args = append(args, boolToInt(*f.IsVisible))
	}
	if f.VisibleCategoriesOnly {
		clauses = append(clauses, "c.is_visible = 1")
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		pattern := "%" + escapeLike(q) + "%"
		clauses = append(clauses,
			`(b.title LIKE ? ESCAPE '\' OR b.description LIKE ? ESCAPE '\' OR b.keywords LIKE ? ESCAPE '\')`)
		args = append(args, pattern, pattern, pattern)
	}
	if len(clauses) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

// ListBookmarks returns matching bookmarks and the total count.
func (s *Store) ListBookmarks(ctx context.Context, f BookmarkFilter) ([]models.Bookmark, int, error) {
	where, args := f.where()

	var total int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM bookmarks b JOIN categories c ON c.id = b.category_id`+where, args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count bookmarks: %w", err)
	}

	query := `SELECT ` + bookmarkColumns + ` FROM bookmarks b JOIN categories c ON c.id = b.category_id` + where +
		` ORDER BY b.category_id, b.sort_order, b.id`
	queryArgs := append([]any{}, args...)
	if f.Limit > 0 {
		query += " LIMIT ? OFFSET ?"
		queryArgs = append(queryArgs, f.Limit, f.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list bookmarks: %w", err)
	}
	defer rows.Close()

	items := make([]models.Bookmark, 0)
	for rows.Next() {
		b, err := scanBookmark(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, b)
	}
	return items, total, rows.Err()
}

// GetBookmark returns a bookmark by ID.
func (s *Store) GetBookmark(ctx context.Context, id int64) (*models.Bookmark, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+bookmarkColumns+` FROM bookmarks b WHERE b.id = ?`, id)
	b, err := scanBookmark(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// CreateBookmark inserts a bookmark and returns it with its ID.
func (s *Store) CreateBookmark(ctx context.Context, b *models.Bookmark) (*models.Bookmark, error) {
	now := models.NowUTC()
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO bookmarks
            (category_id, title, url, description, icon_type, icon_value, keywords, tags, color, sort_order, is_visible, open_in_new_tab, created_at, updated_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		b.CategoryID, b.Title, b.URL, b.Description, b.IconType, b.IconValue, b.Keywords, b.Tags, b.Color,
		b.SortOrder, boolToInt(b.IsVisible), boolToInt(b.OpenInNewTab), now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert bookmark: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("bookmark id: %w", err)
	}
	return s.GetBookmark(ctx, id)
}

// UpdateBookmark replaces the editable fields of a bookmark.
func (s *Store) UpdateBookmark(ctx context.Context, b *models.Bookmark) (*models.Bookmark, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE bookmarks SET category_id = ?, title = ?, url = ?, description = ?, icon_type = ?, icon_value = ?,
            keywords = ?, tags = ?, color = ?, sort_order = ?, is_visible = ?, open_in_new_tab = ?, updated_at = ?
         WHERE id = ?`,
		b.CategoryID, b.Title, b.URL, b.Description, b.IconType, b.IconValue, b.Keywords, b.Tags, b.Color,
		b.SortOrder, boolToInt(b.IsVisible), boolToInt(b.OpenInNewTab), models.NowUTC(), b.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("update bookmark: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrNotFound
	}
	return s.GetBookmark(ctx, b.ID)
}

// DeleteBookmark removes a bookmark.
func (s *Store) DeleteBookmark(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM bookmarks WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete bookmark: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ReorderBookmarks applies sort_order values in a single transaction.
func (s *Store) ReorderBookmarks(ctx context.Context, items []models.OrderItem) (int, error) {
	return s.reorder(ctx, `UPDATE bookmarks SET sort_order = ?, updated_at = ? WHERE id = ?`, items)
}

// MoveBookmark moves a bookmark to another category, optionally setting its order.
func (s *Store) MoveBookmark(ctx context.Context, id, categoryID int64, sortOrder *int) (*models.Bookmark, error) {
	var err error
	if sortOrder != nil {
		_, err = s.db.ExecContext(ctx,
			`UPDATE bookmarks SET category_id = ?, sort_order = ?, updated_at = ? WHERE id = ?`,
			categoryID, *sortOrder, models.NowUTC(), id)
	} else {
		_, err = s.db.ExecContext(ctx,
			`UPDATE bookmarks SET category_id = ?, updated_at = ? WHERE id = ?`,
			categoryID, models.NowUTC(), id)
	}
	if err != nil {
		return nil, fmt.Errorf("move bookmark: %w", err)
	}
	return s.GetBookmark(ctx, id)
}

// ListNavigation returns visible categories, each with its visible bookmarks.
func (s *Store) ListNavigation(ctx context.Context) ([]models.Category, error) {
	cats, err := s.ListCategories(ctx, false, true)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT `+bookmarkColumns+`
         FROM bookmarks b JOIN categories c ON c.id = b.category_id
         WHERE b.is_visible = 1 AND c.is_visible = 1
         ORDER BY b.category_id, b.sort_order, b.id`)
	if err != nil {
		return nil, fmt.Errorf("list navigation bookmarks: %w", err)
	}
	defer rows.Close()

	byCategory := map[int64][]models.Bookmark{}
	for rows.Next() {
		b, err := scanBookmark(rows)
		if err != nil {
			return nil, err
		}
		byCategory[b.CategoryID] = append(byCategory[b.CategoryID], b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range cats {
		if bs, ok := byCategory[cats[i].ID]; ok {
			cats[i].Bookmarks = bs
		} else {
			cats[i].Bookmarks = []models.Bookmark{}
		}
	}
	return cats, nil
}

func scanBookmark(sc scanner) (models.Bookmark, error) {
	var (
		b                                 models.Bookmark
		desc, iconVal, kw, tags, color    sql.NullString
		iconSrc, iconStat, iconErr, iconChk sql.NullString
		visible, openNewTab               int
	)
	err := sc.Scan(&b.ID, &b.CategoryID, &b.Title, &b.URL, &desc, &b.IconType, &iconVal, &kw, &tags, &color,
		&b.SortOrder, &visible, &openNewTab,
		&iconSrc, &iconStat, &iconErr, &iconChk,
		&b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return b, err
	}
	b.Description = desc.String
	b.IconValue = iconVal.String
	b.Keywords = kw.String
	b.Tags = tags.String
	b.Color = color.String
	b.IconSource = iconSrc.String
	b.IconStatus = iconStat.String
	b.IconError = iconErr.String
	b.IconCheckedAt = iconChk.String
	b.IsVisible = intToBool(visible)
	b.OpenInNewTab = intToBool(openNewTab)
	return b, nil
}

// IconBookkeeping is the automatic-icon state persisted on a bookmark.
type IconBookkeeping struct {
	IconType      string
	IconValue     string
	IconSource    string
	IconStatus    string
	IconError     string
	IconCheckedAt string
}

// SetBookmarkIcon writes only the icon bookkeeping columns, so a background
// fetch can never clobber a concurrent edit of the bookmark's other fields.
func (s *Store) SetBookmarkIcon(ctx context.Context, id int64, in IconBookkeeping) (*models.Bookmark, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE bookmarks SET icon_type = ?, icon_value = ?, icon_source = ?, icon_status = ?,
            icon_error = ?, icon_checked_at = ?, updated_at = ?
         WHERE id = ?`,
		in.IconType, in.IconValue, in.IconSource, in.IconStatus,
		in.IconError, in.IconCheckedAt, models.NowUTC(), id,
	)
	if err != nil {
		return nil, fmt.Errorf("set bookmark icon: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrNotFound
	}
	return s.GetBookmark(ctx, id)
}

// IconFillCandidate is a bookmark considered by the bulk icon job.
type IconFillCandidate struct {
	ID            int64
	URL           string
	IconType      string
	IconValue     string
	IconStatus    string
	IconCheckedAt string
}

// ListIconFillCandidates returns bookmarks eligible for automatic icon filling.
//
// Manual icon types (url/upload/simple/text) are never returned: a user-set
// icon must not be overwritten. When onlyMissing is true, bookmarks that were
// checked recently and succeeded are skipped as well.
func (s *Store) ListIconFillCandidates(ctx context.Context, onlyMissing bool, checkedAfter string, limit int) ([]IconFillCandidate, error) {
	query := `SELECT b.id, b.url, b.icon_type, b.icon_value, b.icon_status, b.icon_checked_at
        FROM bookmarks b
        WHERE b.icon_type NOT IN ('url', 'upload', 'simple', 'text')`
	args := []any{}
	if onlyMissing {
		query += ` AND (b.icon_status != 'ok' OR b.icon_checked_at = '' OR b.icon_checked_at < ?)`
		args = append(args, checkedAfter)
	}
	query += ` ORDER BY b.category_id, b.sort_order, b.id`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list icon candidates: %w", err)
	}
	defer rows.Close()

	out := make([]IconFillCandidate, 0)
	for rows.Next() {
		var c IconFillCandidate
		if err := rows.Scan(&c.ID, &c.URL, &c.IconType, &c.IconValue, &c.IconStatus, &c.IconCheckedAt); err != nil {
			return nil, fmt.Errorf("scan icon candidate: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// IconSummary counts bookmarks by icon state for the admin dashboard.
type IconSummary struct {
	Total     int `json:"total"`
	Manual    int `json:"manual"`
	Auto      int `json:"auto"`
	Favicon   int `json:"favicon"`
	Missing   int `json:"missing"`
	Failed    int `json:"failed"`
	Other     int `json:"other"`
}

// CountIconSummary aggregates the icon state of every bookmark.
func (s *Store) CountIconSummary(ctx context.Context) (IconSummary, error) {
	var out IconSummary
	rows, err := s.db.QueryContext(ctx, `SELECT icon_type, icon_status, COUNT(*) FROM bookmarks GROUP BY icon_type, icon_status`)
	if err != nil {
		return out, fmt.Errorf("count icon summary: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var iconType, status string
		var n int
		if err := rows.Scan(&iconType, &status, &n); err != nil {
			return out, fmt.Errorf("scan icon summary: %w", err)
		}
		out.Total += n
		switch iconType {
		case "url", "upload", "simple", "text":
			out.Manual += n
		case "auto":
			out.Auto += n
			if status == models.IconStatusFailed {
				out.Failed += n
			}
		case "favicon":
			out.Favicon += n
		case "none", "":
			out.Missing += n
		default:
			out.Other += n
		}
	}
	return out, rows.Err()
}

// CountBookmarksUsingUpload counts bookmarks whose icon_value points at the
// given stored upload URL. Used to avoid deleting a shared cached icon.
func (s *Store) CountBookmarksUsingUpload(ctx context.Context, iconValue string) (int, error) {
	var n int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM bookmarks WHERE icon_value = ?`, iconValue).Scan(&n); err != nil {
		return 0, fmt.Errorf("count upload references: %w", err)
	}
	return n, nil
}
