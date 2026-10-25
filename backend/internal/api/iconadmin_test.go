package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

// seedAdminCategory creates a category through the API and returns its ID.
func seedAdminCategory(t *testing.T, e *testEnv, name string) int64 {
	t.Helper()
	rec := e.do(http.MethodPost, "/api/v1/admin/categories", map[string]any{"name": name}, nil)
	expectStatus(t, rec, http.StatusCreated)
	var out struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode category: %v", err)
	}
	return out.ID
}

func seedAdminBookmark(t *testing.T, e *testEnv, categoryID int64, title, url, iconType, iconValue string) int64 {
	t.Helper()
	rec := e.do(http.MethodPost, "/api/v1/admin/bookmarks", map[string]any{
		"category_id": categoryID,
		"title":       title,
		"url":         url,
		"icon_type":   iconType,
		"icon_value":  iconValue,
	}, nil)
	expectStatus(t, rec, http.StatusCreated)
	var out struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode bookmark: %v", err)
	}
	return out.ID
}

func newAdminEnv(t *testing.T) *testEnv {
	t.Helper()
	e := newTestEnv(t)
	expectStatus(t, e.setup("admin", "supersecret"), http.StatusCreated)
	return e
}

func TestIconEndpointsRequireAuth(t *testing.T) {
	e := newTestEnv(t)
	expectStatus(t, e.setup("admin", "supersecret"), http.StatusCreated)
	anon := &testEnv{t: t, handler: e.handler, cfg: e.cfg, db: e.db, cookies: map[string]string{}}

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/admin/icons/summary"},
		{http.MethodGet, "/api/v1/admin/icons/bulk"},
		{http.MethodPost, "/api/v1/admin/icons/bulk"},
		{http.MethodPost, "/api/v1/admin/icons/bulk/cancel"},
		{http.MethodPost, "/api/v1/admin/bookmarks/1/icon/auto"},
	}
	for _, tc := range cases {
		rec := anon.do(tc.method, tc.path, nil, nil)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s status = %d, want 401", tc.method, tc.path, rec.Code)
		}
	}
}

func TestIconSummaryShape(t *testing.T) {
	e := newAdminEnv(t)
	cat := seedAdminCategory(t, e, "分类")
	seedAdminBookmark(t, e, cat, "手动", "https://example.com", "text", "文")
	seedAdminBookmark(t, e, cat, "无图标", "https://example.org", "none", "")

	rec := e.do(http.MethodGet, "/api/v1/admin/icons/summary", nil, nil)
	expectStatus(t, rec, http.StatusOK)

	summary := decodeInto[struct {
		Total   int `json:"total"`
		Manual  int `json:"manual"`
		Auto    int `json:"auto"`
		Favicon int `json:"favicon"`
		Missing int `json:"missing"`
		Failed  int `json:"failed"`
	}](t, rec)

	if summary.Total != 2 {
		t.Errorf("total = %d, want 2", summary.Total)
	}
	if summary.Manual != 1 {
		t.Errorf("manual = %d, want 1", summary.Manual)
	}
	if summary.Missing != 1 {
		t.Errorf("missing = %d, want 1", summary.Missing)
	}
}

func TestAutoIconSkipsManualBookmarks(t *testing.T) {
	e := newAdminEnv(t)
	cat := seedAdminCategory(t, e, "分类")
	id := seedAdminBookmark(t, e, cat, "手动", "https://example.com", "simple", "github")

	rec := e.do(http.MethodPost, "/api/v1/admin/bookmarks/"+itoa64(id)+"/icon/auto", nil, nil)
	expectStatus(t, rec, http.StatusOK)

	outcome := decodeInto[struct {
		OK      bool   `json:"ok"`
		Skipped bool   `json:"skipped"`
		Error   string `json:"icon_error"`
	}](t, rec)
	if outcome.OK || !outcome.Skipped {
		t.Errorf("outcome = %+v, want skipped without ok", outcome)
	}

	// The stored icon must be untouched.
	rec = e.do(http.MethodGet, "/api/v1/admin/bookmarks/"+itoa64(id), nil, nil)
	expectStatus(t, rec, http.StatusOK)
	bookmark := decodeInto[struct {
		IconType  string `json:"icon_type"`
		IconValue string `json:"icon_value"`
	}](t, rec)
	if bookmark.IconType != "simple" || bookmark.IconValue != "github" {
		t.Errorf("icon = (%s, %s), want (simple, github)", bookmark.IconType, bookmark.IconValue)
	}
}

func TestAutoIconFailureIsReportedNotFatal(t *testing.T) {
	e := newAdminEnv(t)
	cat := seedAdminCategory(t, e, "分类")
	// A loopback address: the SSRF guard refuses it, which is a realistic failure.
	id := seedAdminBookmark(t, e, cat, "内网站点", "http://127.0.0.1:9/", "none", "")

	rec := e.do(http.MethodPost, "/api/v1/admin/bookmarks/"+itoa64(id)+"/icon/auto", nil, nil)
	expectStatus(t, rec, http.StatusOK)

	outcome := decodeInto[struct {
		OK      bool   `json:"ok"`
		Error   string `json:"icon_error"`
		Source  string `json:"icon_source"`
		Bookmark struct {
			IconType   string `json:"icon_type"`
			IconStatus string `json:"icon_status"`
			IconError  string `json:"icon_error"`
		} `json:"bookmark"`
	}](t, rec)

	if outcome.OK {
		t.Fatalf("outcome = %+v, want a reported failure", outcome)
	}
	if outcome.Error == "" {
		t.Error("the failure reason must be returned")
	}
	if outcome.Bookmark.IconStatus != "failed" || outcome.Bookmark.IconError == "" {
		t.Errorf("bookmark = %+v, want icon_status=failed with a reason", outcome.Bookmark)
	}
	if outcome.Bookmark.IconType != "none" {
		t.Errorf("icon_type = %q, want the previous value", outcome.Bookmark.IconType)
	}
}

func TestAutoIconMissingBookmarkReturns404(t *testing.T) {
	e := newAdminEnv(t)
	rec := e.do(http.MethodPost, "/api/v1/admin/bookmarks/9999/icon/auto", nil, nil)
	expectStatus(t, rec, http.StatusNotFound)
}

func TestBulkIconJobLifecycle(t *testing.T) {
	e := newAdminEnv(t)
	cat := seedAdminCategory(t, e, "分类")
	// Manual icons only, so the job has nothing to do and completes at once.
	seedAdminBookmark(t, e, cat, "手动", "https://example.com", "text", "文")

	rec := e.do(http.MethodPost, "/api/v1/admin/icons/bulk", nil, nil)
	expectStatus(t, rec, http.StatusAccepted)
	started := decodeInto[struct {
		Job struct {
			ID     string `json:"id"`
			Status string `json:"status"`
			Total  int    `json:"total"`
		} `json:"job"`
	}](t, rec)
	if started.Job.Total != 0 || started.Job.Status != "done" {
		t.Errorf("snapshot = %+v, want an immediately finished empty job", started.Job)
	}

	rec = e.do(http.MethodGet, "/api/v1/admin/icons/bulk", nil, nil)
	expectStatus(t, rec, http.StatusOK)
	current := decodeInto[struct {
		Job struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"job"`
	}](t, rec)
	if current.Job.ID != started.Job.ID {
		t.Errorf("status id = %q, want %q", current.Job.ID, started.Job.ID)
	}

	rec = e.do(http.MethodPost, "/api/v1/admin/icons/bulk/cancel", nil, nil)
	expectStatus(t, rec, http.StatusOK)
	canceled := decodeInto[struct {
		Canceled bool `json:"canceled"`
	}](t, rec)
	if canceled.Canceled {
		t.Error("canceling a finished job must report canceled=false")
	}
}

func TestBulkIconStatusIsNullBeforeAnyRun(t *testing.T) {
	e := newAdminEnv(t)
	rec := e.do(http.MethodGet, "/api/v1/admin/icons/bulk", nil, nil)
	expectStatus(t, rec, http.StatusOK)
	body := decodeInto[struct {
		Job *struct {
			ID string `json:"id"`
		} `json:"job"`
	}](t, rec)
	if body.Job != nil {
		t.Errorf("job = %+v, want null before any job ran", body.Job)
	}
}

func itoa64(v int64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for v > 0 {
		pos--
		buf[pos] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[pos:])
}
