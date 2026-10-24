package api

import (
	"context"
	"time"
)

// RunMaintenance periodically removes expired sessions and trims the login
// rate limiter. It blocks until ctx is cancelled.
func (s *Server) RunMaintenance(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Hour
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if n, err := s.auth.CleanupExpiredSessions(ctx); err != nil {
				s.log.Warn("session cleanup failed", "error", err)
			} else if n > 0 {
				s.log.Info("expired sessions removed", "count", n)
			}
			s.auth.CleanupLimiter()
		}
	}
}
