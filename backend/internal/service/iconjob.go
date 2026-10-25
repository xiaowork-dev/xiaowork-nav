package service

import (
	"context"
	"sync"
	"time"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/models"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/store"
)

// Manual icon types are never touched by the automatic fetcher: a user-set icon
// always wins (requirement 2.2).
var manualIconTypes = map[string]bool{
	"url":    true,
	"upload": true,
	"simple": true,
	"text":   true,
}

// IsManualIconType reports whether an icon type was chosen by the user.
func IsManualIconType(iconType string) bool {
	return manualIconTypes[iconType]
}

// Bulk job defaults.
const (
	defaultBulkLimit      = 500
	maxBulkLimit          = 2000
	defaultRefreshDays    = 30
	bulkWorkerCount       = 3
	bulkRequestDelay      = 150 * time.Millisecond
	maxRecordedJobErrors  = 50
	maxIconErrorLength    = 200
)

// FillOutcome is the result of fetching an icon for one bookmark.
//
// A failed fetch is not an API error: the bookmark keeps its previous icon and
// the reason is reported so the admin UI can show it.
type FillOutcome struct {
	Bookmark *models.Bookmark `json:"bookmark"`
	OK       bool             `json:"ok"`
	Skipped  bool             `json:"skipped"`
	Source   string           `json:"icon_source,omitempty"`
	Error    string           `json:"icon_error,omitempty"`
}

// BulkOptions configures a bulk icon completion run.
type BulkOptions struct {
	OnlyMissing bool
	Force       bool
	Limit       int
	RefreshDays int
}

// JobStatus values.
const (
	JobStatusRunning  = "running"
	JobStatusDone     = "done"
	JobStatusCanceled = "canceled"
)

// JobItem is one entry of the job's failure report.
type JobItem struct {
	BookmarkID int64  `json:"bookmark_id"`
	Title      string `json:"title,omitempty"`
	OK         bool   `json:"ok"`
	Source     string `json:"icon_source,omitempty"`
	Error      string `json:"icon_error,omitempty"`
}

// JobSnapshot is the observable state of a bulk job.
type JobSnapshot struct {
	ID         string    `json:"id"`
	Status     string    `json:"status"`
	Total      int       `json:"total"`
	Done       int       `json:"done"`
	Succeeded  int       `json:"succeeded"`
	Failed     int       `json:"failed"`
	Skipped    int       `json:"skipped"`
	StartedAt  string    `json:"started_at"`
	FinishedAt string    `json:"finished_at,omitempty"`
	Errors     []JobItem `json:"errors,omitempty"`
}

// IconJob is the mutable state of a running bulk job.
type IconJob struct {
	id     string
	cancel context.CancelFunc
	ctx    context.Context

	mu        sync.Mutex
	status    string
	total     int
	done      int
	succeeded int
	failed    int
	skipped   int
	startedAt string
	finishedAt string
	errors    []JobItem
}

func (j *IconJob) record(item JobItem) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.done++
	switch {
	case item.OK:
		j.succeeded++
	case item.Error == "":
		j.skipped++
	default:
		j.failed++
		if len(j.errors) < maxRecordedJobErrors {
			j.errors = append(j.errors, item)
		}
	}
}

// Snapshot returns a copy of the job state.
func (j *IconJob) Snapshot() JobSnapshot {
	j.mu.Lock()
	defer j.mu.Unlock()
	snap := JobSnapshot{
		ID:         j.id,
		Status:     j.status,
		Total:      j.total,
		Done:       j.done,
		Succeeded:  j.succeeded,
		Failed:     j.failed,
		Skipped:    j.skipped,
		StartedAt:  j.startedAt,
		FinishedAt: j.finishedAt,
	}
	if len(j.errors) > 0 {
		snap.Errors = append([]JobItem(nil), j.errors...)
	}
	return snap
}

func (j *IconJob) finish(status string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.status = status
	j.finishedAt = models.NowUTC()
}

// IconJobManager runs at most one bulk icon job at a time.
//
// The job is intentionally in-memory: it is a bounded, admin-triggered
// operation whose progress is only meaningful while it runs. A restart loses
// the progress but never the data, and the admin can simply run it again.
type IconJobManager struct {
	mu      sync.Mutex
	current *IconJob
	seq     int
	workers int
	delay   time.Duration
}

// NewIconJobManager builds a job manager.
func NewIconJobManager() *IconJobManager {
	return &IconJobManager{workers: bulkWorkerCount, delay: bulkRequestDelay}
}

// begin creates a job, or returns nil when one is already running.
func (m *IconJobManager) begin(total int) *IconJob {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current != nil && m.current.Snapshot().Status == JobStatusRunning {
		return nil
	}
	m.seq++
	ctx, cancel := context.WithCancel(context.Background())
	job := &IconJob{
		id:        "iconjob-" + time.Now().UTC().Format("20060102T150405") + "-" + itoa(m.seq),
		cancel:    cancel,
		ctx:       ctx,
		status:    JobStatusRunning,
		total:     total,
		startedAt: models.NowUTC(),
	}
	m.current = job
	return job
}

// Current returns the most recent job snapshot, or nil when none ever ran.
func (m *IconJobManager) Current() *JobSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == nil {
		return nil
	}
	snap := m.current.Snapshot()
	return &snap
}

// Cancel stops the running job; it reports whether a job was running.
func (m *IconJobManager) Cancel() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == nil || m.current.Snapshot().Status != JobStatusRunning {
		return false
	}
	m.current.cancel()
	return true
}

// IconFillService fills bookmark icons from the network and caches them.
type IconFillService struct {
	icons   *IconService
	store   *store.Store
	catalog *CatalogService
	manager *IconJobManager
}

// NewIconFillService builds an IconFillService.
func NewIconFillService(icons *IconService, st *store.Store, catalog *CatalogService) *IconFillService {
	return &IconFillService{icons: icons, store: st, catalog: catalog, manager: NewIconJobManager()}
}

// FillBookmark fetches an icon for a single bookmark and persists the outcome.
func (s *IconFillService) FillBookmark(ctx context.Context, id int64, force bool) (*FillOutcome, error) {
	bookmark, err := s.catalog.GetBookmark(ctx, id)
	if err != nil {
		return nil, err
	}
	if IsManualIconType(bookmark.IconType) {
		return &FillOutcome{
			Bookmark: bookmark,
			Skipped:  true,
			Error:    "该网站已手动设置图标，自动获取不会覆盖",
		}, nil
	}
	return s.fill(ctx, store.IconFillCandidate{
		ID:            bookmark.ID,
		URL:           bookmark.URL,
		IconType:      bookmark.IconType,
		IconValue:     bookmark.IconValue,
		IconStatus:    bookmark.IconStatus,
		IconCheckedAt: bookmark.IconCheckedAt,
	}, force), nil
}

// fill performs the fetch and writes the bookkeeping columns.
func (s *IconFillService) fill(ctx context.Context, c store.IconFillCandidate, force bool) *FillOutcome {
	now := models.NowUTC()
	result, err := s.icons.FetchIcon(ctx, c.URL, force)
	if err != nil {
		reason := truncate(err.Error(), maxIconErrorLength)
		updated, updateErr := s.store.SetBookmarkIcon(ctx, c.ID, store.IconBookkeeping{
			IconType:      c.IconType,
			IconValue:     c.IconValue,
			IconStatus:    models.IconStatusFailed,
			IconError:     reason,
			IconCheckedAt: now,
		})
		if updateErr != nil {
			return &FillOutcome{OK: false, Error: reason}
		}
		return &FillOutcome{Bookmark: updated, OK: false, Error: reason}
	}

	updated, err := s.store.SetBookmarkIcon(ctx, c.ID, store.IconBookkeeping{
		IconType:      result.IconType,
		IconValue:     result.IconValue,
		IconSource:    result.Source,
		IconStatus:    models.IconStatusOK,
		IconCheckedAt: now,
	})
	if err != nil {
		return &FillOutcome{OK: false, Error: truncate(err.Error(), maxIconErrorLength)}
	}

	// Drop the previous auto-cached file when nothing references it any more.
	if c.IconValue != "" && c.IconValue != result.IconValue {
		_ = s.icons.DeleteAutoUploadIfUnreferenced(ctx, c.IconValue)
	}
	return &FillOutcome{Bookmark: updated, OK: true, Source: result.Source}
}

// Summary reports icon coverage for the admin dashboard.
func (s *IconFillService) Summary(ctx context.Context) (store.IconSummary, error) {
	return s.store.CountIconSummary(ctx)
}

// StartBulk launches the bulk completion job.
func (s *IconFillService) StartBulk(opts BulkOptions) (*JobSnapshot, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = defaultBulkLimit
	}
	if limit > maxBulkLimit {
		limit = maxBulkLimit
	}
	refreshDays := opts.RefreshDays
	if refreshDays <= 0 {
		refreshDays = defaultRefreshDays
	}

	checkedAfter := ""
	if opts.OnlyMissing {
		checkedAfter = time.Now().UTC().AddDate(0, 0, -refreshDays).Format(time.RFC3339)
	}
	items, err := s.store.ListIconFillCandidates(context.Background(), opts.OnlyMissing, checkedAfter, limit)
	if err != nil {
		return nil, err
	}

	job := s.manager.begin(len(items))
	if job == nil {
		return nil, Conflict("job_running", "已有批量获取任务正在运行，请稍后再试")
	}
	if len(items) == 0 {
		job.finish(JobStatusDone)
		snapshot := job.Snapshot()
		return &snapshot, nil
	}

	go s.runBulk(job, items, opts.Force)
	snapshot := job.Snapshot()
	return &snapshot, nil
}

// BulkStatus returns the current or most recent job snapshot.
func (s *IconFillService) BulkStatus() *JobSnapshot {
	return s.manager.Current()
}

// CancelBulk cancels the running job; it reports whether one was running.
func (s *IconFillService) CancelBulk() bool {
	return s.manager.Cancel()
}

// runBulk processes candidates with a bounded worker pool and a small delay
// between requests, which keeps the load on remote sites predictable.
func (s *IconFillService) runBulk(job *IconJob, items []store.IconFillCandidate, force bool) {
	ctx := job.ctx
	sem := make(chan struct{}, s.manager.workers)
	var wg sync.WaitGroup

dispatch:
	for _, item := range items {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			break dispatch
		}
		wg.Add(1)
		go func(c store.IconFillCandidate) {
			defer wg.Done()
			defer func() { <-sem }()
			outcome := s.fill(ctx, c, force)
			job.record(JobItem{
				BookmarkID: c.ID,
				OK:         outcome.OK,
				Source:     outcome.Source,
				Error:      outcome.Error,
			})
			select {
			case <-time.After(s.manager.delay):
			case <-ctx.Done():
			}
		}(item)
	}

	wg.Wait()
	if ctx.Err() != nil {
		job.finish(JobStatusCanceled)
		return
	}
	job.finish(JobStatusDone)
}

// itoa avoids pulling strconv in for a single call site.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	negative := n < 0
	if negative {
		n = -n
	}
	var buf [20]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if negative {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
