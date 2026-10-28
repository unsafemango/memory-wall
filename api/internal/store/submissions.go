package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
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

// Memory is an approved submission as shown on the public event page.
type Memory struct {
	ID         string    `json:"id"`
	SenderName string    `json:"sender_name"`
	Type       string    `json:"type"`
	Message    string    `json:"message"`
	CreatedAt  time.Time `json:"created_at"`
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

// ListApprovedBySlug returns an event's approved submissions, oldest first.
func (s *SubmissionStore) ListApprovedBySlug(ctx context.Context, slug string) ([]Memory, error) {
	rows, err := s.db.Query(ctx,
		`SELECT s.id, s.sender_name, s.type, COALESCE(s.message, ''), s.created_at
		 FROM submissions s
		 JOIN events e ON e.id = s.event_id
		 WHERE e.slug = $1 AND s.status = 'approved'
		 ORDER BY s.created_at`,
		slug,
	)
	if err != nil {
		return nil, err
	}
	memories, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Memory, error) {
		var m Memory
		err := row.Scan(&m.ID, &m.SenderName, &m.Type, &m.Message, &m.CreatedAt)
		return m, err
	})
	if memories == nil {
		memories = []Memory{}
	}
	return memories, err
}
