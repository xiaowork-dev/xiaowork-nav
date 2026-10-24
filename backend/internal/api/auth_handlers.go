package api

import (
	"net"
	"net/http"
	"strings"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/models"
)

type authStatusResponse struct {
	Initialized   bool         `json:"initialized"`
	Authenticated bool         `json:"authenticated"`
	User          *models.User `json:"user,omitempty"`
}

func (s *Server) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	initialized, err := s.auth.Initialized(r.Context())
	if err != nil {
		s.log.Error("auth status", "error", err)
		writeError(w, r, err)
		return
	}
	resp := authStatusResponse{Initialized: initialized}
	if ac := authFrom(r.Context()); ac != nil {
		resp.Authenticated = true
		resp.User = ac.User
	}
	writeJSON(w, r, http.StatusOK, resp)
}

type setupRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	var req setupRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	user, err := s.auth.Setup(r.Context(), req.Username, req.Password)
	if err != nil {
		writeError(w, r, err)
		return
	}

	// Log the new administrator in immediately for a smoother first run.
	if _, session, err := s.auth.Login(r.Context(), req.Username, req.Password, clientIP(r, s.cfg.TrustedProxies), r.UserAgent()); err == nil {
		s.setSessionCookie(w, session)
	}

	writeJSON(w, r, http.StatusCreated, map[string]any{"user": user})
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	user, session, err := s.auth.Login(r.Context(), req.Username, req.Password, clientIP(r, s.cfg.TrustedProxies), r.UserAgent())
	if err != nil {
		writeError(w, r, err)
		return
	}
	s.setSessionCookie(w, session)
	writeJSON(w, r, http.StatusOK, map[string]any{"user": user})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if ac := authFrom(r.Context()); ac != nil {
		if err := s.auth.Logout(r.Context(), ac.Session.ID); err != nil {
			s.log.Error("logout", "error", err)
		}
	}
	s.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	ac := authFrom(r.Context())
	if ac == nil {
		writeError(w, r, NewAPIError(http.StatusUnauthorized, CodeUnauthorized, "未登录"))
		return
	}
	writeJSON(w, r, http.StatusOK, ac.User)
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	ac := authFrom(r.Context())
	if ac == nil {
		writeError(w, r, NewAPIError(http.StatusUnauthorized, CodeUnauthorized, "未登录"))
		return
	}
	var req changePasswordRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	if err := s.auth.ChangePassword(r.Context(), ac.User.ID, req.CurrentPassword, req.NewPassword); err != nil {
		writeError(w, r, err)
		return
	}
	// All sessions were invalidated by the password change.
	s.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// clientIP returns the best-effort client address, honouring forwarding
// headers only from configured trusted proxies.
func clientIP(r *http.Request, trustedProxies []string) string {
	remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteHost = r.RemoteAddr
	}
	if !isTrustedProxy(remoteHost, trustedProxies) {
		return remoteHost
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		first := strings.TrimSpace(strings.Split(xff, ",")[0])
		if first != "" {
			return first
		}
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	return remoteHost
}

func isTrustedProxy(host string, trusted []string) bool {
	if len(trusted) == 0 {
		return false
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, entry := range trusted {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if _, cidr, err := net.ParseCIDR(entry); err == nil {
			if cidr.Contains(ip) {
				return true
			}
			continue
		}
		if candidate := net.ParseIP(entry); candidate != nil && candidate.Equal(ip) {
			return true
		}
	}
	return false
}
