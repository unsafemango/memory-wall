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
	// The json:"-" tag keeps the hash out of every API response.
	PasswordHash string `json:"-"`
}

type OrganizerStore struct {
	db *pgxpool.Pool
}

func NewOrganizerStore(db *pgxpool.Pool) *OrganizerStore {
	return &OrganizerStore{db: db}
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
	var o Organizer
	err := s.db.QueryRow(ctx,
		`SELECT id, email, display_name, created_at, password_hash FROM organizers WHERE email = $1`,
		email,
	).Scan(&o.ID, &o.Email, &o.DisplayName, &o.CreatedAt, &o.PasswordHash)
	return o, err
}

func (s *OrganizerStore) GetByID(ctx context.Context, id string) (Organizer, error) {
	var o Organizer
	err := s.db.QueryRow(ctx,
		`SELECT id, email, display_name, created_at, password_hash FROM organizers WHERE id = $1`,
		id,
	).Scan(&o.ID, &o.Email, &o.DisplayName, &o.CreatedAt, &o.PasswordHash)
	return o, err
}

// UpdateProfile changes an organizer's name and email. Returns ErrDuplicate
// if another account already uses the email.
func (s *OrganizerStore) UpdateProfile(ctx context.Context, id, email, displayName string) error {
	_, err := s.db.Exec(ctx,
		`UPDATE organizers SET email = $2, display_name = $3 WHERE id = $1`,
		id, email, displayName,
	)
	return mapDuplicate(err)
}

func (s *OrganizerStore) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	_, err := s.db.Exec(ctx, `UPDATE organizers SET password_hash = $2 WHERE id = $1`, id, passwordHash)
	return err
}
