package auth

import (
	"strings"
	"testing"
)

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("unexpected hash format: %s", hash)
	}

	tests := []struct {
		name     string
		password string
		encoded  string
		want     bool
	}{
		{"right password", "correct horse battery", hash, true},
		{"wrong password", "correct horse battery!", hash, false},
		{"empty password", "", hash, false},
		{"malformed hash", "correct horse battery", "not-a-hash", false},
		{"wrong algorithm", "correct horse battery", strings.Replace(hash, "argon2id", "argon2i", 1), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VerifyPassword(tt.password, tt.encoded); got != tt.want {
				t.Fatalf("VerifyPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHashPasswordUsesRandomSalt(t *testing.T) {
	a, _ := HashPassword("same password")
	b, _ := HashPassword("same password")
	if a == b {
		t.Fatal("two hashes of the same password should differ")
	}
}

func TestNewToken(t *testing.T) {
	raw, hash, err := NewToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(raw) != 64 || len(hash) != 64 {
		t.Fatalf("expected 64 hex chars, got raw=%d hash=%d", len(raw), len(hash))
	}
	if raw == hash {
		t.Fatal("hash should not equal the raw token")
	}
	if HashToken(raw) != hash {
		t.Fatal("hashing the raw token again should give the same hash")
	}

	other, _, _ := NewToken()
	if other == raw {
		t.Fatal("two tokens should differ")
	}
}
