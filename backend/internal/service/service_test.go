package service

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/config"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/migrations"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := migrations.Apply(db); err != nil {
		t.Fatalf("migrations.Apply() error = %v", err)
	}
	return store.New(db)
}

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	dir := t.TempDir()
	return &config.Config{
		DataDir:           dir,
		DBPath:            filepath.Join(dir, "app.db"),
		UploadsDir:        filepath.Join(dir, "uploads"),
		BackupsDir:        filepath.Join(dir, "backups"),
		SessionCookieName: "xiaowork_session",
		SessionTTLHours:   168,
		AllowGuestAccess:  true,
		BackupKeep:        5,
	}
}

func TestAuthLoginRateLimit(t *testing.T) {
	st := newTestStore(t)
	svc := NewAuthService(st, testConfig(t))
	ctx := context.Background()

	if _, err := svc.Setup(ctx, "admin", "supersecret"); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	// Exhaust the limiter with wrong passwords.
	var lastErr error
	for i := 0; i < loginLimit; i++ {
		_, _, lastErr = svc.Login(ctx, "admin", "wrong-password", "203.0.113.10", "test-agent")
		if lastErr == nil {
			t.Fatal("login with a wrong password must fail")
		}
	}
	// The limiter keys on both the client IP and the username, so the account
	// is protected even across many source IPs.
	_, _, err := svc.Login(ctx, "admin", "supersecret", "198.51.100.7", "test-agent")
	if err == nil {
		t.Fatal("login should be rate limited for the username across IPs")
	}
	var svcErr *Error
	if !asError(err, &svcErr) || svcErr.Kind != KindRateLimited {
		t.Fatalf("error = %v, want a rate-limited service error", err)
	}

	// A fresh service instance (empty limiter) accepts the correct password,
	// proving the block came from the limiter and not from the credentials.
	fresh := NewAuthService(st, testConfig(t))
	if _, _, err := fresh.Login(ctx, "admin", "supersecret", "198.51.100.7", "test-agent"); err != nil {
		t.Fatalf("login with a fresh limiter should succeed, got %v", err)
	}
}

func TestAuthSessionLifecycle(t *testing.T) {
	st := newTestStore(t)
	svc := NewAuthService(st, testConfig(t))
	ctx := context.Background()
	if _, err := svc.Setup(ctx, "admin", "supersecret"); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	user, session, err := svc.Login(ctx, "admin", "supersecret", "203.0.113.10", "agent")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if session.ID == "" {
		t.Fatal("session ID must not be empty")
	}

	gotUser, _, err := svc.Authenticate(ctx, session.ID)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if gotUser.ID != user.ID {
		t.Errorf("authenticated user = %d, want %d", gotUser.ID, user.ID)
	}

	if err := svc.Logout(ctx, session.ID); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if _, _, err := svc.Authenticate(ctx, session.ID); err == nil {
		t.Error("Authenticate() after logout should fail")
	}
	if _, _, err := svc.Authenticate(ctx, "does-not-exist"); err == nil {
		t.Error("Authenticate() with an unknown session should fail")
	}
	if _, _, err := svc.Authenticate(ctx, ""); err == nil {
		t.Error("Authenticate() with an empty session should fail")
	}
}

func TestSettingsUpdateValidatesAndMerges(t *testing.T) {
	st := newTestStore(t)
	svc := NewSettingsService(st)
	ctx := context.Background()

	initial, err := svc.Get(ctx)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if initial.SiteTitle != "XiaoWork Nav" || !initial.AllowGuestAccess {
		t.Fatalf("unexpected defaults: %+v", initial)
	}
	if len(initial.SearchEngines) == 0 || initial.DefaultSearchEngine == "" {
		t.Fatalf("expected default search engines: %+v", initial.SearchEngines)
	}

	patch := map[string]json.RawMessage{
		"site_title": json.RawMessage(`"新标题"`),
	}
	updated, err := svc.Update(ctx, patch)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.SiteTitle != "新标题" {
		t.Errorf("site_title = %q", updated.SiteTitle)
	}
	// Unspecified fields keep their defaults.
	if updated.Theme != "auto" {
		t.Errorf("theme = %q, want auto", updated.Theme)
	}

	// Persisted across reads.
	reloaded, err := svc.Get(ctx)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if reloaded.SiteTitle != "新标题" {
		t.Errorf("reloaded site_title = %q", reloaded.SiteTitle)
	}

	invalid := []map[string]json.RawMessage{
		{"theme": json.RawMessage(`"neon"`)},
		{"homepage_layout": json.RawMessage(`"masonry"`)},
		{"site_title": json.RawMessage(`""`)},
		{"nope": json.RawMessage(`1`)},
		{"default_search_engine": json.RawMessage(`"Missing"`)},
	}
	for _, p := range invalid {
		if _, err := svc.Update(ctx, p); err == nil {
			t.Errorf("Update(%v) should have failed", p)
		}
	}
}

func TestCatalogValidation(t *testing.T) {
	st := newTestStore(t)
	settings := NewSettingsService(st)
	catalog := NewCatalogService(st, settings)
	ctx := context.Background()

	if _, err := catalog.CreateCategory(ctx, CategoryInput{Name: "   "}); err == nil {
		t.Error("empty category name should be rejected")
	}
	if _, err := catalog.CreateCategory(ctx, CategoryInput{Name: string(make([]byte, 0))}); err == nil {
		t.Error("empty category name should be rejected")
	}

	cat, err := catalog.CreateCategory(ctx, CategoryInput{Name: "Valid"})
	if err != nil {
		t.Fatalf("CreateCategory() error = %v", err)
	}

	badURLs := []string{"javascript:alert(1)", "ftp://x.example", "", "not a url"}
	for _, u := range badURLs {
		_, err := catalog.CreateBookmark(ctx, BookmarkInput{CategoryID: &cat.ID, Title: "T", URL: u})
		if err == nil {
			t.Errorf("bookmark with url %q should be rejected", u)
		}
	}

	if _, err := catalog.CreateBookmark(ctx, BookmarkInput{CategoryID: &cat.ID, Title: "T", URL: "https://ok.example"}); err != nil {
		t.Fatalf("valid bookmark rejected: %v", err)
	}

	// Moving to a non-existent category fails.
	if _, err := catalog.MoveBookmark(ctx, 1, 9999, nil); err == nil {
		t.Error("move to a missing category should fail")
	}

	// Search requires a query.
	if _, _, err := catalog.Search(ctx, "   ", 1, 20); err == nil {
		t.Error("empty search query should be rejected")
	}
}

func asError(err error, target **Error) bool {
	e, ok := err.(*Error)
	if ok {
		*target = e
	}
	return ok
}
