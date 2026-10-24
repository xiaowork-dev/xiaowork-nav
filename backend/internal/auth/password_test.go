package auth

import (
	"strings"
	"testing"
	"time"
)

func TestHashAndVerify(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("hash is not in PHC argon2id format: %q", hash)
	}
	if !VerifyPassword(hash, "correct horse battery staple") {
		t.Error("VerifyPassword() = false for the correct password")
	}
	if VerifyPassword(hash, "wrong password") {
		t.Error("VerifyPassword() = true for a wrong password")
	}
}

func TestHashIsSalted(t *testing.T) {
	a, _ := HashPassword("same-password")
	b, _ := HashPassword("same-password")
	if a == b {
		t.Error("two hashes of the same password must differ (unique salts)")
	}
}

func TestVerifyRejectsMalformedHash(t *testing.T) {
	for _, bad := range []string{"", "plaintext", "$argon2id$v=19$m=1,t=1,p=1$notbase64$notbase64", "$bcrypt$whatever"} {
		if VerifyPassword(bad, "x") {
			t.Errorf("VerifyPassword(%q) = true, want false", bad)
		}
	}
}

func TestNewSessionIDIsUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 64; i++ {
		id, err := NewSessionID()
		if err != nil {
			t.Fatalf("NewSessionID() error = %v", err)
		}
		if len(id) < 32 {
			t.Fatalf("session id too short: %q", id)
		}
		if seen[id] {
			t.Fatalf("duplicate session id: %q", id)
		}
		seen[id] = true
	}
}

func TestLimiterBlocksAfterLimit(t *testing.T) {
	l := NewLimiter(3, time.Minute)
	for i := 0; i < 3; i++ {
		if !l.Allow("k") {
			t.Fatalf("attempt %d should be allowed", i+1)
		}
	}
	if l.Allow("k") {
		t.Error("4th attempt should be blocked")
	}
	if l.Allow("other") == false {
		t.Error("a different key must not be affected")
	}
	if l.RetryAfter("k") <= 0 {
		t.Error("RetryAfter should report a positive duration")
	}

	l.Reset("k")
	if !l.Allow("k") {
		t.Error("after Reset the key should be allowed again")
	}
}

func TestLimiterWindowExpiry(t *testing.T) {
	l := NewLimiter(1, 20*time.Millisecond)
	if !l.Allow("k") {
		t.Fatal("first attempt should be allowed")
	}
	if l.Allow("k") {
		t.Fatal("second attempt should be blocked within the window")
	}
	time.Sleep(30 * time.Millisecond)
	if !l.Allow("k") {
		t.Error("attempt after the window should be allowed")
	}
}
