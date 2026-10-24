package api

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/models"
)

type authCtxKeyType struct{}

var authCtxKey authCtxKeyType

type authContext struct {
	User    *models.User
	Session *models.Session
}

func authFrom(ctx context.Context) *authContext {
	if v, ok := ctx.Value(authCtxKey).(*authContext); ok {
		return v
	}
	return nil
}

// isUnsafeMethod reports whether the method changes state.
func isUnsafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	default:
		return true
	}
}

// withOriginCheck rejects state-changing requests whose Origin/Referer does
// not match the request host (CSRF defence). Requests without either header
// (e.g. non-browser API clients) are allowed.
func (s *Server) withOriginCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isUnsafeMethod(r.Method) {
			if err := s.checkOrigin(r); err != nil {
				writeError(w, r, err)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) checkOrigin(r *http.Request) error {
	target := r.Header.Get("Origin")
	if target == "" {
		target = r.Header.Get("Referer")
	}
	if target == "" {
		return nil
	}
	u, err := url.Parse(target)
	if err != nil || u.Host == "" {
		return NewAPIError(http.StatusForbidden, CodeCSRFFailed, "请求来源校验失败")
	}
	if sameHost(u.Host, r.Host) {
		return nil
	}
	for _, allowed := range s.cfg.AllowedOrigins {
		if allowed == "*" {
			return nil
		}
		if a, err := url.Parse(allowed); err == nil && sameHost(a.Host, u.Host) {
			return nil
		}
	}
	return NewAPIError(http.StatusForbidden, CodeCSRFFailed, "请求来源校验失败")
}

func sameHost(a, b string) bool {
	return equalFold(trimPort(a), trimPort(b))
}

func trimPort(host string) string {
	for i := len(host) - 1; i >= 0; i-- {
		if host[i] == ':' {
			return host[:i]
		}
	}
	return host
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

// requireAuth wraps a handler so that a valid session is mandatory.
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(s.cfg.SessionCookieName)
		if err != nil || cookie.Value == "" {
			writeError(w, r, NewAPIError(http.StatusUnauthorized, CodeUnauthorized, "未登录"))
			return
		}
		user, session, err := s.auth.Authenticate(r.Context(), cookie.Value)
		if err != nil {
			s.clearSessionCookie(w)
			writeError(w, r, err)
			return
		}
		ctx := context.WithValue(r.Context(), authCtxKey, &authContext{User: user, Session: session})
		next(w, r.WithContext(ctx))
	}
}

// publicRead allows anonymous access when guest access is enabled, otherwise
// requires an authenticated session.
func (s *Server) publicRead(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowed, err := s.settings.AllowGuestAccess(r.Context())
		if err != nil {
			writeError(w, r, err)
			return
		}
		if allowed {
			next(w, r)
			return
		}
		s.requireAuth(next)(w, r)
	}
}

// optionalAuth resolves a session when present without failing otherwise.
func (s *Server) optionalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(s.cfg.SessionCookieName)
		if err == nil && cookie.Value != "" {
			if user, session, err := s.auth.Authenticate(r.Context(), cookie.Value); err == nil {
				ctx := context.WithValue(r.Context(), authCtxKey, &authContext{User: user, Session: session})
				next(w, r.WithContext(ctx))
				return
			}
		}
		next(w, r)
	}
}

func (s *Server) setSessionCookie(w http.ResponseWriter, session *models.Session) {
	expires, err := time.Parse(time.RFC3339, session.ExpiresAt)
	if err != nil {
		expires = time.Now().Add(s.auth.SessionTTL())
	}
	http.SetCookie(w, &http.Cookie{
		Name:     s.cfg.SessionCookieName,
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cfg.SecureCookie,
		SameSite: http.SameSiteLaxMode,
		Expires:  expires,
		MaxAge:   int(s.auth.SessionTTL().Seconds()),
	})
}

func (s *Server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.cfg.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cfg.SecureCookie,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}
