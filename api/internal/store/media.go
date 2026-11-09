package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// JobProcessVideo is the job type queued for every new video. It lives here
// because the job is queued inside the submission's transaction.
const JobProcessVideo = "process_video"

type Media struct {
	StorageKey string
	MimeType   string
	FileSize   int64
}

// VideoInfo is what the video job learns about an uploaded video.
type VideoInfo struct {
	Width           int
	Height          int
	DurationSeconds float64
	ThumbnailKey    string
}

type MediaStore struct {
	db *pgxpool.Pool
}

func NewMediaStore(db *pgxpool.Pool) *MediaStore {
	return &MediaStore{db: db}
}

// StorageKeyFor returns the bucket key of one file.
func (s *MediaStore) StorageKeyFor(ctx context.Context, mediaID string) (string, error) {
	var key string
	err := s.db.QueryRow(ctx, `SELECT storage_key FROM media WHERE id = $1`, mediaID).Scan(&key)
	return key, err
}

func (s *MediaStore) SetVideoInfo(ctx context.Context, mediaID string, info VideoInfo) error {
	_, err := s.db.Exec(ctx,
		`UPDATE media SET width = $2, height = $3, duration_seconds = $4, thumbnail_key = NULLIF($5, '')
		 WHERE id = $1`,
		mediaID, info.Width, info.Height, info.DurationSeconds, info.ThumbnailKey,
	)
	return err
}
