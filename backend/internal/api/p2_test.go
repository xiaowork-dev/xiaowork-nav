package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/config"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/migrations"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/store"
)

// --- harness ---

type testEnv struct {
	t       *testing.T
	handler http.Handler
	cfg     *config.Config
	db      *sql.DB
	cookies map[string]string
}

func newTestEnv(t *testing.T) *testEnv {
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
		SecureCookie:      false,
		BackupKeep:        5,
	}

	db, err := store.Open(cfg.DBPath)
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := migrations.Apply(db); err != nil {
		t.Fatalf("migrations.Apply() error = %v", err)
	}

	handler := New(Deps{
		Config: cfg,
		DB:     db,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}).Handler()

	return &testEnv{t: t, handler: handler, cfg: cfg, db: db, cookies: map[string]string{}}
}

func (e *testEnv) do(method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	e.t.Helper()

	var reader io.Reader
	switch b := body.(type) {
	case nil:
	case string:
		reader = strings.NewReader(b)
	case []byte:
		reader = bytes.NewReader(b)
	default:
		data, err := json.Marshal(b)
		if err != nil {
			e.t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(data)
	}

	req := httptest.NewRequest(method, path, reader)
	if body != nil && headers["Content-Type"] == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	for name, value := range e.cookies {
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}

	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)

	for _, c := range rec.Result().Cookies() {
		if c.MaxAge < 0 {
			delete(e.cookies, c.Name)
			continue
		}
		e.cookies[c.Name] = c.Value
	}
	return rec
}

func (e *testEnv) setup(username, password string) *httptest.ResponseRecorder {
	return e.do(http.MethodPost, "/api/v1/setup", map[string]string{
		"username": username, "password": password,
	}, nil)
}

func (e *testEnv) login(username, password string) *httptest.ResponseRecorder {
	return e.do(http.MethodPost, "/api/v1/auth/login", map[string]string{
		"username": username, "password": password,
	}, nil)
}

func decodeInto[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	return out
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	body := decodeInto[errorEnvelope](t, rec)
	return body.Error.Code
}

func expectStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, want, rec.Body.String())
	}
}

// --- authentication ---

func TestSetupLoginLogoutFlow(t *testing.T) {
	e := newTestEnv(t)

	rec := e.do(http.MethodGet, "/api/v1/auth/status", nil, nil)
	expectStatus(t, rec, http.StatusOK)
	if st := decodeInto[authStatusResponse](t, rec); st.Initialized || st.Authenticated {
		t.Fatalf("fresh status = %+v, want uninitialized and unauthenticated", st)
	}

	rec = e.setup("admin", "supersecret")
	expectStatus(t, rec, http.StatusCreated)

	rec = e.do(http.MethodGet, "/api/v1/auth/me", nil, nil)
	expectStatus(t, rec, http.StatusOK)

	rec = e.do(http.MethodPost, "/api/v1/auth/logout", nil, nil)
	expectStatus(t, rec, http.StatusNoContent)

	rec = e.do(http.MethodGet, "/api/v1/auth/me", nil, nil)
	expectStatus(t, rec, http.StatusUnauthorized)
	if code := errorCode(t, rec); code != CodeUnauthorized {
		t.Errorf("logout error code = %q, want %q", code, CodeUnauthorized)
	}

	rec = e.login("admin", "supersecret")
	expectStatus(t, rec, http.StatusOK)
	rec = e.do(http.MethodGet, "/api/v1/auth/me", nil, nil)
	expectStatus(t, rec, http.StatusOK)
}

func TestSetupIsOneTimeOnly(t *testing.T) {
	e := newTestEnv(t)
	expectStatus(t, e.setup("admin", "supersecret"), http.StatusCreated)

	rec := e.setup("other", "anothersecret")
	expectStatus(t, rec, http.StatusConflict)
	if code := errorCode(t, rec); code != CodeAlreadyInitialized {
		t.Errorf("code = %q, want %q", code, CodeAlreadyInitialized)
	}
}

func TestSetupValidatesCredentials(t *testing.T) {
	cases := []struct{ user, pass string }{
		{"ab", "supersecret"},        // username too short
		{"has space", "supersecret"}, // invalid characters
		{"admin", "short"},           // password too short
	}
	for _, c := range cases {
		e := newTestEnv(t)
		rec := e.setup(c.user, c.pass)
		expectStatus(t, rec, http.StatusBadRequest)
	}
}

func TestLoginRejectsBadCredentials(t *testing.T) {
	e := newTestEnv(t)
	expectStatus(t, e.setup("admin", "supersecret"), http.StatusCreated)
	expectStatus(t, e.do(http.MethodPost, "/api/v1/auth/logout", nil, nil), http.StatusNoContent)

	rec := e.login("admin", "wrong-password")
	expectStatus(t, rec, http.StatusUnauthorized)
	if code := errorCode(t, rec); code != CodeInvalidCredentials {
		t.Errorf("code = %q, want %q", code, CodeInvalidCredentials)
	}

	rec = e.login("nobody", "supersecret")
	expectStatus(t, rec, http.StatusUnauthorized)
}

func TestChangePasswordInvalidatesSessions(t *testing.T) {
	e := newTestEnv(t)
	expectStatus(t, e.setup("admin", "supersecret"), http.StatusCreated)

	rec := e.do(http.MethodPut, "/api/v1/auth/password", map[string]string{
		"current_password": "supersecret",
		"new_password":     "brandnewsecret",
	}, nil)
	expectStatus(t, rec, http.StatusNoContent)

	rec = e.do(http.MethodGet, "/api/v1/auth/me", nil, nil)
	expectStatus(t, rec, http.StatusUnauthorized)

	expectStatus(t, e.login("admin", "supersecret"), http.StatusUnauthorized)
	expectStatus(t, e.login("admin", "brandnewsecret"), http.StatusOK)
}

func TestChangePasswordRejectsWrongCurrent(t *testing.T) {
	e := newTestEnv(t)
	expectStatus(t, e.setup("admin", "supersecret"), http.StatusCreated)

	rec := e.do(http.MethodPut, "/api/v1/auth/password", map[string]string{
		"current_password": "not-the-password",
		"new_password":     "brandnewsecret",
	}, nil)
	expectStatus(t, rec, http.StatusUnauthorized)
}

// --- permission boundaries ---

func TestAdminEndpointsRequireAuth(t *testing.T) {
	e := newTestEnv(t)

	protected := []struct {
		method, path string
	}{
		{http.MethodGet, "/api/v1/admin/categories"},
		{http.MethodPost, "/api/v1/admin/categories"},
		{http.MethodPut, "/api/v1/admin/categories/order"},
		{http.MethodGet, "/api/v1/admin/categories/1"},
		{http.MethodPatch, "/api/v1/admin/categories/1"},
		{http.MethodDelete, "/api/v1/admin/categories/1"},
		{http.MethodGet, "/api/v1/admin/bookmarks"},
		{http.MethodPost, "/api/v1/admin/bookmarks"},
		{http.MethodGet, "/api/v1/admin/settings"},
		{http.MethodPut, "/api/v1/admin/settings"},
		{http.MethodGet, "/api/v1/admin/export"},
		{http.MethodPost, "/api/v1/admin/import"},
		{http.MethodGet, "/api/v1/admin/backups"},
		{http.MethodPost, "/api/v1/admin/backups"},
		{http.MethodGet, "/api/v1/admin/update-check"},
		{http.MethodGet, "/api/v1/auth/me"},
	}

	for _, p := range protected {
		rec := e.do(p.method, p.path, nil, nil)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s status = %d, want 401", p.method, p.path, rec.Code)
			continue
		}
		if code := errorCode(t, rec); code != CodeUnauthorized {
			t.Errorf("%s %s error code = %q, want %q", p.method, p.path, code, CodeUnauthorized)
		}
	}
}

func TestPublicEndpointsAreAnonymous(t *testing.T) {
	e := newTestEnv(t)
	for _, path := range []string{
		"/api/v1/public/navigation",
		"/api/v1/public/categories",
		"/api/v1/public/settings",
	} {
		expectStatus(t, e.do(http.MethodGet, path, nil, nil), http.StatusOK)
	}
}

func TestGuestAccessDisabledRequiresLogin(t *testing.T) {
	e := newTestEnv(t)
	expectStatus(t, e.setup("admin", "supersecret"), http.StatusCreated)

	// Disable guest access via settings.
	rec := e.do(http.MethodPut, "/api/v1/admin/settings", map[string]any{
		"allow_guest_access": false,
	}, nil)
	expectStatus(t, rec, http.StatusOK)

	// Admin session still works.
	expectStatus(t, e.do(http.MethodGet, "/api/v1/public/navigation", nil, nil), http.StatusOK)

	// Anonymous session is refused.
	anon := &testEnv{t: t, handler: e.handler, cfg: e.cfg, db: e.db, cookies: map[string]string{}}
	rec = anon.do(http.MethodGet, "/api/v1/public/navigation", nil, nil)
	expectStatus(t, rec, http.StatusUnauthorized)
}

func TestOriginCheckBlocksCrossOriginWrites(t *testing.T) {
	e := newTestEnv(t)

	rec := e.do(http.MethodPost, "/api/v1/setup", map[string]string{
		"username": "admin", "password": "supersecret",
	}, map[string]string{"Origin": "https://evil.example"})
	expectStatus(t, rec, http.StatusForbidden)
	if code := errorCode(t, rec); code != CodeCSRFFailed {
		t.Errorf("code = %q, want %q", code, CodeCSRFFailed)
	}

	// Same-origin writes are accepted (httptest requests use host example.com).
	rec = e.do(http.MethodPost, "/api/v1/setup", map[string]string{
		"username": "admin", "password": "supersecret",
	}, map[string]string{"Origin": "http://example.com"})
	expectStatus(t, rec, http.StatusCreated)
}

// --- categories ---

func TestCategoryCRUDAndVisibility(t *testing.T) {
	e := newTestEnv(t)
	expectStatus(t, e.setup("admin", "supersecret"), http.StatusCreated)

	rec := e.do(http.MethodPost, "/api/v1/admin/categories", map[string]any{
		"name": "常用", "description": "常用站点", "sort_order": 10, "is_visible": true,
	}, nil)
	expectStatus(t, rec, http.StatusCreated)
	created := decodeInto[struct {
		ID int64 `json:"id"`
	}](t, rec)

	rec = e.do(http.MethodPost, "/api/v1/admin/categories", map[string]any{
		"name": "私有", "is_visible": false,
	}, nil)
	expectStatus(t, rec, http.StatusCreated)
	private := decodeInto[struct {
		ID int64 `json:"id"`
	}](t, rec)

	// Admin sees both.
	rec = e.do(http.MethodGet, "/api/v1/admin/categories", nil, nil)
	expectStatus(t, rec, http.StatusOK)
	adminList := decodeInto[listResponse[map[string]any]](t, rec)
	if adminList.Total != 2 {
		t.Errorf("admin categories total = %d, want 2", adminList.Total)
	}

	// Public sees only the visible one.
	rec = e.do(http.MethodGet, "/api/v1/public/categories", nil, nil)
	expectStatus(t, rec, http.StatusOK)
	publicList := decodeInto[listResponse[map[string]any]](t, rec)
	if publicList.Total != 1 {
		t.Errorf("public categories total = %d, want 1", publicList.Total)
	}

	// Patch a category.
	rec = e.do(http.MethodPatch, "/api/v1/admin/categories/"+itoa(created.ID), map[string]any{
		"name": "常用导航",
	}, nil)
	expectStatus(t, rec, http.StatusOK)
	patched := decodeInto[map[string]any](t, rec)
	if patched["name"] != "常用导航" {
		t.Errorf("patched name = %v", patched["name"])
	}

	// Delete.
	rec = e.do(http.MethodDelete, "/api/v1/admin/categories/"+itoa(private.ID), nil, nil)
	expectStatus(t, rec, http.StatusNoContent)

	rec = e.do(http.MethodDelete, "/api/v1/admin/categories/"+itoa(private.ID), nil, nil)
	expectStatus(t, rec, http.StatusNotFound)
}

func TestCategoryReorder(t *testing.T) {
	e := newTestEnv(t)
	expectStatus(t, e.setup("admin", "supersecret"), http.StatusCreated)

	ids := make([]int64, 0, 2)
	for _, name := range []string{"A", "B"} {
		rec := e.do(http.MethodPost, "/api/v1/admin/categories", map[string]any{"name": name}, nil)
		expectStatus(t, rec, http.StatusCreated)
		ids = append(ids, decodeInto[struct {
			ID int64 `json:"id"`
		}](t, rec).ID)
	}

	rec := e.do(http.MethodPut, "/api/v1/admin/categories/order", map[string]any{
		"items": []map[string]any{
			{"id": ids[0], "sort_order": 20},
			{"id": ids[1], "sort_order": 10},
		},
	}, nil)
	expectStatus(t, rec, http.StatusOK)
	result := decodeInto[map[string]any](t, rec)
	if result["updated"].(float64) != 2 {
		t.Errorf("updated = %v, want 2", result["updated"])
	}

	rec = e.do(http.MethodGet, "/api/v1/admin/categories", nil, nil)
	list := decodeInto[listResponse[map[string]any]](t, rec)
	first := list.Items[0]["name"]
	if first != "B" {
		t.Errorf("first category after reorder = %v, want B", first)
	}
}

// --- bookmarks ---

func TestBookmarkCRUDAndURLValidation(t *testing.T) {
	e := newTestEnv(t)
	expectStatus(t, e.setup("admin", "supersecret"), http.StatusCreated)

	rec := e.do(http.MethodPost, "/api/v1/admin/categories", map[string]any{"name": "AI"}, nil)
	catID := decodeInto[struct {
		ID int64 `json:"id"`
	}](t, rec).ID

	// Invalid URL scheme must be rejected.
	rec = e.do(http.MethodPost, "/api/v1/admin/bookmarks", map[string]any{
		"category_id": catID, "title": "Bad", "url": "javascript:alert(1)",
	}, nil)
	expectStatus(t, rec, http.StatusUnprocessableEntity)
	if code := errorCode(t, rec); code != CodeInvalidURLScheme {
		t.Errorf("code = %q, want %q", code, CodeInvalidURLScheme)
	}

	// Unknown category must be rejected.
	rec = e.do(http.MethodPost, "/api/v1/admin/bookmarks", map[string]any{
		"category_id": 9999, "title": "X", "url": "https://example.com",
	}, nil)
	expectStatus(t, rec, http.StatusBadRequest)

	// Valid create.
	rec = e.do(http.MethodPost, "/api/v1/admin/bookmarks", map[string]any{
		"category_id": catID, "title": "ChatGPT", "url": "https://chat.openai.com",
		"keywords": "ai,chat", "icon_type": "url", "icon_value": "https://chat.openai.com/favicon.ico",
	}, nil)
	expectStatus(t, rec, http.StatusCreated)
	bookmark := decodeInto[map[string]any](t, rec)
	bookmarkID := int64(bookmark["id"].(float64))

	// Patch.
	rec = e.do(http.MethodPatch, "/api/v1/admin/bookmarks/"+itoa(bookmarkID), map[string]any{
		"title": "ChatGPT 4",
	}, nil)
	expectStatus(t, rec, http.StatusOK)

	// Move to a second category.
	rec = e.do(http.MethodPost, "/api/v1/admin/categories", map[string]any{"name": "Other"}, nil)
	otherID := decodeInto[struct {
		ID int64 `json:"id"`
	}](t, rec).ID
	rec = e.do(http.MethodPost, "/api/v1/admin/bookmarks/"+itoa(bookmarkID)+"/move", map[string]any{
		"category_id": otherID, "sort_order": 5,
	}, nil)
	expectStatus(t, rec, http.StatusOK)
	moved := decodeInto[map[string]any](t, rec)
	if int64(moved["category_id"].(float64)) != otherID {
		t.Errorf("moved category_id = %v, want %d", moved["category_id"], otherID)
	}

	// Delete.
	expectStatus(t, e.do(http.MethodDelete, "/api/v1/admin/bookmarks/"+itoa(bookmarkID), nil, nil), http.StatusNoContent)
	expectStatus(t, e.do(http.MethodGet, "/api/v1/admin/bookmarks/"+itoa(bookmarkID), nil, nil), http.StatusNotFound)
}

func TestPublicNavigationAndSearchRespectVisibility(t *testing.T) {
	e := newTestEnv(t)
	expectStatus(t, e.setup("admin", "supersecret"), http.StatusCreated)

	rec := e.do(http.MethodPost, "/api/v1/admin/categories", map[string]any{"name": "公开", "is_visible": true}, nil)
	publicCat := decodeInto[struct {
		ID int64 `json:"id"`
	}](t, rec).ID

	rec = e.do(http.MethodPost, "/api/v1/admin/categories", map[string]any{"name": "私有", "is_visible": false}, nil)
	privateCat := decodeInto[struct {
		ID int64 `json:"id"`
	}](t, rec).ID

	expectStatus(t, e.do(http.MethodPost, "/api/v1/admin/bookmarks", map[string]any{
		"category_id": publicCat, "title": "Public Site", "url": "https://public.example", "keywords": "share",
	}, nil), http.StatusCreated)

	expectStatus(t, e.do(http.MethodPost, "/api/v1/admin/bookmarks", map[string]any{
		"category_id": publicCat, "title": "Hidden Site", "url": "https://hidden.example",
		"keywords": "share", "is_visible": false,
	}, nil), http.StatusCreated)

	expectStatus(t, e.do(http.MethodPost, "/api/v1/admin/bookmarks", map[string]any{
		"category_id": privateCat, "title": "Private Cat Site", "url": "https://private.example", "keywords": "share",
	}, nil), http.StatusCreated)

	// Anonymous navigation must not leak private entries.
	anon := &testEnv{t: t, handler: e.handler, cfg: e.cfg, db: e.db, cookies: map[string]string{}}
	rec = anon.do(http.MethodGet, "/api/v1/public/navigation", nil, nil)
	expectStatus(t, rec, http.StatusOK)
	nav := decodeInto[struct {
		Categories []struct {
			Name      string `json:"name"`
			Bookmarks []struct {
				Title string `json:"title"`
			} `json:"bookmarks"`
		} `json:"categories"`
	}](t, rec)
	if len(nav.Categories) != 1 {
		t.Fatalf("public navigation categories = %d, want 1", len(nav.Categories))
	}
	if len(nav.Categories[0].Bookmarks) != 1 || nav.Categories[0].Bookmarks[0].Title != "Public Site" {
		t.Errorf("public navigation bookmarks = %+v, want only Public Site", nav.Categories[0].Bookmarks)
	}

	// Search must only return visible entries.
	rec = anon.do(http.MethodGet, "/api/v1/public/search?q=share", nil, nil)
	expectStatus(t, rec, http.StatusOK)
	results := decodeInto[listResponse[map[string]any]](t, rec)
	if results.Total != 1 {
		t.Errorf("public search total = %d, want 1", results.Total)
	}

	// Admin search sees everything.
	rec = e.do(http.MethodGet, "/api/v1/admin/bookmarks?q=share", nil, nil)
	expectStatus(t, rec, http.StatusOK)
	adminResults := decodeInto[listResponse[map[string]any]](t, rec)
	if adminResults.Total != 3 {
		t.Errorf("admin search total = %d, want 3", adminResults.Total)
	}
}

// --- settings ---

func TestSettingsDefaultsAndValidation(t *testing.T) {
	e := newTestEnv(t)
	expectStatus(t, e.setup("admin", "supersecret"), http.StatusCreated)

	rec := e.do(http.MethodGet, "/api/v1/admin/settings", nil, nil)
	expectStatus(t, rec, http.StatusOK)
	settings := decodeInto[map[string]any](t, rec)
	if settings["site_title"] != "XiaoWork Nav" {
		t.Errorf("default site_title = %v", settings["site_title"])
	}
	if settings["allow_guest_access"] != true {
		t.Errorf("default allow_guest_access = %v", settings["allow_guest_access"])
	}

	// Invalid theme is rejected.
	rec = e.do(http.MethodPut, "/api/v1/admin/settings", map[string]any{"theme": "neon"}, nil)
	expectStatus(t, rec, http.StatusBadRequest)

	// Unknown key is rejected.
	rec = e.do(http.MethodPut, "/api/v1/admin/settings", map[string]any{"nope": 1}, nil)
	expectStatus(t, rec, http.StatusBadRequest)

	// Search engine without {query} is rejected.
	rec = e.do(http.MethodPut, "/api/v1/admin/settings", map[string]any{
		"search_engines": []map[string]string{{"name": "X", "url_template": "https://x.example/search"}},
	}, nil)
	expectStatus(t, rec, http.StatusBadRequest)

	// Valid partial update.
	rec = e.do(http.MethodPut, "/api/v1/admin/settings", map[string]any{
		"site_title": "我的导航", "theme": "dark",
	}, nil)
	expectStatus(t, rec, http.StatusOK)

	// Public settings must not expose allow_guest_access.
	anon := &testEnv{t: t, handler: e.handler, cfg: e.cfg, db: e.db, cookies: map[string]string{}}
	rec = anon.do(http.MethodGet, "/api/v1/public/settings", nil, nil)
	expectStatus(t, rec, http.StatusOK)
	publicSettings := decodeInto[map[string]any](t, rec)
	if publicSettings["site_title"] != "我的导航" {
		t.Errorf("public site_title = %v", publicSettings["site_title"])
	}
	if _, leaked := publicSettings["allow_guest_access"]; leaked {
		t.Error("public settings must not include allow_guest_access")
	}
}

// --- icons ---

func TestIconUploadValidation(t *testing.T) {
	e := newTestEnv(t)
	expectStatus(t, e.setup("admin", "supersecret"), http.StatusCreated)

	png := append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, bytes.Repeat([]byte{0}, 64)...)

	rec := e.uploadFile("icon.png", png)
	expectStatus(t, rec, http.StatusCreated)
	uploaded := decodeInto[map[string]any](t, rec)
	if !strings.HasPrefix(uploaded["url"].(string), "/api/v1/public/uploads/") {
		t.Errorf("upload url = %v", uploaded["url"])
	}

	// The uploaded icon is served back.
	rec = e.do(http.MethodGet, uploaded["url"].(string), nil, nil)
	expectStatus(t, rec, http.StatusOK)
	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("served content type = %q, want image/png", ct)
	}

	// A text file is rejected.
	rec = e.uploadFile("evil.txt", []byte("this is definitely not an image"))
	expectStatus(t, rec, http.StatusUnprocessableEntity)

	// Anonymous access to the icon endpoint is refused.
	anon := &testEnv{t: t, handler: e.handler, cfg: e.cfg, db: e.db, cookies: map[string]string{}}
	rec = anon.uploadFile("icon.png", png)
	expectStatus(t, rec, http.StatusUnauthorized)
}

func (e *testEnv) uploadFile(filename string, data []byte) *httptest.ResponseRecorder {
	e.t.Helper()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		e.t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		e.t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		e.t.Fatalf("close multipart writer: %v", err)
	}

	return e.do(http.MethodPost, "/api/v1/admin/uploads/icon", buf.Bytes(), map[string]string{
		"Content-Type": writer.FormDataContentType(),
	})
}

func TestIconFetchRejectsInternalTargets(t *testing.T) {
	e := newTestEnv(t)
	expectStatus(t, e.setup("admin", "supersecret"), http.StatusCreated)

	for _, target := range []string{
		"http://127.0.0.1/favicon.ico",
		"http://169.254.169.254/latest/meta-data/",
		"http://10.0.0.5/",
		"file:///etc/passwd",
	} {
		rec := e.do(http.MethodPost, "/api/v1/admin/icons/fetch", map[string]string{"url": target}, nil)
		if rec.Code == http.StatusOK {
			t.Errorf("fetch %q unexpectedly succeeded", target)
		}
	}
}

// --- export / import ---

func TestExportImportRoundTrip(t *testing.T) {
	source := newTestEnv(t)
	expectStatus(t, source.setup("admin", "supersecret"), http.StatusCreated)

	rec := source.do(http.MethodPost, "/api/v1/admin/categories", map[string]any{"name": "AI"}, nil)
	catID := decodeInto[struct {
		ID int64 `json:"id"`
	}](t, rec).ID
	expectStatus(t, source.do(http.MethodPost, "/api/v1/admin/bookmarks", map[string]any{
		"category_id": catID, "title": "ChatGPT", "url": "https://chat.openai.com",
	}, nil), http.StatusCreated)

	rec = source.do(http.MethodGet, "/api/v1/admin/export", nil, nil)
	expectStatus(t, rec, http.StatusOK)
	bundle := rec.Body.Bytes()

	exported := decodeInto[map[string]any](t, rec)
	if exported["format"] != "xiaowork-nav-v1" {
		t.Errorf("export format = %v", exported["format"])
	}
	if !strings.HasPrefix(exported["checksum"].(string), "sha256:") {
		t.Errorf("export checksum = %v", exported["checksum"])
	}

	// Import into a fresh instance.
	target := newTestEnv(t)
	expectStatus(t, target.setup("admin", "supersecret"), http.StatusCreated)

	rec = target.do(http.MethodPost, "/api/v1/admin/import/preview", bundle, nil)
	expectStatus(t, rec, http.StatusOK)
	preview := decodeInto[map[string]any](t, rec)
	if preview["categories_new"].(float64) != 1 || preview["bookmarks_new"].(float64) != 1 {
		t.Errorf("preview = %+v, want 1 new category and 1 new bookmark", preview)
	}

	rec = target.do(http.MethodPost, "/api/v1/admin/import", bundle, nil)
	expectStatus(t, rec, http.StatusOK)
	result := decodeInto[map[string]any](t, rec)
	if result["categories_imported"].(float64) != 1 || result["bookmarks_imported"].(float64) != 1 {
		t.Errorf("import result = %+v", result)
	}

	rec = target.do(http.MethodGet, "/api/v1/public/navigation", nil, nil)
	nav := decodeInto[struct {
		Categories []struct {
			Bookmarks []struct {
				Title string `json:"title"`
			} `json:"bookmarks"`
		} `json:"categories"`
	}](t, rec)
	if len(nav.Categories) != 1 || len(nav.Categories[0].Bookmarks) != 1 {
		t.Fatalf("imported navigation = %+v", nav)
	}

	// Re-importing with the default strategy must not duplicate.
	rec = target.do(http.MethodPost, "/api/v1/admin/import", bundle, nil)
	expectStatus(t, rec, http.StatusOK)
	again := decodeInto[map[string]any](t, rec)
	if again["skipped"].(float64) != 2 {
		t.Errorf("second import skipped = %v, want 2", again["skipped"])
	}
}

func TestImportLegacyFormat(t *testing.T) {
	e := newTestEnv(t)
	expectStatus(t, e.setup("admin", "supersecret"), http.StatusCreated)

	legacy := []byte(`{
      "format": "legacy-v1",
      "categories": [
        {"name": "常用", "links": [
          {"name": "YouTube", "url": "https://youtube.com", "icon": "https://youtube.com/favicon.ico"},
          {"name": "GitHub", "link": "https://github.com"}
        ]}
      ]
    }`)

	rec := e.do(http.MethodPost, "/api/v1/admin/import", legacy, nil)
	expectStatus(t, rec, http.StatusOK)
	result := decodeInto[map[string]any](t, rec)
	if result["categories_imported"].(float64) != 1 || result["bookmarks_imported"].(float64) != 2 {
		t.Errorf("legacy import result = %+v", result)
	}

	rec = e.do(http.MethodGet, "/api/v1/admin/bookmarks", nil, nil)
	list := decodeInto[listResponse[map[string]any]](t, rec)
	if list.Total != 2 {
		t.Errorf("bookmarks after legacy import = %d, want 2", list.Total)
	}
}

func TestImportRejectsInvalidPayload(t *testing.T) {
	e := newTestEnv(t)
	expectStatus(t, e.setup("admin", "supersecret"), http.StatusCreated)

	rec := e.do(http.MethodPost, "/api/v1/admin/import", []byte("not json"), nil)
	expectStatus(t, rec, http.StatusBadRequest)

	rec = e.do(http.MethodPost, "/api/v1/admin/import", map[string]any{"format": "unknown-format"}, nil)
	expectStatus(t, rec, http.StatusBadRequest)
}

// --- backups ---

func TestBackupCreateListAndRestore(t *testing.T) {
	e := newTestEnv(t)
	expectStatus(t, e.setup("admin", "supersecret"), http.StatusCreated)

	expectStatus(t, e.do(http.MethodPost, "/api/v1/admin/categories", map[string]any{"name": "Original"}, nil), http.StatusCreated)

	rec := e.do(http.MethodPost, "/api/v1/admin/backups", nil, nil)
	expectStatus(t, rec, http.StatusCreated)
	backup := decodeInto[map[string]any](t, rec)
	backupID := backup["id"].(string)

	rec = e.do(http.MethodGet, "/api/v1/admin/backups", nil, nil)
	expectStatus(t, rec, http.StatusOK)
	list := decodeInto[struct {
		Items []map[string]any `json:"items"`
	}](t, rec)
	if len(list.Items) == 0 {
		t.Fatal("expected at least one backup")
	}

	// Change data after the snapshot.
	expectStatus(t, e.do(http.MethodPost, "/api/v1/admin/categories", map[string]any{"name": "AfterBackup"}, nil), http.StatusCreated)
	rec = e.do(http.MethodGet, "/api/v1/admin/categories", nil, nil)
	if total := decodeInto[listResponse[map[string]any]](t, rec).Total; total != 2 {
		t.Fatalf("categories before restore = %d, want 2", total)
	}

	// Restore the snapshot.
	rec = e.do(http.MethodPost, "/api/v1/admin/backups/"+backupID+"/restore", nil, nil)
	expectStatus(t, rec, http.StatusOK)

	rec = e.do(http.MethodGet, "/api/v1/admin/categories", nil, nil)
	restored := decodeInto[listResponse[map[string]any]](t, rec)
	if restored.Total != 1 {
		t.Fatalf("categories after restore = %d, want 1", restored.Total)
	}
	if restored.Items[0]["name"] != "Original" {
		t.Errorf("restored category = %v, want Original", restored.Items[0]["name"])
	}

	// Restoring a missing snapshot is a 404.
	rec = e.do(http.MethodPost, "/api/v1/admin/backups/nope.db/restore", nil, nil)
	expectStatus(t, rec, http.StatusNotFound)
}

// --- update check ---

func TestUpdateCheckWithoutRepoConfigured(t *testing.T) {
	e := newTestEnv(t)
	expectStatus(t, e.setup("admin", "supersecret"), http.StatusCreated)

	rec := e.do(http.MethodGet, "/api/v1/admin/update-check", nil, nil)
	expectStatus(t, rec, http.StatusOK)
	info := decodeInto[map[string]any](t, rec)
	if info["update_available"] != false {
		t.Errorf("update_available = %v, want false", info["update_available"])
	}
}

// --- helpers ---

func itoa(v int64) string {
	return strconv.FormatInt(v, 10)
}
