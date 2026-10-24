package config

import (
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// Ensure no ambient configuration leaks into the test.
	for _, k := range []string{
		"XN_HOST", "XN_PORT", "XN_DATA_DIR", "XN_DB_PATH", "XN_UPLOADS_DIR",
		"XN_BACKUPS_DIR", "XN_LOG_LEVEL", "XN_ALLOW_GUEST_ACCESS",
		"XN_TRUSTED_PROXIES", "XN_ALLOWED_ORIGINS", "XN_SESSION_COOKIE",
		"XN_SESSION_TTL_HOURS",
	} {
		t.Setenv(k, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Host != DefaultHost {
		t.Errorf("Host = %q, want %q", cfg.Host, DefaultHost)
	}
	if cfg.Port != DefaultPort {
		t.Errorf("Port = %d, want %d", cfg.Port, DefaultPort)
	}
	if cfg.DBPath != filepath.Join(DefaultDir, "app.db") {
		t.Errorf("DBPath = %q", cfg.DBPath)
	}
	if !cfg.AllowGuestAccess {
		t.Error("AllowGuestAccess should default to true")
	}
	if cfg.SessionCookieName != "xiaowork_session" {
		t.Errorf("SessionCookieName = %q", cfg.SessionCookieName)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("XN_HOST", "127.0.0.1")
	t.Setenv("XN_PORT", "9000")
	t.Setenv("XN_DATA_DIR", "/data")
	t.Setenv("XN_ALLOW_GUEST_ACCESS", "false")
	t.Setenv("XN_TRUSTED_PROXIES", "172.18.0.0/16, 127.0.0.1")
	t.Setenv("XN_SESSION_TTL_HOURS", "24")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Addr() != "127.0.0.1:9000" {
		t.Errorf("Addr() = %q", cfg.Addr())
	}
	if cfg.DBPath != filepath.Join("/data", "app.db") {
		t.Errorf("DBPath = %q", cfg.DBPath)
	}
	if cfg.AllowGuestAccess {
		t.Error("AllowGuestAccess should be false")
	}
	if len(cfg.TrustedProxies) != 2 {
		t.Errorf("TrustedProxies = %v", cfg.TrustedProxies)
	}
	if cfg.SessionTTLHours != 24 {
		t.Errorf("SessionTTLHours = %d", cfg.SessionTTLHours)
	}
}

func TestLoadInvalidPort(t *testing.T) {
	t.Setenv("XN_PORT", "70000")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for invalid port")
	}
}

func TestEnsureDirs(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XN_DATA_DIR", base)
	t.Setenv("XN_UPLOADS_DIR", filepath.Join(base, "uploads"))
	t.Setenv("XN_BACKUPS_DIR", filepath.Join(base, "backups"))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if err := cfg.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs() error = %v", err)
	}
}
