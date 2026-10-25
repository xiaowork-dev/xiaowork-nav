package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/models"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/store"
)

func seedCategory(t *testing.T, st *store.Store) *models.Category {
	t.Helper()
	cat, err := st.CreateCategory(context.Background(), &models.Category{Name: "测试", IsVisible: true})
	if err != nil {
		t.Fatalf("CreateCategory() error = %v", err)
	}
	return cat
}

func seedBookmark(t *testing.T, st *store.Store, categoryID int64, title, url, iconType, iconValue string) *models.Bookmark {
	t.Helper()
	b, err := st.CreateBookmark(context.Background(), &models.Bookmark{
		CategoryID: categoryID,
		Title:      title,
		URL:        url,
		IconType:   iconType,
		IconValue:  iconValue,
		IsVisible:  true,
	})
	if err != nil {
		t.Fatalf("CreateBookmark(%q) error = %v", title, err)
	}
	return b
}

func newFillTestService(t *testing.T) (*IconFillService, *store.Store, *IconService) {
	t.Helper()
	st := newTestStore(t)
	icons := NewIconService(st, t.TempDir())
	icons.client = &http.Client{}
	catalog := NewCatalogService(st, NewSettingsService(st))
	return NewIconFillService(icons, st, catalog), st, icons
}

// faviconSite serves a page plus a usable favicon.
func faviconSite(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<html><head></head></html>`))
		case "/favicon.ico":
			w.Header().Set("Content-Type", "image/x-icon")
			_, _ = w.Write(icoWithSize(32))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func waitForJob(t *testing.T, svc *IconFillService, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		snap := svc.BulkStatus()
		if snap != nil && snap.Status != JobStatusRunning {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("bulk job did not finish in time")
}

// --- manual icon protection ---

func TestFillBookmarkNeverOverwritesManualIcons(t *testing.T) {
	svc, st, _ := newFillTestService(t)
	cat := seedCategory(t, st)

	manual := map[string]string{
		"url":    "https://cdn.example.com/logo.png",
		"upload": UploadURLPrefix + "abc123.png",
		"simple": "github",
		"text":   "延迟",
	}
	for iconType, iconValue := range manual {
		b := seedBookmark(t, st, cat.ID, "手动-"+iconType, "https://example.com", iconType, iconValue)

		outcome, err := svc.FillBookmark(context.Background(), b.ID, true)
		if err != nil {
			t.Fatalf("FillBookmark(%s) error = %v", iconType, err)
		}
		if !outcome.Skipped || outcome.OK {
			t.Errorf("%s: outcome = %+v, want skipped", iconType, outcome)
		}

		after, err := st.GetBookmark(context.Background(), b.ID)
		if err != nil {
			t.Fatalf("GetBookmark() error = %v", err)
		}
		if after.IconType != iconType || after.IconValue != iconValue {
			t.Errorf("%s: icon changed to (%s, %s); manual icons must be preserved",
				iconType, after.IconType, after.IconValue)
		}
	}
}

// --- success and failure bookkeeping ---

func TestFillBookmarkStoresIconAndSource(t *testing.T) {
	svc, st, _ := newFillTestService(t)
	cat := seedCategory(t, st)
	site := faviconSite(t)
	b := seedBookmark(t, st, cat.ID, "示例", site.URL, "none", "")

	outcome, err := svc.FillBookmark(context.Background(), b.ID, false)
	if err != nil {
		t.Fatalf("FillBookmark() error = %v", err)
	}
	if !outcome.OK {
		t.Fatalf("outcome = %+v, want ok", outcome)
	}
	if outcome.Source != models.IconSourceFaviconICO {
		t.Errorf("source = %q, want %q", outcome.Source, models.IconSourceFaviconICO)
	}
	if outcome.Bookmark.IconType != "auto" || outcome.Bookmark.IconStatus != models.IconStatusOK {
		t.Errorf("bookmark = %+v, want auto/ok", outcome.Bookmark)
	}
	if outcome.Bookmark.IconCheckedAt == "" {
		t.Error("icon_checked_at must be recorded")
	}

	summary, err := svc.Summary(context.Background())
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	if summary.Auto != 1 || summary.Total != 1 {
		t.Errorf("summary = %+v, want 1 auto of 1", summary)
	}
}

func TestFillBookmarkRecordsFailureAndKeepsPreviousIcon(t *testing.T) {
	svc, st, _ := newFillTestService(t)
	cat := seedCategory(t, st)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	b := seedBookmark(t, st, cat.ID, "坏站点", srv.URL, "none", "")

	outcome, err := svc.FillBookmark(context.Background(), b.ID, false)
	if err != nil {
		t.Fatalf("FillBookmark() error = %v", err)
	}
	if outcome.OK {
		t.Fatalf("outcome = %+v, want a failure", outcome)
	}
	if outcome.Error == "" {
		t.Error("a failure must carry a reason for the admin UI")
	}

	after, err := st.GetBookmark(context.Background(), b.ID)
	if err != nil {
		t.Fatalf("GetBookmark() error = %v", err)
	}
	if after.IconStatus != models.IconStatusFailed {
		t.Errorf("icon_status = %q, want failed", after.IconStatus)
	}
	if after.IconError == "" {
		t.Error("icon_error must be persisted")
	}
	if after.IconType != "none" || after.IconValue != "" {
		t.Errorf("failed fetch changed the icon to (%s, %s)", after.IconType, after.IconValue)
	}
}

func TestFillBookmarkMissingReturnsNotFound(t *testing.T) {
	svc, _, _ := newFillTestService(t)
	if _, err := svc.FillBookmark(context.Background(), 4242, false); err == nil {
		t.Fatal("FillBookmark() on a missing bookmark must fail")
	}
}

// --- bulk job ---

func TestBulkJobFillsEligibleBookmarksOnly(t *testing.T) {
	svc, st, _ := newFillTestService(t)
	cat := seedCategory(t, st)
	site := faviconSite(t)

	seedBookmark(t, st, cat.ID, "自动一", site.URL, "none", "")
	seedBookmark(t, st, cat.ID, "自动二", site.URL, "favicon", "")
	seedBookmark(t, st, cat.ID, "自动三", site.URL, "auto", "")
	manual := seedBookmark(t, st, cat.ID, "手动", "https://example.com", "text", "文")

	snapshot, err := svc.StartBulk(BulkOptions{OnlyMissing: true})
	if err != nil {
		t.Fatalf("StartBulk() error = %v", err)
	}
	if snapshot.Total != 3 {
		t.Errorf("total = %d, want 3 (manual icons are excluded)", snapshot.Total)
	}

	waitForJob(t, svc, 20*time.Second)
	final := svc.BulkStatus()
	if final.Status != JobStatusDone {
		t.Fatalf("job status = %q, want done", final.Status)
	}
	if final.Done != 3 || final.Succeeded != 3 || final.Failed != 0 {
		t.Errorf("job = %+v, want 3/3 succeeded", final)
	}

	after, err := st.GetBookmark(context.Background(), manual.ID)
	if err != nil {
		t.Fatalf("GetBookmark() error = %v", err)
	}
	if after.IconType != "text" || after.IconValue != "文" {
		t.Errorf("bulk job changed a manual icon to (%s, %s)", after.IconType, after.IconValue)
	}
}

func TestBulkJobRejectsConcurrentRuns(t *testing.T) {
	svc, st, _ := newFillTestService(t)
	cat := seedCategory(t, st)

	// A slow site keeps the first job running long enough to collide.
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-time.After(5 * time.Second):
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	defer close(release)

	seedBookmark(t, st, cat.ID, "慢站点", srv.URL, "none", "")

	if _, err := svc.StartBulk(BulkOptions{}); err != nil {
		t.Fatalf("StartBulk() error = %v", err)
	}
	if _, err := svc.StartBulk(BulkOptions{}); err == nil {
		t.Fatal("a second concurrent bulk job must be rejected")
	}

	if !svc.CancelBulk() {
		t.Error("CancelBulk() must report that it canceled a running job")
	}
	waitForJob(t, svc, 20*time.Second)
	if final := svc.BulkStatus(); final.Status != JobStatusCanceled {
		t.Errorf("job status = %q, want canceled", final.Status)
	}
}

func TestBulkJobWithNoCandidatesCompletesImmediately(t *testing.T) {
	svc, st, _ := newFillTestService(t)
	cat := seedCategory(t, st)
	seedBookmark(t, st, cat.ID, "手动", "https://example.com", "simple", "github")

	snapshot, err := svc.StartBulk(BulkOptions{})
	if err != nil {
		t.Fatalf("StartBulk() error = %v", err)
	}
	if snapshot.Total != 0 || snapshot.Status != JobStatusDone {
		t.Errorf("snapshot = %+v, want an immediately completed empty job", snapshot)
	}
}

// --- cached file cleanup ---

func TestDeleteAutoUploadIfUnreferenced(t *testing.T) {
	_, st, icons := newFillTestService(t)
	cat := seedCategory(t, st)
	ctx := context.Background()

	autoUpload, err := icons.saveUpload(ctx, solidPNG(t, 32), "icon.png", "https://example.com/favicon.ico", models.UploadOriginAuto)
	if err != nil {
		t.Fatalf("saveUpload() error = %v", err)
	}
	manualUpload, err := icons.SaveUpload(ctx, solidPNG(t, 32), "manual.png")
	if err != nil {
		t.Fatalf("SaveUpload() error = %v", err)
	}

	autoURL := UploadURL(autoUpload.Filename)
	manualURL := UploadURL(manualUpload.Filename)

	// Two bookmarks share the cached icon: it must survive cleanup.
	first := seedBookmark(t, st, cat.ID, "共享一", "https://example.com", "auto", autoURL)
	seedBookmark(t, st, cat.ID, "共享二", "https://example.com", "auto", autoURL)

	if err := icons.DeleteAutoUploadIfUnreferenced(ctx, autoURL); err != nil {
		t.Fatalf("DeleteAutoUploadIfUnreferenced() error = %v", err)
	}
	if _, err := st.GetUploadByFilename(ctx, autoUpload.Filename); err != nil {
		t.Error("a cached icon still referenced by a bookmark must not be deleted")
	}

	// Once the last reference is gone it is cleaned up.
	if _, err := st.SetBookmarkIcon(ctx, first.ID, store.IconBookkeeping{IconType: "none"}); err != nil {
		t.Fatalf("SetBookmarkIcon() error = %v", err)
	}
	// The second bookmark still references it, so it stays.
	if err := icons.DeleteAutoUploadIfUnreferenced(ctx, autoURL); err != nil {
		t.Fatalf("DeleteAutoUploadIfUnreferenced() error = %v", err)
	}
	if _, err := st.GetUploadByFilename(ctx, autoUpload.Filename); err != nil {
		t.Error("a cached icon still referenced by another bookmark must not be deleted")
	}

	// A user upload is never removed by the automatic cleanup.
	if err := icons.DeleteAutoUploadIfUnreferenced(ctx, manualURL); err != nil {
		t.Fatalf("DeleteAutoUploadIfUnreferenced() error = %v", err)
	}
	if _, err := st.GetUploadByFilename(ctx, manualUpload.Filename); err != nil {
		t.Error("user uploads must never be deleted by the automatic cleanup")
	}
}
