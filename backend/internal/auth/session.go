package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

// NewSessionID returns a cryptographically random, URL-safe session identifier.
func NewSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Limiter is a simple in-memory fixed-window rate limiter.
//
// It is intentionally process-local: the application runs as a single
// instance, so no shared store is required.
type Limiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	limit    int
	window   time.Duration
}

// NewLimiter creates a limiter allowing `limit` attempts per `window` per key.
func NewLimiter(limit int, window time.Duration) *Limiter {
	return &Limiter{
		attempts: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

// Allow reports whether another attempt is permitted for key, recording it.
func (l *Limiter) Allow(key string) bool {
	now := time.Now()
	cutoff := now.Add(-l.window)

	l.mu.Lock()
	defer l.mu.Unlock()

	kept := l.attempts[key][:0]
	for _, t := range l.attempts[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= l.limit {
		l.attempts[key] = kept
		return false
	}
	l.attempts[key] = append(kept, now)
	return true
}

// Reset clears recorded attempts for key (e.g. after a successful login).
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, key)
}

// RetryAfter returns the remaining lockout duration for key, if any.
func (l *Limiter) RetryAfter(key string) time.Duration {
	now := time.Now()
	cutoff := now.Add(-l.window)

	l.mu.Lock()
	defer l.mu.Unlock()

	var oldest time.Time
	for _, t := range l.attempts[key] {
		if t.After(cutoff) && (oldest.IsZero() || t.Before(oldest)) {
			oldest = t
		}
	}
	if oldest.IsZero() {
		return 0
	}
	if d := oldest.Add(l.window).Sub(now); d > 0 {
		return d
	}
	return 0
}

// Cleanup removes empty entries to bound memory usage.
func (l *Limiter) Cleanup() {
	cutoff := time.Now().Add(-l.window)

	l.mu.Lock()
	defer l.mu.Unlock()
	for k, times := range l.attempts {
		kept := times[:0]
		for _, t := range times {
			if t.After(cutoff) {
				kept = append(kept, t)
			}
		}
		if len(kept) == 0 {
			delete(l.attempts, k)
		} else {
			l.attempts[k] = kept
		}
	}
}
