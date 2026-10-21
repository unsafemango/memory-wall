package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Invitation struct {
	ID            string `json:"id"`
	EventID       string `json:"event_id"`
	RecipientName string `json:"recipient_name"`
}

type InvitationStore struct {
	db *pgxpool.Pool
}

func NewInvitationStore(db *pgxpool.Pool) *InvitationStore {
	return &InvitationStore{db: db}
}

// GetByToken compares the raw token for now; Phase 5 switches this to a hash
func (s *InvitationStore) GetByToken(ctx context.Context, token string) (Invitation, error) {
	var inv Invitation
	err := s.db.QueryRow(ctx, `SELECT id, event_id, COALESCE(recipient_name, '') FROM invitations WHERE token_hash = $1`, token).Scan(&inv.ID, &inv.EventID, &inv.RecipientName)

	return inv, err
}
