// Package security provides input validation and outbound-request safety
// helpers (URL scheme allow-listing and SSRF protection).
package security

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// MaxURLLength is the maximum accepted bookmark/icon URL length.
const MaxURLLength = 2048

// ErrInvalidURL indicates a URL that fails validation.
var ErrInvalidURL = errors.New("invalid url")

// ValidateHTTPURL accepts only absolute http/https URLs of bounded length.
func ValidateHTTPURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("%w: empty", ErrInvalidURL)
	}
	if len(raw) > MaxURLLength {
		return fmt.Errorf("%w: too long", ErrInvalidURL)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%w: parse: %v", ErrInvalidURL, err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("%w: only http and https are allowed", ErrInvalidURL)
	}
	if u.Host == "" {
		return fmt.Errorf("%w: missing host", ErrInvalidURL)
	}
	return nil
}

// NormalizeURL trims whitespace from a URL.
func NormalizeURL(raw string) string {
	return strings.TrimSpace(raw)
}
