package security

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"
)

// SSRF protection: outbound fetches (e.g. favicon retrieval) must not reach
// loopback, private, link-local or cloud-metadata addresses. The check is
// performed on the *resolved* address inside the dialer control hook, which
// also defends against DNS rebinding.

var blockedCIDRs = func() []*net.IPNet {
	cidrs := []string{
		"0.0.0.0/8",          // "this" network
		"10.0.0.0/8",         // private
		"100.64.0.0/10",      // carrier-grade NAT
		"127.0.0.0/8",        // loopback
		"169.254.0.0/16",     // link-local (incl. 169.254.169.254 metadata)
		"172.16.0.0/12",      // private
		"192.0.0.0/24",       // IETF protocol assignments
		"192.0.2.0/24",       // TEST-NET-1
		"192.168.0.0/16",     // private
		"198.18.0.0/15",      // benchmarking
		"198.51.100.0/24",    // TEST-NET-2
		"203.0.113.0/24",     // TEST-NET-3
		"224.0.0.0/4",        // multicast
		"240.0.0.0/4",        // reserved
		"255.255.255.255/32", // broadcast
		"::/128",             // unspecified
		"::1/128",            // loopback
		"fc00::/7",           // unique local
		"fe80::/10",          // link-local
		"ff00::/8",           // multicast
	}
	out := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		if _, n, err := net.ParseCIDR(c); err == nil {
			out = append(out, n)
		}
	}
	return out
}()

// IsBlockedIP reports whether an IP must not be contacted.
func IsBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	for _, n := range blockedCIDRs {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// SafeHTTPClient returns an http.Client that refuses to connect to blocked
// addresses and limits redirects.
func SafeHTTPClient(timeout time.Duration, maxRedirects int) *http.Client {
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 10 * time.Second,
		Control: func(network, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return fmt.Errorf("ssrf: bad address %q: %w", address, err)
			}
			ip := net.ParseIP(host)
			if ip == nil {
				return fmt.Errorf("ssrf: unresolvable address %q", address)
			}
			if IsBlockedIP(ip) {
				return fmt.Errorf("ssrf: blocked address %s", ip)
			}
			return nil
		},
	}

	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: time.Second,
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("ssrf: too many redirects")
			}
			if err := ValidateFetchURL(req.URL.String()); err != nil {
				return fmt.Errorf("ssrf: unsafe redirect: %w", err)
			}
			return nil
		},
	}
}

// Fetch fetches a URL with the safe client, returning the response.
func Fetch(ctx context.Context, client *http.Client, rawURL string) (*http.Response, error) {
	if err := ValidateHTTPURL(rawURL); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "XiaoWork-Nav/2.0 (+https://github.com/xiaowork-dev/xiaowork-nav)")
	// Outbound fetches are icon/page retrievals: prefer images, then HTML.
	req.Header.Set("Accept", "image/avif,image/webp,image/svg+xml,image/*;q=0.9,text/html;q=0.8,*/*;q=0.5")
	return client.Do(req)
}

// ValidateFetchURL applies the stricter rules used for server-side icon
// fetching on top of ValidateHTTPURL:
//
//   - credentials embedded in the URL (`user:pass@host`) are rejected, so a
//     fetch can never be used to smuggle authentication;
//   - a host is required and the URL must be absolute.
//
// Host-level restrictions (loopback, private, link-local, cloud metadata) are
// enforced on the *resolved* address inside the dialer, which also covers DNS
// rebinding; see SafeHTTPClient. The port is intentionally not restricted: the
// requirement targets address ranges, and blocking legitimate non-standard
// ports would break self-hosted sites for no additional safety.
func ValidateFetchURL(raw string) error {
	if err := ValidateHTTPURL(raw); err != nil {
		return err
	}
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("%w: parse: %v", ErrInvalidURL, err)
	}
	if u.User != nil {
		return fmt.Errorf("%w: credentials in the URL are not allowed", ErrInvalidURL)
	}
	if u.Hostname() == "" {
		return fmt.Errorf("%w: missing host", ErrInvalidURL)
	}
	return nil
}
