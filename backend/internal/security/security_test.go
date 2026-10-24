package security

import (
	"net"
	"testing"
)

func TestValidateHTTPURL(t *testing.T) {
	valid := []string{
		"https://example.com",
		"http://example.com/path?q=1",
		"https://sub.example.com:8443/a/b",
	}
	for _, v := range valid {
		if err := ValidateHTTPURL(v); err != nil {
			t.Errorf("ValidateHTTPURL(%q) = %v, want nil", v, err)
		}
	}

	invalid := []string{
		"",
		"javascript:alert(1)",
		"data:text/html;base64,PHNjcmlwdD4=",
		"file:///etc/passwd",
		"ftp://example.com",
		"example.com",
		"//example.com",
	}
	for _, v := range invalid {
		if err := ValidateHTTPURL(v); err == nil {
			t.Errorf("ValidateHTTPURL(%q) = nil, want error", v)
		}
	}
}

func TestValidateHTTPURLRejectsOverlong(t *testing.T) {
	long := "https://example.com/" + string(make([]byte, MaxURLLength))
	if err := ValidateHTTPURL(long); err == nil {
		t.Error("expected an over-long URL to be rejected")
	}
}

func TestIsBlockedIP(t *testing.T) {
	blocked := []string{
		"127.0.0.1",
		"10.1.2.3",
		"172.16.5.5",
		"192.168.1.1",
		"169.254.169.254", // cloud metadata
		"100.64.0.1",
		"0.0.0.0",
		"::1",
		"fc00::1",
		"fe80::1",
	}
	for _, s := range blocked {
		if !IsBlockedIP(net.ParseIP(s)) {
			t.Errorf("IsBlockedIP(%s) = false, want true", s)
		}
	}

	allowed := []string{"8.8.8.8", "1.1.1.1", "93.184.216.34", "2606:4700:4700::1111"}
	for _, s := range allowed {
		if IsBlockedIP(net.ParseIP(s)) {
			t.Errorf("IsBlockedIP(%s) = true, want false", s)
		}
	}

	if !IsBlockedIP(nil) {
		t.Error("IsBlockedIP(nil) should be true")
	}
}
