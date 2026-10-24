// Package api wires HTTP routing, middleware, and handlers.
//
// Handlers translate HTTP requests into service calls; they must only expose
// the fields defined in docs/openapi/openapi.yaml.
package api

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/config"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/migrations"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/service"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/store"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/version"
)

// Deps are the dependencies required to build the HTTP handler.
type Deps struct {
	Config *config.Config
	DB     *sql.DB
	Logger *slog.Logger
	// Static serves the built frontend (may be nil in API-only mode).
	Static http.Handler
}

// Server holds shared handler dependencies.
type Server struct {
	cfg      *config.Config
	db       *sql.DB
	log      *slog.Logger
	static   http.Handler
	store    *store.Store
	auth     *service.AuthService
	settings *service.SettingsService
	catalog  *service.CatalogService
	icons    *service.IconService
	transfer *service.TransferService
	backup   *service.BackupService
}

// New builds a Server and its services.
func New(deps Deps) *Server {
	st := store.New(deps.DB)
	settings := service.NewSettingsService(st)
	return &Server{
		cfg:      deps.Config,
		db:       deps.DB,
		log:      deps.Logger,
		static:   deps.Static,
		store:    st,
		auth:     service.NewAuthService(st, deps.Config),
		settings: settings,
		catalog:  service.NewCatalogService(st, settings),
		icons:    service.NewIconService(st, deps.Config.UploadsDir),
		transfer: service.NewTransferService(st),
		backup:   service.NewBackupService(deps.DB, deps.Config.BackupsDir, deps.Config.UploadsDir, version.Version, deps.Config.BackupKeep),
	}
}

// Handler returns the fully wired HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// System
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /api/v1/version", s.handleVersion)

	// Auth
	mux.HandleFunc("GET /api/v1/auth/status", s.optionalAuth(s.handleAuthStatus))
	mux.HandleFunc("POST /api/v1/setup", s.handleSetup)
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/logout", s.requireAuth(s.handleLogout))
	mux.HandleFunc("GET /api/v1/auth/me", s.requireAuth(s.handleMe))
	mux.HandleFunc("PUT /api/v1/auth/password", s.requireAuth(s.handleChangePassword))

	// Public
	mux.HandleFunc("GET /api/v1/public/navigation", s.publicRead(s.handlePublicNavigation))
	mux.HandleFunc("GET /api/v1/public/categories", s.publicRead(s.handlePublicCategories))
	mux.HandleFunc("GET /api/v1/public/categories/{id}/bookmarks", s.publicRead(s.handlePublicCategoryBookmarks))
	mux.HandleFunc("GET /api/v1/public/search", s.publicRead(s.handlePublicSearch))
	mux.HandleFunc("GET /api/v1/public/settings", s.publicRead(s.handlePublicSettings))
	mux.HandleFunc("GET /api/v1/public/uploads/{filename}", s.handlePublicUpload)

	// Admin: categories
	mux.HandleFunc("GET /api/v1/admin/categories", s.requireAuth(s.handleListCategories))
	mux.HandleFunc("POST /api/v1/admin/categories", s.requireAuth(s.handleCreateCategory))
	mux.HandleFunc("PUT /api/v1/admin/categories/order", s.requireAuth(s.handleReorderCategories))
	mux.HandleFunc("GET /api/v1/admin/categories/{id}", s.requireAuth(s.handleGetCategory))
	mux.HandleFunc("PATCH /api/v1/admin/categories/{id}", s.requireAuth(s.handleUpdateCategory))
	mux.HandleFunc("DELETE /api/v1/admin/categories/{id}", s.requireAuth(s.handleDeleteCategory))

	// Admin: bookmarks
	mux.HandleFunc("GET /api/v1/admin/bookmarks", s.requireAuth(s.handleListBookmarks))
	mux.HandleFunc("POST /api/v1/admin/bookmarks", s.requireAuth(s.handleCreateBookmark))
	mux.HandleFunc("PUT /api/v1/admin/bookmarks/order", s.requireAuth(s.handleReorderBookmarks))
	mux.HandleFunc("GET /api/v1/admin/bookmarks/{id}", s.requireAuth(s.handleGetBookmark))
	mux.HandleFunc("PATCH /api/v1/admin/bookmarks/{id}", s.requireAuth(s.handleUpdateBookmark))
	mux.HandleFunc("DELETE /api/v1/admin/bookmarks/{id}", s.requireAuth(s.handleDeleteBookmark))
	mux.HandleFunc("POST /api/v1/admin/bookmarks/{id}/move", s.requireAuth(s.handleMoveBookmark))

	// Admin: settings, icons, transfer, backups
	mux.HandleFunc("GET /api/v1/admin/settings", s.requireAuth(s.handleGetSettings))
	mux.HandleFunc("PUT /api/v1/admin/settings", s.requireAuth(s.handleUpdateSettings))
	mux.HandleFunc("POST /api/v1/admin/uploads/icon", s.requireAuth(s.handleUploadIcon))
	mux.HandleFunc("POST /api/v1/admin/icons/fetch", s.requireAuth(s.handleFetchIcon))
	mux.HandleFunc("GET /api/v1/admin/export", s.requireAuth(s.handleExport))
	mux.HandleFunc("POST /api/v1/admin/import/preview", s.requireAuth(s.handleImportPreview))
	mux.HandleFunc("POST /api/v1/admin/import", s.requireAuth(s.handleImport))
	mux.HandleFunc("GET /api/v1/admin/backups", s.requireAuth(s.handleListBackups))
	mux.HandleFunc("POST /api/v1/admin/backups", s.requireAuth(s.handleCreateBackup))
	mux.HandleFunc("POST /api/v1/admin/backups/{id}/restore", s.requireAuth(s.handleRestoreBackup))
	mux.HandleFunc("GET /api/v1/admin/update-check", s.requireAuth(s.handleUpdateCheck))

	// Unmatched API routes return a JSON 404 rather than the SPA shell.
	mux.HandleFunc("/api/", notFound)

	if s.static != nil {
		mux.Handle("/", s.static)
	} else {
		mux.HandleFunc("/", s.handleNoFrontend)
	}

	return chain(mux,
		withRecoverer(s.log),
		withRequestID,
		withLogging(s.log),
		withSecurityHeaders,
		s.withOriginCheck,
		withCORS(s.cfg.AllowedOrigins),
		withBodyLimit,
	)
}

type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, r, http.StatusOK, healthResponse{Status: "ok", Version: version.Version})
}

type versionInfo struct {
	Version       string `json:"version"`
	Commit        string `json:"commit"`
	BuildTime     string `json:"build_time"`
	SchemaVersion int    `json:"schema_version"`
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	schemaVersion, err := migrations.CurrentVersion(s.db)
	if err != nil {
		s.log.Error("read schema version", "error", err)
		writeError(w, r, NewAPIError(http.StatusInternalServerError, CodeInternalError, "无法读取数据库版本"))
		return
	}
	writeJSON(w, r, http.StatusOK, versionInfo{
		Version:       version.Version,
		Commit:        version.Commit,
		BuildTime:     version.BuildTime,
		SchemaVersion: schemaVersion,
	})
}

func (s *Server) handleNoFrontend(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		writeJSON(w, r, http.StatusOK, map[string]string{
			"service": "xiaowork-nav",
			"version": version.Version,
			"note":    "frontend not bundled in this build; see /api/v1/version",
		})
		return
	}
	writeError(w, r, NewAPIError(http.StatusNotFound, CodeNotFound, "资源不存在"))
}
