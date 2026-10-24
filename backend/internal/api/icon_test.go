package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// --- SVG and upload safety ---

func TestSVGUploadIsServedAsImageWithSandbox(t *testing.T) {
	e := newTestEnv(t)
	expectStatus(t, e.setup("admin", "supersecret"), http.StatusCreated)

	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><circle cx="12" cy="12" r="10"/></svg>`)
	rec := e.uploadFile("icon.svg", svg)
	expectStatus(t, rec, http.StatusCreated)
	uploaded := decodeInto[map[string]any](t, rec)

	rec = e.do(http.MethodGet, uploaded["url"].(string), nil, nil)
	expectStatus(t, rec, http.StatusOK)

	header := rec.Header()
	if ct := header.Get("Content-Type"); ct != "image/svg+xml" {
		t.Errorf("Content-Type = %q, want image/svg+xml", ct)
	}
	if csp := header.Get("Content-Security-Policy"); !strings.Contains(csp, "sandbox") {
		t.Errorf("CSP = %q, want a sandbox directive", csp)
	}
	if csp := header.Get("Content-Security-Policy"); !strings.Contains(csp, "default-src 'none'") {
		t.Errorf("CSP = %q, want default-src 'none'", csp)
	}
	if header.Get("X-Content-Type-Options") != "nosniff" {
		t.Error("X-Content-Type-Options must be nosniff")
	}
	// The response must not be sniffable as HTML.
	if strings.Contains(strings.ToLower(rec.Body.String()), "<script") {
		t.Error("stored SVG unexpectedly contains script")
	}
}

func TestUploadRejectsHTMLMasqueradingAsSVG(t *testing.T) {
	e := newTestEnv(t)
	expectStatus(t, e.setup("admin", "supersecret"), http.StatusCreated)

	rec := e.uploadFile("evil.svg", []byte(`<script>alert(document.cookie)</script>`))
	expectStatus(t, rec, http.StatusUnprocessableEntity)

	rec = e.uploadFile("evil.svg", []byte(`<html><body>hi</body></html>`))
	expectStatus(t, rec, http.StatusUnprocessableEntity)
}

func TestUploadRejectsUnsupportedAndOversizedFiles(t *testing.T) {
	e := newTestEnv(t)
	expectStatus(t, e.setup("admin", "supersecret"), http.StatusCreated)

	rec := e.uploadFile("note.txt", []byte("just text"))
	expectStatus(t, rec, http.StatusUnprocessableEntity)

	// > 1 MiB PNG must be rejected.
	big := append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, bytes.Repeat([]byte{0}, 1<<20)...)
	rec = e.uploadFile("big.png", big)
	if rec.Code != http.StatusRequestEntityTooLarge && rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("oversized upload status = %d, want 413 or 422", rec.Code)
	}
}

func TestPublicUploadRejectsPathTraversal(t *testing.T) {
	e := newTestEnv(t)

	for _, path := range []string{
		"/api/v1/public/uploads/..%2F..%2Fapp.db",
		"/api/v1/public/uploads/..%5Capp.db",
		"/api/v1/public/uploads/%2e%2e%2fapp.db",
	} {
		rec := e.do(http.MethodGet, path, nil, nil)
		if rec.Code == http.StatusOK {
			t.Errorf("GET %s unexpectedly succeeded", path)
		}
	}
}

func TestUploadRequiresAuth(t *testing.T) {
	e := newTestEnv(t)
	anon := &testEnv{t: t, handler: e.handler, cfg: e.cfg, db: e.db, cookies: map[string]string{}}
	rec := anon.uploadFile("icon.png", []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})
	expectStatus(t, rec, http.StatusUnauthorized)
}

// TestBackupAndRestoreIncludesUploads verifies a snapshot covers both the
// database and the uploaded icon files.
func TestBackupAndRestoreIncludesUploads(t *testing.T) {
	e := newTestEnv(t)
	expectStatus(t, e.setup("admin", "supersecret"), http.StatusCreated)

	png := append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, bytes.Repeat([]byte{1}, 32)...)

	rec := e.uploadFile("first.png", png)
	expectStatus(t, rec, http.StatusCreated)
	first := decodeInto[map[string]any](t, rec)["filename"].(string)

	rec = e.do(http.MethodPost, "/api/v1/admin/backups", nil, nil)
	expectStatus(t, rec, http.StatusCreated)
	backupID := decodeInto[map[string]any](t, rec)["id"].(string)

	// Change the uploads directory after the snapshot.
	rec = e.uploadFile("second.png", png)
	expectStatus(t, rec, http.StatusCreated)
	second := decodeInto[map[string]any](t, rec)["filename"].(string)

	rec = e.do(http.MethodPost, "/api/v1/admin/backups/"+backupID+"/restore", nil, nil)
	expectStatus(t, rec, http.StatusOK)

	entries, err := os.ReadDir(e.cfg.UploadsDir)
	if err != nil {
		t.Fatalf("read uploads dir: %v", err)
	}
	names := map[string]bool{}
	for _, entry := range entries {
		names[entry.Name()] = true
	}
	if !names[first] {
		t.Errorf("restored uploads are missing %q (have %v)", first, names)
	}
	if names[second] {
		t.Errorf("upload %q should have been removed by the restore", second)
	}
}

// --- icon types ---

func TestIconTypeValidation(t *testing.T) {
	e := newTestEnv(t)
	expectStatus(t, e.setup("admin", "supersecret"), http.StatusCreated)

	rec := e.do(http.MethodPost, "/api/v1/admin/categories", map[string]any{"name": "Icons"}, nil)
	catID := decodeInto[struct {
		ID int64 `json:"id"`
	}](t, rec).ID

	create := func(iconType, iconValue string) *httptest.ResponseRecorder {
		return e.do(http.MethodPost, "/api/v1/admin/bookmarks", map[string]any{
			"category_id": catID, "title": "T-" + iconType, "url": "https://example.com/",
			"icon_type": iconType, "icon_value": iconValue,
		}, nil)
	}

	// Valid cases.
	valid := []struct{ iconType, iconValue string }{
		{"url", "https://cdn.example.com/i.png"},
		{"upload", "/api/v1/public/uploads/abc.svg"},
		{"simple", "github"},
		{"text", "延迟"},
		{"favicon", ""},
		{"none", ""},
	}
	for _, c := range valid {
		if rec := create(c.iconType, c.iconValue); rec.Code != http.StatusCreated {
			t.Errorf("icon (%s,%q) status = %d, want 201 (%s)", c.iconType, c.iconValue, rec.Code, rec.Body.String())
		}
	}

	// Invalid cases.
	invalid := []struct{ iconType, iconValue string }{
		{"url", "javascript:alert(1)"},
		{"url", ""},
		{"upload", "https://example.com/x.svg"}, // must be site-relative
		{"simple", "微软"},                        // not a verified slug
		{"simple", "definitely-not-a-slug"},
		{"text", "太长的文字图标内容"},
		{"bogus", "x"},
	}
	for _, c := range invalid {
		rec := create(c.iconType, c.iconValue)
		if rec.Code == http.StatusCreated {
			t.Errorf("icon (%s,%q) unexpectedly accepted", c.iconType, c.iconValue)
		}
	}
}

func TestCategoryIconAndColorValidation(t *testing.T) {
	e := newTestEnv(t)
	expectStatus(t, e.setup("admin", "supersecret"), http.StatusCreated)

	rec := e.do(http.MethodPost, "/api/v1/admin/categories", map[string]any{
		"name": "常用", "icon": "✨", "color": "#2D72D9",
	}, nil)
	expectStatus(t, rec, http.StatusCreated)
	created := decodeInto[map[string]any](t, rec)
	if created["icon"] != "✨" || created["color"] != "#2D72D9" {
		t.Errorf("category icon/color = %v / %v", created["icon"], created["color"])
	}

	rec = e.do(http.MethodPost, "/api/v1/admin/categories", map[string]any{
		"name": "Bad", "color": "red",
	}, nil)
	expectStatus(t, rec, http.StatusBadRequest)

	rec = e.do(http.MethodPost, "/api/v1/admin/categories", map[string]any{
		"name": "Bad2", "icon": "这个图标文字太长了超过八个字符",
	}, nil)
	expectStatus(t, rec, http.StatusBadRequest)
}

func TestBookmarkTagsAndColor(t *testing.T) {
	e := newTestEnv(t)
	expectStatus(t, e.setup("admin", "supersecret"), http.StatusCreated)

	rec := e.do(http.MethodPost, "/api/v1/admin/categories", map[string]any{"name": "C"}, nil)
	catID := decodeInto[struct {
		ID int64 `json:"id"`
	}](t, rec).ID

	rec = e.do(http.MethodPost, "/api/v1/admin/bookmarks", map[string]any{
		"category_id": catID, "title": "Tagged", "url": "https://example.com/",
		"tags": "Info,Art", "color": "#F58025",
	}, nil)
	expectStatus(t, rec, http.StatusCreated)
	created := decodeInto[map[string]any](t, rec)
	if created["tags"] != "Info,Art" || created["color"] != "#F58025" {
		t.Errorf("tags/color = %v / %v", created["tags"], created["color"])
	}

	rec = e.do(http.MethodPost, "/api/v1/admin/bookmarks", map[string]any{
		"category_id": catID, "title": "Bad", "url": "https://example.com/", "color": "blue",
	}, nil)
	expectStatus(t, rec, http.StatusBadRequest)
}
