package store

import (
	"context"
	"errors"
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
	MimeType   string    `json:"mime_type,omitempty"`
	// MediaURL is filled in by the handler with a temporary signed link.
	MediaURL   string `json:"media_url,omitempty"`
	StorageKey string `json:"-"`
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
		`SELECT s.id, s.sender_name, s.type, COALESCE(s.message, ''), s.created_at,
		        COALESCE(m.storage_key, ''), COALESCE(m.mime_type, '')
		 FROM submissions s
		 JOIN events e ON e.id = s.event_id
		 LEFT JOIN media m ON m.submission_id = s.id
		 WHERE e.slug = $1 AND s.status = 'approved'
		 ORDER BY s.created_at`,
		slug,
	)
	if err != nil {
		return nil, err
	}
	memories, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Memory, error) {
		var m Memory
		err := row.Scan(&m.ID, &m.SenderName, &m.Type, &m.Message, &m.CreatedAt, &m.StorageKey, &m.MimeType)
		return m, err
	})
	if memories == nil {
		memories = []Memory{}
	}
	return memories, err
}

// ModerationItem is a submission as the organizer sees it, any status.
type ModerationItem struct {
	Memory
	Status string `json:"status"`
}

// ListForOwner returns every submission on an event, newest first, but only
// if ownerID owns the event. Someone else's event looks empty, not forbidden.
func (s *SubmissionStore) ListForOwner(ctx context.Context, eventID, ownerID string) ([]ModerationItem, error) {
	rows, err := s.db.Query(ctx,
		`SELECT s.id, s.sender_name, s.type, COALESCE(s.message, ''), s.created_at,
		        COALESCE(m.storage_key, ''), COALESCE(m.mime_type, ''), s.status
		 FROM submissions s
		 JOIN events e ON e.id = s.event_id
		 LEFT JOIN media m ON m.submission_id = s.id
		 WHERE s.event_id = $1 AND e.owner_id = $2
		 ORDER BY s.created_at DESC`,
		eventID, ownerID,
	)
	if err != nil {
		return nil, err
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (ModerationItem, error) {
		var it ModerationItem
		err := row.Scan(&it.ID, &it.SenderName, &it.Type, &it.Message, &it.CreatedAt, &it.StorageKey, &it.MimeType, &it.Status)
		return it, err
	})
	if items == nil {
		items = []ModerationItem{}
	}
	return items, err
}

// UpdateStatus approves or rejects a submission. The owner check is part of
// the SQL, so there's no code path that forgets it.
func (s *SubmissionStore) UpdateStatus(ctx context.Context, id, ownerID, status string) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE submissions s SET status = $3
		 FROM events e
		 WHERE s.id = $1 AND s.event_id = e.id AND e.owner_id = $2`,
		id, ownerID, status,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteForOwner removes a submission and returns its media's storage key, if
// any, so the caller can delete the file too.
func (s *SubmissionStore) DeleteForOwner(ctx context.Context, id, ownerID string) (string, error) {
	var key string
	err := s.db.QueryRow(ctx,
		`WITH target AS (
		     SELECT s.id, COALESCE(m.storage_key, '') AS storage_key
		     FROM submissions s
		     JOIN events e ON e.id = s.event_id
		     LEFT JOIN media m ON m.submission_id = s.id
		     WHERE s.id = $1 AND e.owner_id = $2
		 )
		 DELETE FROM submissions USING target WHERE submissions.id = target.id
		 RETURNING target.storage_key`,
		id, ownerID,
	).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return key, err
}
