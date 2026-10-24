package service

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/auth"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/config"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/models"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/store"
)

const (
	minUsernameLen = 3
	maxUsernameLen = 32
	minPasswordLen = 8
	maxPasswordLen = 128

	loginLimit  = 10
	loginWindow = 5 * time.Minute

	sessionRenewThreshold = time.Hour
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// AuthService implements administrator setup, login, sessions and password
// changes.
type AuthService struct {
	store   *store.Store
	cfg     *config.Config
	limiter *auth.Limiter
}

// NewAuthService builds an AuthService.
func NewAuthService(st *store.Store, cfg *config.Config) *AuthService {
	return &AuthService{
		store:   st,
		cfg:     cfg,
		limiter: auth.NewLimiter(loginLimit, loginWindow),
	}
}

// Initialized reports whether an administrator account exists.
func (s *AuthService) Initialized(ctx context.Context) (bool, error) {
	n, err := s.store.CountUsers(ctx)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// SessionTTL is the configured session lifetime.
func (s *AuthService) SessionTTL() time.Duration {
	return time.Duration(s.cfg.SessionTTLHours) * time.Hour
}

// Setup creates the first administrator. It fails if one already exists.
func (s *AuthService) Setup(ctx context.Context, username, password string) (*models.User, error) {
	initialized, err := s.Initialized(ctx)
	if err != nil {
		return nil, err
	}
	if initialized {
		return nil, Conflict("already_initialized", "系统已初始化，无法重复创建管理员")
	}
	if err := validateCredentials(username, password); err != nil {
		return nil, err
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}
	user, err := s.store.CreateUser(ctx, strings.TrimSpace(username), hash)
	if err != nil {
		if errors.Is(err, store.ErrUsernameTaken) {
			return nil, Conflict("username_taken", "用户名已存在")
		}
		return nil, err
	}
	return user, nil
}

// Login verifies credentials, applies rate limiting, and creates a session.
func (s *AuthService) Login(ctx context.Context, username, password, ip, userAgent string) (*models.User, *models.Session, error) {
	ipKey := "ip:" + ip
	userKey := "user:" + strings.ToLower(strings.TrimSpace(username))

	if !s.limiter.Allow(ipKey) || !s.limiter.Allow(userKey) {
		return nil, nil, New(KindRateLimited, "rate_limited", "登录尝试过于频繁，请稍后再试")
	}

	user, err := s.store.GetUserByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil, New(KindUnauthorized, "invalid_credentials", "用户名或密码错误")
		}
		return nil, nil, err
	}
	if !auth.VerifyPassword(user.PasswordHash, password) {
		return nil, nil, New(KindUnauthorized, "invalid_credentials", "用户名或密码错误")
	}

	session, err := s.newSession(ctx, user.ID, ip, userAgent)
	if err != nil {
		return nil, nil, err
	}

	s.limiter.Reset(ipKey)
	s.limiter.Reset(userKey)
	return user, session, nil
}

func (s *AuthService) newSession(ctx context.Context, userID int64, ip, userAgent string) (*models.Session, error) {
	id, err := auth.NewSessionID()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	session := &models.Session{
		ID:         id,
		UserID:     userID,
		CreatedAt:  models.FormatUTC(now),
		ExpiresAt:  models.FormatUTC(now.Add(s.SessionTTL())),
		LastSeenAt: models.FormatUTC(now),
		UserAgent:  truncate(userAgent, 255),
		IP:         truncate(ip, 64),
	}
	if err := s.store.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

// Authenticate resolves a session ID to a user, renewing the session when it
// has not been seen for a while.
func (s *AuthService) Authenticate(ctx context.Context, sessionID string) (*models.User, *models.Session, error) {
	if sessionID == "" {
		return nil, nil, Unauthorized("未登录")
	}
	session, err := s.store.GetSession(ctx, sessionID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil, Unauthorized("会话不存在或已失效")
		}
		return nil, nil, err
	}

	now := time.Now().UTC()
	expires, err := time.Parse(time.RFC3339, session.ExpiresAt)
	if err != nil || now.After(expires) {
		_ = s.store.DeleteSession(ctx, session.ID)
		return nil, nil, Unauthorized("会话已过期")
	}

	lastSeen, _ := time.Parse(time.RFC3339, session.LastSeenAt)
	if now.Sub(lastSeen) > sessionRenewThreshold {
		newExpiry := models.FormatUTC(now.Add(s.SessionTTL()))
		if err := s.store.TouchSession(ctx, session.ID, models.FormatUTC(now), newExpiry); err == nil {
			session.LastSeenAt = models.FormatUTC(now)
			session.ExpiresAt = newExpiry
		}
	}

	user, err := s.store.GetUserByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			_ = s.store.DeleteSession(ctx, session.ID)
			return nil, nil, Unauthorized("用户不存在")
		}
		return nil, nil, err
	}
	return user, session, nil
}

// Logout deletes a session.
func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	return s.store.DeleteSession(ctx, sessionID)
}

// ChangePassword verifies the current password, updates it, and invalidates
// all existing sessions.
func (s *AuthService) ChangePassword(ctx context.Context, userID int64, current, next string) error {
	if err := validatePassword(next); err != nil {
		return err
	}
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if !auth.VerifyPassword(user.PasswordHash, current) {
		return New(KindUnauthorized, "invalid_credentials", "当前密码错误")
	}
	hash, err := auth.HashPassword(next)
	if err != nil {
		return err
	}
	if err := s.store.UpdateUserPassword(ctx, userID, hash); err != nil {
		return err
	}
	return s.store.DeleteUserSessions(ctx, userID)
}

// CleanupExpiredSessions removes expired sessions.
func (s *AuthService) CleanupExpiredSessions(ctx context.Context) (int64, error) {
	return s.store.DeleteExpiredSessions(ctx, models.NowUTC())
}

// CleanupLimiter trims the rate limiter's memory.
func (s *AuthService) CleanupLimiter() {
	s.limiter.Cleanup()
}

func validateCredentials(username, password string) error {
	username = strings.TrimSpace(username)
	if l := len([]rune(username)); l < minUsernameLen || l > maxUsernameLen {
		return Validation("validation_failed", "用户名长度不合法",
			FieldError{Field: "username", Issue: "长度需为 3-32 个字符"})
	}
	if !usernamePattern.MatchString(username) {
		return Validation("validation_failed", "用户名包含非法字符",
			FieldError{Field: "username", Issue: "仅允许字母、数字、下划线和连字符"})
	}
	return validatePassword(password)
}

func validatePassword(password string) error {
	if len([]rune(password)) < minPasswordLen {
		return Validation("validation_failed", "密码长度不足",
			FieldError{Field: "password", Issue: "至少 8 个字符"})
	}
	if len(password) > maxPasswordLen {
		return Validation("validation_failed", "密码过长",
			FieldError{Field: "password", Issue: "最多 128 个字符"})
	}
	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
