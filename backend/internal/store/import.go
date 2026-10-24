package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/models"
)

// Import conflict strategies.
const (
	ImportSkip      = "skip"
	ImportOverwrite = "overwrite"
	ImportRename    = "rename"
)

// ImportData applies an export bundle inside a single transaction.
//
// Existing data is never overwritten unless strategy is ImportOverwrite.
// Categories are matched by name; bookmarks are matched by URL within the
// resolved target category.
func (s *Store) ImportData(ctx context.Context, cats []models.Category, bookmarks []models.Bookmark, settings map[string]string, strategy string) (models.ImportResult, error) {
	result := models.ImportResult{OK: true}
	if strategy == "" {
		strategy = ImportSkip
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("begin import: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	now := models.NowUTC()

	existingCats, err := categoryNameIndex(ctx, tx)
	if err != nil {
		return result, err
	}

	// Map bundle category ID -> actual category ID in the database.
	catIDMap := make(map[int64]int64, len(cats))
	for _, c := range cats {
		name := strings.TrimSpace(c.Name)
		if name == "" {
			continue
		}
		if existingID, found := existingCats[strings.ToLower(name)]; found {
			switch strategy {
			case ImportOverwrite:
				if _, err := tx.ExecContext(ctx,
					`UPDATE categories SET description = ?, is_visible = ?, updated_at = ? WHERE id = ?`,
					c.Description, boolToInt(c.IsVisible), now, existingID,
				); err != nil {
					return result, fmt.Errorf("overwrite category %q: %w", name, err)
				}
				result.CategoriesImported++
				catIDMap[c.ID] = existingID
			case ImportRename:
				newID, err := insertCategory(ctx, tx, name+" (导入)", c.Description, c.IsVisible, now)
				if err != nil {
					return result, err
				}
				result.CategoriesImported++
				catIDMap[c.ID] = newID
			default: // skip
				result.Skipped++
				catIDMap[c.ID] = existingID
			}
			continue
		}
		newID, err := insertCategory(ctx, tx, name, c.Description, c.IsVisible, now)
		if err != nil {
			return result, err
		}
		result.CategoriesImported++
		catIDMap[c.ID] = newID
	}

	for _, b := range bookmarks {
		targetCat, ok := catIDMap[b.CategoryID]
		if !ok {
			// Bookmark references a category that was not part of the bundle
			// (or was skipped without an ID): fall back to its original ID if
			// that category exists.
			exists, err := categoryExistsTx(ctx, tx, b.CategoryID)
			if err != nil {
				return result, err
			}
			if !exists {
				result.Skipped++
				continue
			}
			targetCat = b.CategoryID
		}

		existingID, err := bookmarkIDByURL(ctx, tx, targetCat, b.URL)
		if err != nil {
			return result, err
		}
		if existingID != 0 {
			switch strategy {
			case ImportOverwrite:
				if _, err := tx.ExecContext(ctx,
					`UPDATE bookmarks SET title = ?, description = ?, icon_type = ?, icon_value = ?, keywords = ?, is_visible = ?, updated_at = ?
                     WHERE id = ?`,
					b.Title, b.Description, b.IconType, b.IconValue, b.Keywords, boolToInt(b.IsVisible), now, existingID,
				); err != nil {
					return result, fmt.Errorf("overwrite bookmark %q: %w", b.Title, err)
				}
				result.BookmarksImported++
			case ImportRename:
				if err := insertBookmark(ctx, tx, targetCat, b.Title+" (导入)", b, now); err != nil {
					return result, err
				}
				result.BookmarksImported++
			default:
				result.Skipped++
			}
			continue
		}
		if err := insertBookmark(ctx, tx, targetCat, b.Title, b, now); err != nil {
			return result, err
		}
		result.BookmarksImported++
	}

	for key, value := range settings {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO site_settings (key, value_json, updated_at) VALUES (?, ?, ?)
             ON CONFLICT(key) DO UPDATE SET value_json = excluded.value_json, updated_at = excluded.updated_at`,
			key, value, now,
		); err != nil {
			return result, fmt.Errorf("import setting %q: %w", key, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("commit import: %w", err)
	}
	return result, nil
}

// PreviewImport counts new vs conflicting entries without writing.
func (s *Store) PreviewImport(ctx context.Context, cats []models.Category, bookmarks []models.Bookmark) (int, int, int, int, error) {
	existingCats, err := categoryNameIndex(ctx, s.db)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	var catNew, catConflict int
	for _, c := range cats {
		if _, found := existingCats[strings.ToLower(strings.TrimSpace(c.Name))]; found {
			catConflict++
		} else {
			catNew++
		}
	}

	var bmNew, bmConflict int
	for _, b := range bookmarks {
		id, err := bookmarkIDByURL(ctx, s.db, b.CategoryID, b.URL)
		if err != nil {
			return 0, 0, 0, 0, err
		}
		if id != 0 {
			bmConflict++
		} else {
			bmNew++
		}
	}
	return catNew, catConflict, bmNew, bmConflict, nil
}

type querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func categoryNameIndex(ctx context.Context, q querier) (map[string]int64, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, name FROM categories`)
	if err != nil {
		return nil, fmt.Errorf("index categories: %w", err)
	}
	defer rows.Close()

	out := map[string]int64{}
	for rows.Next() {
		var (
			id   int64
			name string
		)
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		out[strings.ToLower(strings.TrimSpace(name))] = id
	}
	return out, rows.Err()
}

func insertCategory(ctx context.Context, tx *sql.Tx, name, description string, isVisible bool, now string) (int64, error) {
	res, err := tx.ExecContext(ctx,
		`INSERT INTO categories (name, description, sort_order, is_visible, created_at, updated_at)
         VALUES (?, ?, 0, ?, ?, ?)`,
		name, description, boolToInt(isVisible), now, now,
	)
	if err != nil {
		return 0, fmt.Errorf("insert category %q: %w", name, err)
	}
	return res.LastInsertId()
}

func categoryExistsTx(ctx context.Context, tx *sql.Tx, id int64) (bool, error) {
	var n int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM categories WHERE id = ?`, id).Scan(&n); err != nil {
		return false, fmt.Errorf("category exists: %w", err)
	}
	return n > 0, nil
}

func bookmarkIDByURL(ctx context.Context, q querier, categoryID int64, url string) (int64, error) {
	var id int64
	err := q.QueryRowContext(ctx,
		`SELECT id FROM bookmarks WHERE category_id = ? AND url = ? LIMIT 1`, categoryID, url).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("find bookmark by url: %w", err)
	}
	return id, nil
}

func insertBookmark(ctx context.Context, tx *sql.Tx, categoryID int64, title string, b models.Bookmark, now string) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO bookmarks
            (category_id, title, url, description, icon_type, icon_value, keywords, sort_order, is_visible, open_in_new_tab, created_at, updated_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		categoryID, title, b.URL, b.Description, b.IconType, b.IconValue, b.Keywords,
		b.SortOrder, boolToInt(b.IsVisible), boolToInt(b.OpenInNewTab), now, now,
	)
	if err != nil {
		return fmt.Errorf("insert bookmark %q: %w", title, err)
	}
	return nil
}
