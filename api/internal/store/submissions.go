package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Submission struct {
	ID           string
	EventID      string
	InvitationID string
	SenderName   string
	Type         string
	Message      string
	Status       string
}

type SubmissionStore struct {
	db *pgxpool.Pool
}

func NewSubmissionStore(db *pgxpool.Pool) *SubmissionStore {
	return &SubmissionStore{db: db}
}

func (s *SubmissionStore) Create(ctx context.Context, sub Submission) (string, error) {
	var id string
	err := s.db.QueryRow(ctx,
		`INSERT INTO submissions (event_id, invitation_id, sender_name, type, message, status)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		sub.EventID, sub.InvitationID, sub.SenderName, sub.Type, sub.Message, sub.Status,
	).Scan(&id)
	return id, err
}

func (s *SubmissionStore) GetByID(ctx context.Context, id string) (Submission, error) {
	var sub Submission
	err := s.db.QueryRow(ctx,
		`SELECT id, event_id, COALESCE(invitation_id::text, ''), sender_name, type, COALESCE(message, ''), status
		 FROM submissions WHERE id = $1`,
		id,
	).Scan(&sub.ID, &sub.EventID, &sub.InvitationID, &sub.SenderName, &sub.Type, &sub.Message, &sub.Status)
	return sub, err
}
