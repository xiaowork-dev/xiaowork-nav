package store

import (
	"context"
	"fmt"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/models"
)

// ListSettings returns all site settings as key -> raw JSON value.
func (s *Store) ListSettings(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key, value_json FROM site_settings`)
	if err != nil {
		return nil, fmt.Errorf("list settings: %w", err)
	}
	defer rows.Close()

	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, fmt.Errorf("scan setting: %w", err)
		}
		out[k] = v
	}
	return out, rows.Err()
}

// UpsertSettings writes multiple settings in a single transaction.
func (s *Store) UpsertSettings(ctx context.Context, values map[string]string) error {
	if len(values) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin settings tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	now := models.NowUTC()
	for k, v := range values {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO site_settings (key, value_json, updated_at) VALUES (?, ?, ?)
             ON CONFLICT(key) DO UPDATE SET value_json = excluded.value_json, updated_at = excluded.updated_at`,
			k, v, now,
		); err != nil {
			return fmt.Errorf("upsert setting %q: %w", k, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit settings: %w", err)
	}
	return nil
}
