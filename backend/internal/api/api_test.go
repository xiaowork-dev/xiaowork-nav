package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/config"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/migrations"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/static"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/store"
)

func newTestHandler(t *testing.T, withStatic bool) http.Handler {
	t.Helper()
	dir := t.TempDir()

	cfg := &config.Config{
		Host:              "127.0.0.1",
		Port:              8080,
		DataDir:           dir,
		DBPath:            filepath.Join(dir, "app.db"),
		UploadsDir:        filepath.Join(dir, "uploads"),
		BackupsDir:        filepath.Join(dir, "backups"),
		SessionCookieName: "xiaowork_session",
		SessionTTLHours:   168,
		AllowGuestAccess:  true,
	}

	db, err := store.Open(cfg.DBPath)
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := migrations.Apply(db); err != nil {
		t.Fatalf("migrations.Apply() error = %v", err)
	}

	deps := Deps{
		Config: cfg,
		DB:     db,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	if withStatic {
		deps.Static = static.Handler()
	}
	return New(deps).Handler()
}

func do(t *testing.T, h http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHealth(t *testing.T) {
	h := newTestHandler(t, false)
	rec := do(t, h, http.MethodGet, "/healthz")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status field = %v", body["status"])
	}
	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("X-Request-ID header missing")
	}
}

func TestVersionReportsSchema(t *testing.T) {
	h := newTestHandler(t, false)
	rec := do(t, h, http.MethodGet, "/api/v1/version")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Version       string `json:"version"`
		SchemaVersion int    `json:"schema_version"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.SchemaVersion < 1 {
		t.Errorf("schema_version = %d, want >= 1", body.SchemaVersion)
	}
}

func TestAuthStatusNotInitialized(t *testing.T) {
	h := newTestHandler(t, false)
	rec := do(t, h, http.MethodGet, "/api/v1/auth/status")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Initialized   bool `json:"initialized"`
		Authenticated bool `json:"authenticated"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Initialized {
		t.Error("initialized should be false on a fresh database")
	}
	if body.Authenticated {
		t.Error("authenticated should be false")
	}
}

func TestUnknownAPIReturnsJSON404(t *testing.T) {
	h := newTestHandler(t, true)
	rec := do(t, h, http.MethodGet, "/api/v1/does-not-exist")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want JSON", ct)
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Code != CodeNotFound {
		t.Errorf("error.code = %q, want %q", body.Error.Code, CodeNotFound)
	}
}

func TestSPAFallbackServesIndex(t *testing.T) {
	h := newTestHandler(t, true)
	rec := do(t, h, http.MethodGet, "/admin/some/deep/route")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
	if !strings.Contains(rec.Body.String(), "<html") {
		t.Error("expected HTML shell in SPA fallback")
	}
}

func TestMethodNotAllowedOnHealth(t *testing.T) {
	h := newTestHandler(t, false)
	rec := do(t, h, http.MethodPost, "/healthz")
	if rec.Code == http.StatusOK {
		t.Errorf("POST /healthz should not return 200, got %d", rec.Code)
	}
}
