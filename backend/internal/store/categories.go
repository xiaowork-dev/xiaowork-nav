package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/models"
)

const categoryColumns = `c.id, c.name, c.description, c.sort_order, c.is_visible, c.created_at, c.updated_at`

// ListCategories returns categories ordered by sort_order.
//
// When includeHidden is false only visible categories are returned. When
// countVisibleOnly is true the bookmark_count only counts visible bookmarks
// (used by public endpoints so private entries are not leaked).
func (s *Store) ListCategories(ctx context.Context, includeHidden, countVisibleOnly bool) ([]models.Category, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+categoryColumns+`,
             (SELECT COUNT(*) FROM bookmarks b WHERE b.category_id = c.id AND (? = 0 OR b.is_visible = 1)) AS bookmark_count
         FROM categories c
         WHERE (? = 1 OR c.is_visible = 1)
         ORDER BY c.sort_order, c.id`,
		boolToInt(countVisibleOnly), boolToInt(includeHidden),
	)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	out := make([]models.Category, 0)
	for rows.Next() {
		c, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetCategory returns a category by ID.
func (s *Store) GetCategory(ctx context.Context, id int64) (*models.Category, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+categoryColumns+`,
             (SELECT COUNT(*) FROM bookmarks b WHERE b.category_id = c.id) AS bookmark_count
         FROM categories c WHERE c.id = ?`, id)
	c, err := scanCategory(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// CategoryExists reports whether a category ID exists.
func (s *Store) CategoryExists(ctx context.Context, id int64) (bool, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM categories WHERE id = ?`, id).Scan(&n); err != nil {
		return false, fmt.Errorf("category exists: %w", err)
	}
	return n > 0, nil
}

// CreateCategory inserts a category and returns it with its ID.
func (s *Store) CreateCategory(ctx context.Context, c *models.Category) (*models.Category, error) {
	now := models.NowUTC()
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO categories (name, description, sort_order, is_visible, created_at, updated_at)
         VALUES (?, ?, ?, ?, ?, ?)`,
		c.Name, c.Description, c.SortOrder, boolToInt(c.IsVisible), now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert category: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("category id: %w", err)
	}
	c.ID = id
	c.CreatedAt = now
	c.UpdatedAt = now
	return c, nil
}

// UpdateCategory replaces the editable fields of a category.
func (s *Store) UpdateCategory(ctx context.Context, c *models.Category) (*models.Category, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE categories SET name = ?, description = ?, sort_order = ?, is_visible = ?, updated_at = ?
         WHERE id = ?`,
		c.Name, c.Description, c.SortOrder, boolToInt(c.IsVisible), models.NowUTC(), c.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("update category: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrNotFound
	}
	return s.GetCategory(ctx, c.ID)
}

// DeleteCategory removes a category (bookmarks cascade).
func (s *Store) DeleteCategory(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM categories WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ReorderCategories applies sort_order values in a single transaction.
func (s *Store) ReorderCategories(ctx context.Context, items []models.OrderItem) (int, error) {
	return s.reorder(ctx, `UPDATE categories SET sort_order = ?, updated_at = ? WHERE id = ?`, items)
}

func (s *Store) reorder(ctx context.Context, query string, items []models.OrderItem) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin reorder: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("prepare reorder: %w", err)
	}
	defer stmt.Close()

	now := models.NowUTC()
	updated := 0
	for _, it := range items {
		res, err := stmt.ExecContext(ctx, it.SortOrder, now, it.ID)
		if err != nil {
			return 0, fmt.Errorf("reorder item %d: %w", it.ID, err)
		}
		if n, _ := res.RowsAffected(); n > 0 {
			updated++
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit reorder: %w", err)
	}
	return updated, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanCategory(sc scanner) (models.Category, error) {
	var (
		c       models.Category
		visible int
		count   int
		desc    sql.NullString
	)
	if err := sc.Scan(&c.ID, &c.Name, &desc, &c.SortOrder, &visible, &c.CreatedAt, &c.UpdatedAt, &count); err != nil {
		return c, err
	}
	c.Description = desc.String
	c.IsVisible = intToBool(visible)
	c.BookmarkCount = count
	return c, nil
}
