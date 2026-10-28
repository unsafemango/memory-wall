package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Organizer struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
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
		`SELECT id, email, display_name, password_hash FROM organizers WHERE email = $1`,
		email,
	).Scan(&o.ID, &o.Email, &o.DisplayName, &o.PasswordHash)
	return o, err
}

func (s *OrganizerStore) GetByID(ctx context.Context, id string) (Organizer, error) {
	var o Organizer
	err := s.db.QueryRow(ctx,
		`SELECT id, email, display_name, password_hash FROM organizers WHERE id = $1`,
		id,
	).Scan(&o.ID, &o.Email, &o.DisplayName, &o.PasswordHash)
	return o, err
}
