package store

import (
	"context"
)

type Media struct {
	SubmissionID string
	StorageKey   string
	MimeType     string
	FileSize     int64
}

// CreateWithMedia saves a submission and its uploaded file together, so a
// submission never exists without the media row it depends on.
func (s *SubmissionStore) CreateWithMedia(ctx context.Context, sub Submission, media Media) (string, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var id string
	err = tx.QueryRow(ctx,
		`INSERT INTO submissions (event_id, invitation_id, sender_name, type, message, status)
		 VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6) RETURNING id`,
		sub.EventID, sub.InvitationID, sub.SenderName, sub.Type, sub.Message, sub.Status,
	).Scan(&id)
	if err != nil {
		return "", err
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO media (submission_id, storage_key, mime_type, file_size) VALUES ($1, $2, $3, $4)`,
		id, media.StorageKey, media.MimeType, media.FileSize,
	)
	if err != nil {
		return "", err
	}

	return id, tx.Commit(ctx)
}
