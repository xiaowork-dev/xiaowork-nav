package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/models"
)

// ErrUsernameTaken is returned when creating a user with an existing username.
var ErrUsernameTaken = errors.New("store: username taken")

// CountUsers returns the number of administrator accounts.
func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return n, nil
}

// CreateUser inserts a user and returns it with its assigned ID.
func (s *Store) CreateUser(ctx context.Context, username, passwordHash string) (*models.User, error) {
	now := models.NowUTC()
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO users (username, password_hash, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		username, passwordHash, now, now,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrUsernameTaken
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("user id: %w", err)
	}
	return &models.User{ID: id, Username: username, PasswordHash: passwordHash, CreatedAt: now, UpdatedAt: now}, nil
}

// GetUserByUsername looks up a user by username.
func (s *Store) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	return s.scanUser(s.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, created_at, updated_at FROM users WHERE username = ?`, username))
}

// GetUserByID looks up a user by primary key.
func (s *Store) GetUserByID(ctx context.Context, id int64) (*models.User, error) {
	return s.scanUser(s.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, created_at, updated_at FROM users WHERE id = ?`, id))
}

func (s *Store) scanUser(row *sql.Row) (*models.User, error) {
	var u models.User
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}
	return &u, nil
}

// UpdateUserPassword replaces a user's password hash.
func (s *Store) UpdateUserPassword(ctx context.Context, id int64, passwordHash string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`,
		passwordHash, models.NowUTC(), id,
	)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
