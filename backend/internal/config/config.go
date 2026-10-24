// Package config loads runtime configuration from environment variables.
//
// All deployment-specific values (host, port, data paths, trusted proxies)
// are injected via environment variables so that nothing is hard-coded to a
// particular domain, Docker network, or server.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Default values used when an environment variable is not set.
const (
	DefaultHost = "0.0.0.0"
	DefaultPort = 8080
	DefaultDir  = "./data"
)

// Config is the resolved application configuration.
type Config struct {
	// Host is the listen address.
	Host string
	// Port is the listen port.
	Port int
	// DataDir is the persistent data root (SQLite, uploads, backups).
	DataDir string
	// DBPath is the SQLite database file path.
	DBPath string
	// UploadsDir stores uploaded icons.
	UploadsDir string
	// BackupsDir stores database/upload snapshots.
	BackupsDir string
	// LogLevel is one of debug, info, warn, error.
	LogLevel string
	// AllowGuestAccess controls whether anonymous visitors may read public data.
	AllowGuestAccess bool
	// TrustedProxies is a list of CIDRs/IPs whose X-Forwarded-* headers are honored.
	TrustedProxies []string
	// AllowedOrigins restricts cross-origin requests in development.
	AllowedOrigins []string
	// SessionCookieName is the auth cookie name.
	SessionCookieName string
	// SessionTTLHours is the session lifetime in hours.
	SessionTTLHours int
	// SecureCookie marks the session cookie Secure (HTTPS only).
	SecureCookie bool
	// UpdateRepo is the "owner/name" GitHub repository used for update checks
	// (empty disables remote version checks).
	UpdateRepo string
	// BackupKeep is the number of snapshots to retain (0 = unlimited).
	BackupKeep int
}

// Load reads configuration from the environment, applying defaults.
func Load() (*Config, error) {
	dataDir := getEnv("XN_DATA_DIR", DefaultDir)

	cfg := &Config{
		Host:              getEnv("XN_HOST", DefaultHost),
		Port:              getEnvInt("XN_PORT", DefaultPort),
		DataDir:           dataDir,
		DBPath:            getEnv("XN_DB_PATH", filepath.Join(dataDir, "app.db")),
		UploadsDir:        getEnv("XN_UPLOADS_DIR", filepath.Join(dataDir, "uploads")),
		BackupsDir:        getEnv("XN_BACKUPS_DIR", filepath.Join(dataDir, "backups")),
		LogLevel:          strings.ToLower(getEnv("XN_LOG_LEVEL", "info")),
		AllowGuestAccess:  getEnvBool("XN_ALLOW_GUEST_ACCESS", true),
		TrustedProxies:    getEnvList("XN_TRUSTED_PROXIES"),
		AllowedOrigins:    getEnvList("XN_ALLOWED_ORIGINS"),
		SessionCookieName: getEnv("XN_SESSION_COOKIE", "xiaowork_session"),
		SessionTTLHours:   getEnvInt("XN_SESSION_TTL_HOURS", 168),
		SecureCookie:      getEnvBool("XN_SECURE_COOKIE", true),
		UpdateRepo:        getEnv("XN_UPDATE_REPO", ""),
		BackupKeep:        getEnvInt("XN_BACKUP_KEEP", 5),
	}

	if cfg.Port < 1 || cfg.Port > 65535 {
		return nil, fmt.Errorf("invalid XN_PORT: %d", cfg.Port)
	}
	if cfg.SessionTTLHours < 1 {
		return nil, fmt.Errorf("invalid XN_SESSION_TTL_HOURS: %d", cfg.SessionTTLHours)
	}
	return cfg, nil
}

// Addr returns the host:port listen address.
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// EnsureDirs creates the data directories if they do not exist.
func (c *Config) EnsureDirs() error {
	for _, dir := range []string{c.DataDir, c.UploadsDir, c.BackupsDir} {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("create directory %q: %w", dir, err)
		}
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return fallback
	}
	return n
}

func getEnvBool(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(strings.TrimSpace(v))
	if err != nil {
		return fallback
	}
	return b
}

func getEnvList(key string) []string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
