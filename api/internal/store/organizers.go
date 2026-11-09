package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Organizer struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	CreatedAt   time.Time `json:"created_at"`
	// EmailVerified is true once they've confirmed their address.
	EmailVerified     bool `json:"email_verified"`
	NotifyNewMemories bool `json:"notify_new_memories"`
	// The json:"-" tag keeps the hash out of every API response.
	PasswordHash string `json:"-"`
}

type OrganizerStore struct {
	db *pgxpool.Pool
}

func NewOrganizerStore(db *pgxpool.Pool) *OrganizerStore {
	return &OrganizerStore{db: db}
}

const organizerColumns = `id, email, display_name, created_at, email_verified_at IS NOT NULL, notify_new_memories, password_hash`

func scanOrganizer(row interface{ Scan(...any) error }) (Organizer, error) {
	var o Organizer
	err := row.Scan(&o.ID, &o.Email, &o.DisplayName, &o.CreatedAt, &o.EmailVerified, &o.NotifyNewMemories, &o.PasswordHash)
	return o, err
}

func (s *OrganizerStore) Create(ctx context.Context, email, passwordHash, displayName string) (string, error) {
	var id string
	err := s.db.QueryRow(ctx,
		`INSERT INTO organizers (email, password_hash, display_name)
		 VALUES ($1, $2, $3) RETURNING id`,
		email, passwordHash, displayName,
	).Scan(&id)
	return id, mapDuplicate(err)
}

func (s *OrganizerStore) GetByEmail(ctx context.Context, email string) (Organizer, error) {
	return scanOrganizer(s.db.QueryRow(ctx, `SELECT `+organizerColumns+` FROM organizers WHERE email = $1`, email))
}

func (s *OrganizerStore) GetByID(ctx context.Context, id string) (Organizer, error) {
	return scanOrganizer(s.db.QueryRow(ctx, `SELECT `+organizerColumns+` FROM organizers WHERE id = $1`, id))
}

// UpdateProfile changes an organizer's name and email. A new email has to be
// confirmed again. Returns ErrDuplicate if another account already uses it.
func (s *OrganizerStore) UpdateProfile(ctx context.Context, id, email, displayName string) error {
	_, err := s.db.Exec(ctx,
		`UPDATE organizers
		 SET display_name = $3, email = $2,
		     email_verified_at = CASE WHEN email = $2 THEN email_verified_at END
		 WHERE id = $1`,
		id, email, displayName,
	)
	return mapDuplicate(err)
}

func (s *OrganizerStore) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	_, err := s.db.Exec(ctx, `UPDATE organizers SET password_hash = $2 WHERE id = $1`, id, passwordHash)
	return err
}

// SetNotifications turns new-memory emails on or off.
func (s *OrganizerStore) SetNotifications(ctx context.Context, id string, enabled bool) error {
	_, err := s.db.Exec(ctx, `UPDATE organizers SET notify_new_memories = $2 WHERE id = $1`, id, enabled)
	return err
}

// Unsubscribe turns off new-memory emails for whoever owns the token, without
// needing them to log in. Returns ErrNotFound for an unknown token.
func (s *OrganizerStore) Unsubscribe(ctx context.Context, token string) error {
	tag, err := s.db.Exec(ctx, `UPDATE organizers SET notify_new_memories = false WHERE unsubscribe_token = $1`, token)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateEmailConfirmation saves a confirmation token for an address, replacing
// any earlier one so only the newest link works.
func (s *OrganizerStore) CreateEmailConfirmation(ctx context.Context, tokenHash, organizerID, email string, expiresAt time.Time) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM email_confirmations WHERE organizer_id = $1`, organizerID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO email_confirmations (token_hash, organizer_id, email, expires_at) VALUES ($1, $2, $3, $4)`,
		tokenHash, organizerID, email, expiresAt,
	); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ConfirmEmail uses up a confirmation token and marks the address confirmed.
// The token only counts if the account still has that email. Returns
// ErrNotFound for an unknown, expired, or out-of-date token.
func (s *OrganizerStore) ConfirmEmail(ctx context.Context, tokenHash string) error {
	tag, err := s.db.Exec(ctx,
		`WITH used AS (
		     DELETE FROM email_confirmations
		     WHERE token_hash = $1 AND expires_at > now()
		     RETURNING organizer_id, email
		 )
		 UPDATE organizers o SET email_verified_at = now()
		 FROM used WHERE o.id = used.organizer_id AND o.email = used.email`,
		tokenHash,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
