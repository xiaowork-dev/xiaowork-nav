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
	if u.Origin == "" {
		u.Origin = models.UploadOriginManual
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO uploads (filename, original_name, mime, size, source_url, origin, created_at)
         VALUES (?, ?, ?, ?, ?, ?, ?)`,
		u.Filename, u.OriginalName, u.Mime, u.Size, u.SourceURL, u.Origin, now,
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

const uploadColumns = `id, filename, original_name, mime, size, source_url, origin, created_at`

func scanUpload(sc scanner) (models.Upload, error) {
	var (
		u        models.Upload
		source   sql.NullString
		origin   sql.NullString
	)
	err := sc.Scan(&u.ID, &u.Filename, &u.OriginalName, &u.Mime, &u.Size, &source, &origin, &u.CreatedAt)
	if err != nil {
		return u, err
	}
	u.SourceURL = source.String
	u.Origin = origin.String
	return u, nil
}

// GetUploadByFilename returns upload metadata by stored filename.
func (s *Store) GetUploadByFilename(ctx context.Context, filename string) (*models.Upload, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+uploadColumns+` FROM uploads WHERE filename = ?`, filename)
	u, err := scanUpload(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan upload: %w", err)
	}
	return &u, nil
}

// FindUploadBySourceURL returns the newest auto-cached upload for a source URL.
//
// This is the icon cache lookup: when the same remote icon has already been
// downloaded it is reused instead of being fetched again.
func (s *Store) FindUploadBySourceURL(ctx context.Context, sourceURL string) (*models.Upload, error) {
	if sourceURL == "" {
		return nil, ErrNotFound
	}
	row := s.db.QueryRowContext(ctx,
		`SELECT `+uploadColumns+` FROM uploads WHERE source_url = ? ORDER BY id DESC LIMIT 1`, sourceURL)
	u, err := scanUpload(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan upload by source: %w", err)
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
		`SELECT `+uploadColumns+` FROM uploads ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list uploads: %w", err)
	}
	defer rows.Close()

	out := make([]models.Upload, 0)
	for rows.Next() {
		u, err := scanUpload(rows)
		if err != nil {
			return nil, fmt.Errorf("scan upload: %w", err)
		}
		out = append(out, u)
	}
	return out, rows.Err()
}
