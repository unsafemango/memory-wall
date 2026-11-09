package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// JobNotifyMemories is the job that emails an organizer about new memories.
// It's queued inside the submission's transaction.
const JobNotifyMemories = "notify_memories"

// notifyGap is the least time between emails about one event. Memories that
// arrive in between wait and go out together, so a busy event doesn't fill
// the organizer's inbox.
const notifyGap = "10 minutes"

// NotificationTarget is what's needed to tell an organizer about their event.
type NotificationTarget struct {
	EventTitle       string
	OwnerEmail       string
	CanNotify        bool
	UnsubscribeToken string
}

// UnseenMemory is a memory the organizer hasn't been told about yet.
type UnseenMemory struct {
	SenderName string
	Message    string
	Photos     int
	Videos     int
}

// UnseenMemories is a batch of memories to tell an organizer about. Previews
// holds only the first few of them, while Total counts them all.
type UnseenMemories struct {
	Previews    []UnseenMemory
	Total       int
	NeedsReview bool
	Through     time.Time
}

// queueNotification makes sure a notification job is waiting for the event.
// The first memory after a quiet spell goes out right away; later ones join a
// job that's waiting, or schedule one for when the gap is over.
func queueNotification(ctx context.Context, tx pgx.Tx, eventID string) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO jobs (type, payload, run_at)
		 SELECT $1::text, json_build_object('event_id', $2::uuid),
		        GREATEST(now(), COALESCE(e.last_notified_at + interval '`+notifyGap+`', now()))
		 FROM events e
		 WHERE e.id = $2::uuid AND NOT EXISTS (
		     SELECT 1 FROM jobs j
		     WHERE j.type = $1::text AND j.status = 'pending' AND j.payload->>'event_id' = $2::uuid::text
		 )`,
		JobNotifyMemories, eventID,
	)
	return err
}

// NotificationTarget looks up who to tell about an event. Returns ErrNotFound
// if the event has since been deleted.
func (s *SubmissionStore) NotificationTarget(ctx context.Context, eventID string) (NotificationTarget, error) {
	var t NotificationTarget
	err := s.db.QueryRow(ctx,
		`SELECT e.title, o.email, o.email_verified_at IS NOT NULL AND o.notify_new_memories, o.unsubscribe_token
		 FROM events e JOIN organizers o ON o.id = e.owner_id
		 WHERE e.id = $1`,
		eventID,
	).Scan(&t.EventTitle, &t.OwnerEmail, &t.CanNotify, &t.UnsubscribeToken)
	if errors.Is(err, pgx.ErrNoRows) {
		return t, ErrNotFound
	}
	return t, err
}

// ListUnseenMemories returns memories newer than the last notification, up to
// limit previews, oldest first. Rejected ones don't count. Total is 0 when
// there's nothing new.
func (s *SubmissionStore) ListUnseenMemories(ctx context.Context, eventID string, limit int) (UnseenMemories, error) {
	rows, err := s.db.Query(ctx,
		`SELECT sub.sender_name, COALESCE(sub.message, ''),
		        (SELECT count(*) FROM media m WHERE m.submission_id = sub.id AND m.mime_type LIKE 'image/%'),
		        (SELECT count(*) FROM media m WHERE m.submission_id = sub.id AND m.mime_type LIKE 'video/%'),
		        count(*) OVER (), bool_or(sub.status = 'pending') OVER (), max(sub.created_at) OVER ()
		 FROM submissions sub
		 JOIN events e ON e.id = sub.event_id
		 WHERE sub.event_id = $1 AND sub.created_at > e.notified_through AND sub.status <> 'rejected'
		 ORDER BY sub.created_at
		 LIMIT $2`,
		eventID, limit,
	)
	if err != nil {
		return UnseenMemories{}, err
	}
	defer rows.Close()

	var out UnseenMemories
	for rows.Next() {
		var m UnseenMemory
		if err := rows.Scan(&m.SenderName, &m.Message, &m.Photos, &m.Videos, &out.Total, &out.NeedsReview, &out.Through); err != nil {
			return UnseenMemories{}, err
		}
		out.Previews = append(out.Previews, m)
	}
	return out, rows.Err()
}

// MarkNotified records that memories up to through are dealt with. emailed
// says whether an email really went out, which starts the gap until the next
// one. Without one, memories from while emails were off don't all arrive the
// moment they're turned back on.
func (s *SubmissionStore) MarkNotified(ctx context.Context, eventID string, through time.Time, emailed bool) error {
	_, err := s.db.Exec(ctx,
		`UPDATE events SET notified_through = $2,
		        last_notified_at = CASE WHEN $3::boolean THEN now() ELSE last_notified_at END
		 WHERE id = $1`,
		eventID, through, emailed,
	)
	return err
}
