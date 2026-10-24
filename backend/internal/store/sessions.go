package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/models"
)

// CreateSession inserts a session row.
func (s *Store) CreateSession(ctx context.Context, sess *models.Session) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, created_at, expires_at, last_seen_at, user_agent, ip)
         VALUES (?, ?, ?, ?, ?, ?, ?)`,
		sess.ID, sess.UserID, sess.CreatedAt, sess.ExpiresAt, sess.LastSeenAt, sess.UserAgent, sess.IP,
	)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

// GetSession returns a session by ID.
func (s *Store) GetSession(ctx context.Context, id string) (*models.Session, error) {
	var sess models.Session
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, created_at, expires_at, last_seen_at, user_agent, ip FROM sessions WHERE id = ?`, id,
	).Scan(&sess.ID, &sess.UserID, &sess.CreatedAt, &sess.ExpiresAt, &sess.LastSeenAt, &sess.UserAgent, &sess.IP)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan session: %w", err)
	}
	return &sess, nil
}

// TouchSession updates the sliding expiration of a session.
func (s *Store) TouchSession(ctx context.Context, id, lastSeenAt, expiresAt string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE sessions SET last_seen_at = ?, expires_at = ? WHERE id = ?`, lastSeenAt, expiresAt, id)
	if err != nil {
		return fmt.Errorf("touch session: %w", err)
	}
	return nil
}

// DeleteSession removes a single session (logout).
func (s *Store) DeleteSession(ctx context.Context, id string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// DeleteUserSessions removes all sessions of a user (password change).
func (s *Store) DeleteUserSessions(ctx context.Context, userID int64) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("delete user sessions: %w", err)
	}
	return nil
}

// DeleteExpiredSessions purges sessions that expired before now.
func (s *Store) DeleteExpiredSessions(ctx context.Context, now string) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= ?`, now)
	if err != nil {
		return 0, fmt.Errorf("delete expired sessions: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}
