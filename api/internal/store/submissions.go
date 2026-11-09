package store

import (
	"context"
	"strings"
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
	// Files are the memory's photos and videos, in the order they were added.
	Files []MemoryFile `json:"files"`
}

// MemoryFile is one photo or video in a memory.
type MemoryFile struct {
	ID       string `json:"id"`
	MimeType string `json:"mime_type"`
	// URL and ThumbnailURL are filled in by the handler with temporary signed links.
	URL          string `json:"url,omitempty"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	StorageKey   string `json:"-"`
	ThumbnailKey string `json:"-"`
}

type SubmissionStore struct {
	db *pgxpool.Pool
}

func NewSubmissionStore(db *pgxpool.Pool) *SubmissionStore {
	return &SubmissionStore{db: db}
}

// Create saves a submission with its uploaded files, if it has any.
// In the same transaction it marks the invitation as used, so a link can only
// be used once even if two requests race. Returns ErrInvitationUsed if the
// invitation was already used or has expired.
func (s *SubmissionStore) Create(ctx context.Context, sub Submission, files []Media) (string, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	if sub.InvitationID != "" {
		tag, err := tx.Exec(ctx,
			`UPDATE invitations SET used_at = now()
			 WHERE id = $1 AND used_at IS NULL AND (expires_at IS NULL OR expires_at > now())`,
			sub.InvitationID,
		)
		if err != nil {
			return "", err
		}
		if tag.RowsAffected() == 0 {
			return "", ErrInvitationUsed
		}
	}

	var id string
	err = tx.QueryRow(ctx,
		`INSERT INTO submissions (event_id, invitation_id, sender_name, type, message, status)
		 VALUES ($1, NULLIF($2, '')::uuid, $3, $4, NULLIF($5, ''), $6) RETURNING id`,
		sub.EventID, sub.InvitationID, sub.SenderName, sub.Type, sub.Message, sub.Status,
	).Scan(&id)
	if err != nil {
		return "", err
	}

	for position, file := range files {
		var mediaID string
		err = tx.QueryRow(ctx,
			`INSERT INTO media (submission_id, storage_key, mime_type, file_size, position)
			 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
			id, file.StorageKey, file.MimeType, file.FileSize, position,
		).Scan(&mediaID)
		if err != nil {
			return "", err
		}

		// Videos get a thumbnail and their duration from a background job.
		// Queueing it in this transaction means a saved video always has its job.
		if strings.HasPrefix(strings.ToLower(file.MimeType), "video/") {
			_, err = tx.Exec(ctx,
				`INSERT INTO jobs (type, payload) VALUES ($1, json_build_object('media_id', $2::uuid))`,
				JobProcessVideo, mediaID,
			)
			if err != nil {
				return "", err
			}
		}
	}

	if err := queueNotification(ctx, tx, sub.EventID); err != nil {
		return "", err
	}

	return id, tx.Commit(ctx)
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
	if err != nil {
		return nil, err
	}
	if memories == nil {
		memories = []Memory{}
	}

	ids := make([]string, len(memories))
	for i, m := range memories {
		ids[i] = m.ID
	}
	files, err := s.filesFor(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range memories {
		memories[i].Files = files[memories[i].ID]
	}
	return memories, nil
}

// filesFor loads the files of several submissions in one query. Every id gets
// a list, even if it's empty, so the API always sends an array.
func (s *SubmissionStore) filesFor(ctx context.Context, submissionIDs []string) (map[string][]MemoryFile, error) {
	files := make(map[string][]MemoryFile, len(submissionIDs))
	for _, id := range submissionIDs {
		files[id] = []MemoryFile{}
	}
	if len(submissionIDs) == 0 {
		return files, nil
	}

	rows, err := s.db.Query(ctx,
		`SELECT submission_id, id, mime_type, storage_key, COALESCE(thumbnail_key, '')
		 FROM media
		 WHERE submission_id = ANY($1::uuid[])
		 ORDER BY position, created_at`,
		submissionIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var submissionID string
		var f MemoryFile
		if err := rows.Scan(&submissionID, &f.ID, &f.MimeType, &f.StorageKey, &f.ThumbnailKey); err != nil {
			return nil, err
		}
		files[submissionID] = append(files[submissionID], f)
	}
	return files, rows.Err()
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
		`SELECT s.id, s.sender_name, s.type, COALESCE(s.message, ''), s.created_at, s.status
		 FROM submissions s
		 JOIN events e ON e.id = s.event_id
		 WHERE s.event_id = $1 AND e.owner_id = $2
		 ORDER BY s.created_at DESC`,
		eventID, ownerID,
	)
	if err != nil {
		return nil, err
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (ModerationItem, error) {
		var it ModerationItem
		err := row.Scan(&it.ID, &it.SenderName, &it.Type, &it.Message, &it.CreatedAt, &it.Status)
		return it, err
	})
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []ModerationItem{}
	}

	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = it.ID
	}
	files, err := s.filesFor(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].Files = files[items[i].ID]
	}
	return items, nil
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

// DeleteForOwner removes a submission and returns the bucket keys of all its
// files and thumbnails, so the caller can delete those too.
func (s *SubmissionStore) DeleteForOwner(ctx context.Context, id, ownerID string) ([]string, error) {
	var deleted int
	var keys []string
	err := s.db.QueryRow(ctx,
		`WITH target AS (
		     SELECT s.id
		     FROM submissions s
		     JOIN events e ON e.id = s.event_id
		     WHERE s.id = $1 AND e.owner_id = $2
		 ),
		 file_keys AS (
		     SELECT unnest(ARRAY[m.storage_key, m.thumbnail_key]) AS key
		     FROM media m JOIN target ON m.submission_id = target.id
		 ),
		 removed AS (
		     DELETE FROM submissions USING target WHERE submissions.id = target.id RETURNING submissions.id
		 )
		 SELECT (SELECT count(*) FROM removed),
		        COALESCE((SELECT array_agg(key) FROM file_keys WHERE key IS NOT NULL), '{}')`,
		id, ownerID,
	).Scan(&deleted, &keys)
	if err != nil {
		return nil, err
	}
	if deleted == 0 {
		return nil, ErrNotFound
	}
	return keys, nil
}
