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
    b.keywords, b.sort_order, b.is_visible, b.open_in_new_tab, b.created_at, b.updated_at`

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
            (category_id, title, url, description, icon_type, icon_value, keywords, sort_order, is_visible, open_in_new_tab, created_at, updated_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		b.CategoryID, b.Title, b.URL, b.Description, b.IconType, b.IconValue, b.Keywords,
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
            keywords = ?, sort_order = ?, is_visible = ?, open_in_new_tab = ?, updated_at = ?
         WHERE id = ?`,
		b.CategoryID, b.Title, b.URL, b.Description, b.IconType, b.IconValue, b.Keywords,
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
		b                     models.Bookmark
		desc, iconVal, kw     sql.NullString
		visible, openNewTab   int
	)
	err := sc.Scan(&b.ID, &b.CategoryID, &b.Title, &b.URL, &desc, &b.IconType, &iconVal, &kw,
		&b.SortOrder, &visible, &openNewTab, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return b, err
	}
	b.Description = desc.String
	b.IconValue = iconVal.String
	b.Keywords = kw.String
	b.IsVisible = intToBool(visible)
	b.OpenInNewTab = intToBool(openNewTab)
	return b, nil
}
