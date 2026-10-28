package auth

import (
	"strings"
	"testing"
)

const testKey = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"

func TestSealerRoundTrip(t *testing.T) {
	s, err := NewSealer(testKey)
	if err != nil {
		t.Fatalf("new sealer: %v", err)
	}
	sealed, err := s.Seal("invite-token")
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if strings.Contains(sealed, "invite-token") {
		t.Fatal("sealed value contains the plaintext")
	}
	again, _ := s.Seal("invite-token")
	if again == sealed {
		t.Fatal("sealing twice should give different values")
	}
	got, err := s.Open(sealed)
	if err != nil || got != "invite-token" {
		t.Fatalf("open = %q, %v", got, err)
	}
}

func TestSealerRejectsTamperingAndWrongKey(t *testing.T) {
	s, _ := NewSealer(testKey)
	sealed, _ := s.Seal("invite-token")

	tampered := []byte(sealed)
	tampered[len(tampered)-2] ^= 1
	if _, err := s.Open(string(tampered)); err == nil {
		t.Fatal("a changed value should fail to open")
	}

	other, _ := NewSealer(strings.Repeat("ab", 32))
	if _, err := other.Open(sealed); err == nil {
		t.Fatal("a different key should fail to open")
	}
}

func TestNewSealerChecksKey(t *testing.T) {
	for _, key := range []string{"", "abc", strings.Repeat("zz", 32), strings.Repeat("ab", 16)} {
		if _, err := NewSealer(key); err == nil {
			t.Errorf("key %q should be rejected", key)
		}
	}
}
