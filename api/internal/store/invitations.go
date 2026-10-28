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
	// Shared is true for an event's reusable share link. It has no ID or
	// name, and using it doesn't use it up.
	Shared bool `json:"shared"`
	// ShareEnabled is false when the organizer has turned the share link off.
	ShareEnabled bool `json:"-"`
	// TokenEncrypted is the raw token sealed with the server's key, so the
	// organizer can see the link again. Empty for invitations made before
	// links were kept.
	TokenEncrypted string `json:"-"`
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

// GetShareLink looks up an event by its share token and returns it as a
// shared invitation. Check ShareEnabled before letting anyone use it.
func (s *InvitationStore) GetShareLink(ctx context.Context, token string) (Invitation, error) {
	inv := Invitation{Shared: true}
	err := s.db.QueryRow(ctx,
		`SELECT id, share_enabled FROM events WHERE share_token = $1`,
		token,
	).Scan(&inv.EventID, &inv.ShareEnabled)
	return inv, err
}

// Create saves a new invitation and returns its ID. tokenEncrypted may be
// empty, in which case the link can't be shown again.
func (s *InvitationStore) Create(ctx context.Context, eventID string, inv NewInvitation, expiresAt time.Time) (string, error) {
	ids, err := s.CreateMany(ctx, eventID, []NewInvitation{inv}, expiresAt)
	if err != nil {
		return "", err
	}
	return ids[0], nil
}

// NewInvitation is one invitation to create in bulk.
type NewInvitation struct {
	TokenHash      string
	TokenEncrypted string
	RecipientName  string
}

// CreateMany saves several invitations in one transaction, so either all of
// them exist or none do. It returns their IDs in the same order.
func (s *InvitationStore) CreateMany(ctx context.Context, eventID string, invitations []NewInvitation, expiresAt time.Time) ([]string, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	ids := make([]string, len(invitations))
	for i, inv := range invitations {
		err := tx.QueryRow(ctx,
			`INSERT INTO invitations (event_id, token_hash, token_encrypted, recipient_name, expires_at)
			 VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5) RETURNING id`,
			eventID, inv.TokenHash, inv.TokenEncrypted, inv.RecipientName, expiresAt,
		).Scan(&ids[i])
		if err != nil {
			return nil, err
		}
	}
	return ids, tx.Commit(ctx)
}

// ListForEvent returns an event's invitations, newest first, with their
// encrypted tokens so the caller can rebuild the links.
func (s *InvitationStore) ListForEvent(ctx context.Context, eventID string) ([]Invitation, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, event_id, COALESCE(recipient_name, ''), expires_at, used_at, COALESCE(token_encrypted, '')
		 FROM invitations WHERE event_id = $1 ORDER BY created_at DESC`,
		eventID,
	)
	if err != nil {
		return nil, err
	}
	invitations, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Invitation, error) {
		var inv Invitation
		err := row.Scan(&inv.ID, &inv.EventID, &inv.RecipientName, &inv.ExpiresAt, &inv.UsedAt, &inv.TokenEncrypted)
		return inv, err
	})
	if invitations == nil {
		invitations = []Invitation{}
	}
	return invitations, err
}
