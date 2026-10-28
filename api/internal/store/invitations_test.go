package store

import (
	"testing"
	"time"
)

func TestInvitationUsable(t *testing.T) {
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name string
		inv  Invitation
		want bool
	}{
		{"no expiry, unused", Invitation{}, true},
		{"expires later", Invitation{ExpiresAt: &future}, true},
		{"expired", Invitation{ExpiresAt: &past}, false},
		{"expires right now", Invitation{ExpiresAt: &now}, false},
		{"already used", Invitation{UsedAt: &past, ExpiresAt: &future}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.inv.Usable(now); got != tt.want {
				t.Fatalf("Usable() = %v, want %v", got, tt.want)
			}
		})
	}
}
