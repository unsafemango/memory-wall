package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
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

func (s *InvitationStore) Create(ctx context.Context, eventID, tokenHash, recipientName string, expiresAt time.Time) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO invitations (event_id, token_hash, recipient_name, expires_at)
		 VALUES ($1, $2, NULLIF($3, ''), $4)`,
		eventID, tokenHash, recipientName, expiresAt,
	)
	return err
}

// ListForEvent returns an event's invitations, newest first. Tokens can't be
// shown again since only their hashes are stored.
func (s *InvitationStore) ListForEvent(ctx context.Context, eventID string) ([]Invitation, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, event_id, COALESCE(recipient_name, ''), expires_at, used_at
		 FROM invitations WHERE event_id = $1 ORDER BY created_at DESC`,
		eventID,
	)
	if err != nil {
		return nil, err
	}
	invitations, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Invitation, error) {
		var inv Invitation
		err := row.Scan(&inv.ID, &inv.EventID, &inv.RecipientName, &inv.ExpiresAt, &inv.UsedAt)
		return inv, err
	})
	if invitations == nil {
		invitations = []Invitation{}
	}
	return invitations, err
}
