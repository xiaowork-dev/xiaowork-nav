package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/models"
)

// CreateUpload records an uploaded file's metadata.
func (s *Store) CreateUpload(ctx context.Context, u *models.Upload) error {
	now := models.NowUTC()
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO uploads (filename, original_name, mime, size, created_at) VALUES (?, ?, ?, ?, ?)`,
		u.Filename, u.OriginalName, u.Mime, u.Size, now,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("upload %q already exists", u.Filename)
		}
		return fmt.Errorf("insert upload: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("upload id: %w", err)
	}
	u.ID = id
	u.CreatedAt = now
	return nil
}

// GetUploadByFilename returns upload metadata by stored filename.
func (s *Store) GetUploadByFilename(ctx context.Context, filename string) (*models.Upload, error) {
	var u models.Upload
	err := s.db.QueryRowContext(ctx,
		`SELECT id, filename, original_name, mime, size, created_at FROM uploads WHERE filename = ?`, filename,
	).Scan(&u.ID, &u.Filename, &u.OriginalName, &u.Mime, &u.Size, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan upload: %w", err)
	}
	return &u, nil
}

// DeleteUpload removes upload metadata.
func (s *Store) DeleteUpload(ctx context.Context, filename string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM uploads WHERE filename = ?`, filename); err != nil {
		return fmt.Errorf("delete upload: %w", err)
	}
	return nil
}

// ListUploads returns all upload metadata.
func (s *Store) ListUploads(ctx context.Context) ([]models.Upload, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, filename, original_name, mime, size, created_at FROM uploads ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list uploads: %w", err)
	}
	defer rows.Close()

	out := make([]models.Upload, 0)
	for rows.Next() {
		var u models.Upload
		if err := rows.Scan(&u.ID, &u.Filename, &u.OriginalName, &u.Mime, &u.Size, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan upload: %w", err)
		}
		out = append(out, u)
	}
	return out, rows.Err()
}
