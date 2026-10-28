package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Invitation struct {
	ID            string     `json:"id"`
	EventID       string     `json:"event_id"`
	RecipientName string     `json:"recipient_name"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	UsedAt        *time.Time `json:"used_at,omitempty"`
}

// Usable reports whether the invitation can still be used to submit.
func (inv Invitation) Usable(now time.Time) bool {
	return inv.UsedAt == nil && (inv.ExpiresAt == nil || now.Before(*inv.ExpiresAt))
}

type InvitationStore struct {
	db *pgxpool.Pool
}

func NewInvitationStore(db *pgxpool.Pool) *InvitationStore {
	return &InvitationStore{db: db}
}

// GetByTokenHash looks an invitation up by the SHA-256 hash of its token.
// Raw tokens are never stored.
func (s *InvitationStore) GetByTokenHash(ctx context.Context, tokenHash string) (Invitation, error) {
	var inv Invitation
	err := s.db.QueryRow(ctx,
		`SELECT id, event_id, COALESCE(recipient_name, ''), expires_at, used_at
		 FROM invitations WHERE token_hash = $1`,
		tokenHash,
	).Scan(&inv.ID, &inv.EventID, &inv.RecipientName, &inv.ExpiresAt, &inv.UsedAt)
	return inv, err
}
