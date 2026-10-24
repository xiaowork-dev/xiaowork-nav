package service

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/migrations"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/models"
)

var backupIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*\.db$`)

// filesSuffix is appended to a snapshot's filename for its icon directory.
const filesSuffix = ".files"

// BackupService creates, lists and restores SQLite snapshots.
//
// A snapshot consists of:
//   - `<id>`         : a consistent database copy produced with `VACUUM INTO`;
//   - `<id>.files/`  : a copy of the uploads (icon) directory, when non-empty.
//
// VACUUM INTO yields a consistent copy without stopping the service.
type BackupService struct {
	db         *sql.DB
	backupsDir string
	uploadsDir string
	appVersion string
	keep       int
}

// NewBackupService builds a BackupService. keep <= 0 disables pruning.
func NewBackupService(db *sql.DB, backupsDir, uploadsDir, appVersion string, keep int) *BackupService {
	return &BackupService{
		db:         db,
		backupsDir: backupsDir,
		uploadsDir: uploadsDir,
		appVersion: appVersion,
		keep:       keep,
	}
}

// List returns known snapshots, newest first.
func (b *BackupService) List() ([]models.Backup, error) {
	entries, err := os.ReadDir(b.backupsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []models.Backup{}, nil
		}
		return nil, fmt.Errorf("read backups dir: %w", err)
	}

	out := make([]models.Backup, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".db") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, models.Backup{
			ID:        e.Name(),
			CreatedAt: info.ModTime().UTC().Format(time.RFC3339),
			Size:      info.Size(),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	return out, nil
}

// Create produces a new snapshot and returns its metadata.
func (b *BackupService) Create(ctx context.Context) (*models.Backup, error) {
	schemaVersion, err := migrations.CurrentVersion(b.db)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(b.backupsDir, 0o750); err != nil {
		return nil, fmt.Errorf("create backups dir: %w", err)
	}

	stamp := time.Now().UTC().Format("20060102-150405")
	id := fmt.Sprintf("backup-%s-%s.db", b.appVersion, stamp)
	path := filepath.Join(b.backupsDir, id)
	if _, err := os.Stat(path); err == nil {
		// Same-second collision: add a suffix.
		id = fmt.Sprintf("backup-%s-%s-%d.db", b.appVersion, stamp, time.Now().UnixNano()%100000)
		path = filepath.Join(b.backupsDir, id)
	}

	// VACUUM INTO writes a consistent copy and must not target an existing file.
	if _, err := b.db.ExecContext(ctx, `VACUUM INTO ?`, path); err != nil {
		return nil, fmt.Errorf("vacuum into %q: %w", path, err)
	}

	if err := copyDir(b.uploadsDir, path+filesSuffix); err != nil {
		return nil, fmt.Errorf("snapshot uploads: %w", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat backup: %w", err)
	}

	if b.keep > 0 {
		_ = b.prune(b.keep)
	}

	return &models.Backup{
		ID:            id,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		AppVersion:    b.appVersion,
		SchemaVersion: schemaVersion,
		Size:          info.Size(),
	}, nil
}

func (b *BackupService) prune(keep int) error {
	list, err := b.List()
	if err != nil {
		return err
	}
	if len(list) <= keep {
		return nil
	}
	for _, old := range list[keep:] {
		path := filepath.Join(b.backupsDir, old.ID)
		_ = os.Remove(path)
		_ = os.RemoveAll(path + filesSuffix)
	}
	return nil
}

// Restore replaces the current data with a snapshot.
//
// It refuses snapshots whose schema version differs from the running one, and
// takes a safety snapshot of the current data before applying anything.
// User accounts and sessions are preserved; icons are restored from the
// snapshot's `.files` directory when present.
func (b *BackupService) Restore(ctx context.Context, id string) error {
	path, err := b.resolve(id)
	if err != nil {
		return err
	}

	current, err := migrations.CurrentVersion(b.db)
	if err != nil {
		return err
	}

	// The safety snapshot must be taken before the connection below is
	// checked out, because the pool holds a single connection.
	if _, err := b.Create(ctx); err != nil {
		return fmt.Errorf("create safety snapshot: %w", err)
	}

	conn, err := b.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Close()

	quoted := "'" + strings.ReplaceAll(path, "'", "''") + "'"

	if _, err := conn.ExecContext(ctx, `ATTACH DATABASE `+quoted+` AS snap`); err != nil {
		return fmt.Errorf("attach snapshot: %w", err)
	}
	defer func() { _, _ = conn.ExecContext(context.Background(), `DETACH DATABASE snap`) }()

	var snapshotVersion sql.NullInt64
	if err := conn.QueryRowContext(ctx, `SELECT MAX(version) FROM snap.schema_migrations`).Scan(&snapshotVersion); err != nil {
		return Validation("validation_failed", "快照文件无效或不包含数据库结构")
	}
	if int(snapshotVersion.Int64) != current {
		return Validation("validation_failed",
			fmt.Sprintf("快照结构版本(%d)与当前版本(%d)不一致，拒绝恢复", snapshotVersion.Int64, current))
	}

	if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		return fmt.Errorf("disable foreign keys: %w", err)
	}
	defer func() { _, _ = conn.ExecContext(context.Background(), `PRAGMA foreign_keys = ON`) }()

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin restore: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	steps := []string{
		`DELETE FROM bookmarks`,
		`DELETE FROM categories`,
		`DELETE FROM site_settings`,
		`DELETE FROM uploads`,
		`INSERT INTO categories SELECT * FROM snap.categories`,
		`INSERT INTO bookmarks SELECT * FROM snap.bookmarks`,
		`INSERT INTO site_settings SELECT * FROM snap.site_settings`,
		`INSERT INTO uploads SELECT * FROM snap.uploads`,
	}
	for _, stmt := range steps {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("restore step %q: %w", stmt, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit restore: %w", err)
	}

	if dirExists(path + filesSuffix) {
		if err := replaceDir(b.uploadsDir, path+filesSuffix); err != nil {
			return fmt.Errorf("restore uploads: %w", err)
		}
	}
	return nil
}

func (b *BackupService) resolve(id string) (string, error) {
	if !backupIDPattern.MatchString(id) || strings.Contains(id, "..") {
		return "", NotFound("备份不存在")
	}
	path := filepath.Join(b.backupsDir, id)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return "", NotFound("备份不存在")
	}
	return path, nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// copyDir copies the contents of src into dst (created if missing).
func copyDir(src, dst string) error {
	if !dirExists(src) {
		return nil
	}
	if err := os.MkdirAll(dst, 0o750); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if err := copyFile(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// replaceDir makes dst contain exactly the files of src.
func replaceDir(dst, src string) error {
	if err := os.RemoveAll(dst); err != nil {
		return err
	}
	return copyDir(src, dst)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o640)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
