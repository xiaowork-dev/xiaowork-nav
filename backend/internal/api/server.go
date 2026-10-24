// Package api wires HTTP routing, middleware, and handlers.
//
// Handlers translate HTTP requests into domain operations; they must only
// expose the fields defined in docs/openapi/openapi.yaml.
package api

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/config"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/migrations"
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
	cfg    *config.Config
	db     *sql.DB
	log    *slog.Logger
	static http.Handler
}

// New builds a Server.
func New(deps Deps) *Server {
	return &Server{
		cfg:    deps.Config,
		db:     deps.DB,
		log:    deps.Logger,
		static: deps.Static,
	}
}

// Handler returns the fully wired HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /api/v1/version", s.handleVersion)
	mux.HandleFunc("GET /api/v1/auth/status", s.handleAuthStatus)

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

type authStatus struct {
	Initialized   bool `json:"initialized"`
	Authenticated bool `json:"authenticated"`
}

func (s *Server) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		s.log.Error("count users", "error", err)
		writeError(w, r, NewAPIError(http.StatusInternalServerError, CodeInternalError, "无法读取用户信息"))
		return
	}
	// Session-based authentication is implemented in the auth milestone;
	// until then no request is considered authenticated.
	writeJSON(w, r, http.StatusOK, authStatus{
		Initialized:   count > 0,
		Authenticated: false,
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
