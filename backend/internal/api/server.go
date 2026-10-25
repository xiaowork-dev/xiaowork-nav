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
	iconFill *service.IconFillService
	transfer *service.TransferService
	backup   *service.BackupService
}

// New builds a Server and its services.
func New(deps Deps) *Server {
	st := store.New(deps.DB)
	settings := service.NewSettingsService(st)
	icons := service.NewIconService(st, deps.Config.UploadsDir)
	catalog := service.NewCatalogService(st, settings)
	return &Server{
		cfg:      deps.Config,
		db:       deps.DB,
		log:      deps.Logger,
		static:   deps.Static,
		store:    st,
		auth:     service.NewAuthService(st, deps.Config),
		settings: settings,
		catalog:  catalog,
		icons:    icons,
		iconFill: service.NewIconFillService(icons, st, catalog),
		transfer: service.NewTransferService(st, icons),
		backup:   service.NewBackupService(deps.DB, deps.Config.BackupsDir, deps.Config.UploadsDir, version.Version, deps.Config.BackupKeep),
	}
}

// Handler returns the fully wired HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	for _, rt := range s.routes() {
		mux.HandleFunc(rt.Method+" "+rt.Pattern, rt.Handler)
	}

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
