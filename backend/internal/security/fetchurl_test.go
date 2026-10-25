package security

import "testing"

// TestValidateFetchURL covers the extra restrictions applied to server-side
// icon fetching: no credentials in the URL and only the standard web ports.
func TestValidateFetchURL(t *testing.T) {
	allowed := []string{
		"https://example.com/favicon.ico",
		"http://example.com/icon.png",
		"https://example.com:443/icon.svg",
		"http://example.com:80/icon.ico",
		"https://cdn.example.com/a/b/c.png?x=1",
		"http://127.0.0.1:8080/icon.png", // the port is allowed; the dialer blocks the address
	}
	for _, raw := range allowed {
		if err := ValidateFetchURL(raw); err != nil {
			t.Errorf("ValidateFetchURL(%q) = %v, want nil", raw, err)
		}
	}

	rejected := map[string]string{
		"https://user:pass@example.com/icon.png": "credentials must be rejected",
		"ftp://example.com/icon.png":             "non-http scheme must be rejected",
		"javascript:alert(1)":                    "javascript scheme must be rejected",
		"":                                       "empty url must be rejected",
		"/favicon.ico":                           "relative url must be rejected",
		"https:///icon.png":                      "missing host must be rejected",
	}
	for raw, reason := range rejected {
		if err := ValidateFetchURL(raw); err == nil {
			t.Errorf("ValidateFetchURL(%q) = nil, want an error (%s)", raw, reason)
		}
	}
}

// TestValidateFetchURLStillAcceptsPlainHTTPURLs guards the relationship with the
// looser validator used for user-supplied bookmark URLs.
func TestValidateFetchURLStillAcceptsPlainHTTPURLs(t *testing.T) {
	if err := ValidateHTTPURL("https://example.com:8443/x"); err != nil {
		t.Fatalf("ValidateHTTPURL() rejected a normal URL: %v", err)
	}
	// The fetch validator adds the credentials rule on top.
	if err := ValidateFetchURL("https://user:pw@example.com:8443/x"); err == nil {
		t.Fatal("ValidateFetchURL() must reject embedded credentials")
	}
	if err := ValidateFetchURL("https://example.com:8443/x"); err != nil {
		t.Fatalf("ValidateFetchURL() rejected a credential-free URL: %v", err)
	}
}
