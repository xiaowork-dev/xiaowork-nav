package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/migrations"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/models"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/store"
)

func TestHasFlagAndFlagValue(t *testing.T) {
	args := []string{"--db-info", "--db", "/data/app.db"}
	if !hasFlag(args, "--db-info") {
		t.Error("hasFlag(--db-info) = false, want true")
	}
	if hasFlag(args, "--nope") {
		t.Error("hasFlag(--nope) = true, want false")
	}
	if got := flagValue(args, "--db"); got != "/data/app.db" {
		t.Errorf("flagValue(--db) = %q", got)
	}
	if got := flagValue([]string{"--db=/tmp/x.db"}, "--db"); got != "/tmp/x.db" {
		t.Errorf("flagValue(--db=...) = %q", got)
	}
	if got := flagValue(args, "--missing"); got != "" {
		t.Errorf("flagValue(--missing) = %q, want empty", got)
	}
}

// captureStdout runs fn while capturing everything written to stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		buf := make([]byte, 0, 4096)
		tmp := make([]byte, 512)
		for {
			n, err := r.Read(tmp)
			buf = append(buf, tmp[:n]...)
			if err != nil {
				break
			}
		}
		done <- string(buf)
	}()
	fn()
	_ = w.Close()
	os.Stdout = old
	return <-done
}

func TestRunDBInfoReportsSchemaAndCounts(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "app.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	if _, err := migrations.Apply(db); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	st := store.New(db)
	if _, err := st.CreateCategory(t.Context(), &models.Category{Name: "示例", IsVisible: true}); err != nil {
		t.Fatalf("CreateCategory() error = %v", err)
	}
	_ = db.Close()

	out := captureStdout(t, func() {
		if code := runDBInfo([]string{"--db", dbPath}); code != 0 {
			t.Errorf("runDBInfo() = %d, want 0", code)
		}
	})

	for _, want := range []string{"exists=true", "schema_version=", "categories=", "bookmarks=", "uploads="} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
		}
	}
}

func TestRunDBInfoMissingFile(t *testing.T) {
	out := captureStdout(t, func() {
		if code := runDBInfo([]string{"--db", filepath.Join(t.TempDir(), "absent.db")}); code != 0 {
			t.Errorf("runDBInfo() = %d, want 0", code)
		}
	})
	if !strings.Contains(out, "exists=false") || !strings.Contains(out, "schema_version=0") {
		t.Errorf("unexpected output for a missing database:\n%s", out)
	}
}
