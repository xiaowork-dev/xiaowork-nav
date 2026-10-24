package api

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/config"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/migrations"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/store"
	"gopkg.in/yaml.v3"
)

func newServerForRoutes(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{
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
	return New(Deps{Config: cfg, DB: db, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
}

// TestRoutesMatchOpenAPIContract guards against drift between the registered
// HTTP routes and docs/openapi/openapi.yaml. Any interface change must update
// both sides.
func TestRoutesMatchOpenAPIContract(t *testing.T) {
	docPath := filepath.Join("..", "..", "..", "docs", "openapi", "openapi.yaml")
	raw, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("read OpenAPI contract: %v", err)
	}

	var doc struct {
		Servers []struct {
			URL string `yaml:"url"`
		} `yaml:"servers"`
		Paths map[string]map[string]any `yaml:"paths"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse OpenAPI contract: %v", err)
	}
	if len(doc.Paths) == 0 {
		t.Fatal("OpenAPI contract declares no paths")
	}

	// Paths in the contract are relative to the server base URL.
	baseURL := "/api/v1"
	if len(doc.Servers) > 0 && doc.Servers[0].URL != "" {
		baseURL = doc.Servers[0].URL
	}

	httpMethods := map[string]bool{
		"get": true, "post": true, "put": true, "patch": true,
		"delete": true, "head": true, "options": true,
	}

	documented := map[string]bool{}
	for path, operations := range doc.Paths {
		for key, value := range operations {
			method := strings.ToLower(key)
			if !httpMethods[method] {
				continue
			}
			// An operation may override the server base (e.g. /healthz).
			opBase := baseURL
			if op, ok := value.(map[string]any); ok {
				if servers, ok := op["servers"].([]any); ok && len(servers) > 0 {
					if first, ok := servers[0].(map[string]any); ok {
						if url, ok := first["url"].(string); ok && url != "" {
							opBase = url
						}
					}
				}
			}
			documented[strings.ToUpper(method)+" "+joinURL(opBase, path)] = true
		}
	}

	registered := map[string]bool{}
	for _, rt := range newServerForRoutes(t).routes() {
		registered[rt.Method+" "+rt.Pattern] = true
	}

	var undocumented, unimplemented []string
	for key := range registered {
		if !documented[key] {
			undocumented = append(undocumented, key)
		}
	}
	for key := range documented {
		if !registered[key] {
			unimplemented = append(unimplemented, key)
		}
	}

	sort.Strings(undocumented)
	sort.Strings(unimplemented)

	for _, key := range undocumented {
		t.Errorf("route %q is registered but not documented in openapi.yaml", key)
	}
	for _, key := range unimplemented {
		t.Errorf("operation %q is documented in openapi.yaml but has no route", key)
	}
	if len(registered) != len(documented) {
		t.Logf("registered=%d documented=%d", len(registered), len(documented))
	}
}

// joinURL joins a server base URL and a path, avoiding duplicate slashes.
func joinURL(base, path string) string {
	if base == "" || base == "/" {
		return "/" + strings.TrimPrefix(path, "/")
	}
	return strings.TrimSuffix(base, "/") + "/" + strings.TrimPrefix(path, "/")
}
