package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SessionStore keeps logged-in sessions. Only a hash of each session token is
// stored, so a leaked database can't be used to log in as anyone.
type SessionStore struct {
	db *pgxpool.Pool
}

func NewSessionStore(db *pgxpool.Pool) *SessionStore {
	return &SessionStore{db: db}
}

func (s *SessionStore) Create(ctx context.Context, tokenHash, organizerID string, expiresAt time.Time) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO sessions (token_hash, organizer_id, expires_at) VALUES ($1, $2, $3)`,
		tokenHash, organizerID, expiresAt,
	)
	return err
}

// OrganizerIDFor returns who a session belongs to, if it hasn't expired.
func (s *SessionStore) OrganizerIDFor(ctx context.Context, tokenHash string) (string, error) {
	var id string
	err := s.db.QueryRow(ctx,
		`SELECT organizer_id FROM sessions WHERE token_hash = $1 AND expires_at > now()`,
		tokenHash,
	).Scan(&id)
	return id, err
}

func (s *SessionStore) Delete(ctx context.Context, tokenHash string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}
