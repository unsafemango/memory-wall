package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Event struct {
	ID                string `json:"id"`
	OwnerID           string `json:"-"`
	Title             string `json:"title"`
	Slug              string `json:"slug"`
	EventDate         string `json:"event_date,omitempty"` // "2026-11-14", empty when not set
	ClosingMessage    string `json:"closing_message,omitempty"`
	ModerationEnabled bool   `json:"moderation_enabled"`
	Occasion          string `json:"occasion"`
	HonoreeName       string `json:"honoree_name,omitempty"`
	EventTime         string `json:"event_time,omitempty"` // "18:30", empty when not set
	Timezone          string `json:"timezone"`             // IANA name, like America/Toronto
	// EventAt is the exact moment of the event, worked out from the date, time,
	// and timezone. Nil when the event has no date.
	EventAt        *time.Time `json:"event_at,omitempty"`
	HideUntilEvent bool       `json:"hide_until_event"`
	// ShareToken is the event's reusable link. It's never sent as-is; the
	// organizer endpoints turn it into ShareURL.
	ShareToken   string `json:"-"`
	ShareEnabled bool   `json:"-"`
}

// Hidden reports whether the event's memories are still under wraps.
func (e Event) Hidden(now time.Time) bool {
	return e.HideUntilEvent && e.EventAt != nil && now.Before(*e.EventAt)
}

type EventStore struct {
	db *pgxpool.Pool
}

func NewEventStore(db *pgxpool.Pool) *EventStore {
	return &EventStore{db: db}
}

const eventColumns = `id, COALESCE(owner_id::text, ''), title, slug, COALESCE(event_date::text, ''),
	COALESCE(closing_message, ''), moderation_enabled, occasion, COALESCE(honoree_name, ''),
	COALESCE(event_time, ''), timezone, event_at, hide_until_event, share_token, share_enabled`

func scanEvent(row interface{ Scan(...any) error }) (Event, error) {
	var e Event
	err := row.Scan(&e.ID, &e.OwnerID, &e.Title, &e.Slug, &e.EventDate, &e.ClosingMessage, &e.ModerationEnabled,
		&e.Occasion, &e.HonoreeName, &e.EventTime, &e.Timezone, &e.EventAt, &e.HideUntilEvent,
		&e.ShareToken, &e.ShareEnabled)
	return e, err
}

func (s *EventStore) GetBySlug(ctx context.Context, slug string) (Event, error) {
	return scanEvent(s.db.QueryRow(ctx, `SELECT `+eventColumns+` FROM events WHERE slug = $1`, slug))
}

func (s *EventStore) GetByID(ctx context.Context, id string) (Event, error) {
	return scanEvent(s.db.QueryRow(ctx, `SELECT `+eventColumns+` FROM events WHERE id = $1`, id))
}

func (s *EventStore) Create(ctx context.Context, e Event) (string, error) {
	var id string
	err := s.db.QueryRow(ctx,
		`INSERT INTO events (owner_id, title, slug, event_date, closing_message, moderation_enabled, occasion, honoree_name,
		                     event_time, timezone, event_at, hide_until_event)
		 VALUES ($1, $2, $3, NULLIF($4, '')::date, NULLIF($5, ''), $6, COALESCE(NULLIF($7, ''), 'other'), NULLIF($8, ''),
		         NULLIF($9, ''), COALESCE(NULLIF($10, ''), 'UTC'), $11, $12)
		 RETURNING id`,
		e.OwnerID, e.Title, e.Slug, e.EventDate, e.ClosingMessage, e.ModerationEnabled, e.Occasion, e.HonoreeName,
		e.EventTime, e.Timezone, e.EventAt, e.HideUntilEvent,
	).Scan(&id)
	return id, mapDuplicate(err)
}

// EventSummary is an event in the organizer's list, with how many
// submissions are waiting and how many are live.
type EventSummary struct {
	Event
	PendingCount  int `json:"pending_count"`
	ApprovedCount int `json:"approved_count"`
}

// ListByOwner returns an organizer's events with submission counts, newest
// first.
func (s *EventStore) ListByOwner(ctx context.Context, ownerID string) ([]EventSummary, error) {
	rows, err := s.db.Query(ctx,
		`SELECT e.id, COALESCE(e.owner_id::text, ''), e.title, e.slug, COALESCE(e.event_date::text, ''),
		        COALESCE(e.closing_message, ''), e.moderation_enabled, e.occasion, COALESCE(e.honoree_name, ''),
		        COALESCE(e.event_time, ''), e.timezone, e.event_at, e.hide_until_event,
		        e.share_token, e.share_enabled,
		        COUNT(s.id) FILTER (WHERE s.status = 'pending'),
		        COUNT(s.id) FILTER (WHERE s.status = 'approved')
		 FROM events e
		 LEFT JOIN submissions s ON s.event_id = e.id
		 WHERE e.owner_id = $1
		 GROUP BY e.id
		 ORDER BY e.created_at DESC`,
		ownerID,
	)
	if err != nil {
		return nil, err
	}
	events, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (EventSummary, error) {
		var e EventSummary
		err := row.Scan(&e.ID, &e.OwnerID, &e.Title, &e.Slug, &e.EventDate, &e.ClosingMessage,
			&e.ModerationEnabled, &e.Occasion, &e.HonoreeName, &e.EventTime, &e.Timezone, &e.EventAt, &e.HideUntilEvent,
			&e.ShareToken, &e.ShareEnabled,
			&e.PendingCount, &e.ApprovedCount)
		return e, err
	})
	if events == nil {
		events = []EventSummary{}
	}
	return events, err
}

// GetOwned returns an event only if ownerID owns it, otherwise ErrNotFound.
func (s *EventStore) GetOwned(ctx context.Context, id, ownerID string) (Event, error) {
	e, err := scanEvent(s.db.QueryRow(ctx,
		`SELECT `+eventColumns+` FROM events WHERE id = $1 AND owner_id = $2`,
		id, ownerID,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return e, ErrNotFound
	}
	return e, err
}

// Update changes an event's settings. The owner check is in the WHERE clause,
// so there's no way to edit someone else's event.
func (s *EventStore) Update(ctx context.Context, e Event) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE events SET title = $3, slug = $4, event_date = NULLIF($5, '')::date,
		        closing_message = NULLIF($6, ''), moderation_enabled = $7,
		        occasion = COALESCE(NULLIF($8, ''), 'other'), honoree_name = NULLIF($9, ''),
		        event_time = NULLIF($10, ''), timezone = COALESCE(NULLIF($11, ''), 'UTC'), event_at = $12,
		        hide_until_event = $13
		 WHERE id = $1 AND owner_id = $2`,
		e.ID, e.OwnerID, e.Title, e.Slug, e.EventDate, e.ClosingMessage, e.ModerationEnabled, e.Occasion, e.HonoreeName,
		e.EventTime, e.Timezone, e.EventAt, e.HideUntilEvent,
	)
	if err != nil {
		return mapDuplicate(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetShareEnabled turns an event's share link on or off.
func (s *EventStore) SetShareEnabled(ctx context.Context, id, ownerID string, enabled bool) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE events SET share_enabled = $3 WHERE id = $1 AND owner_id = $2`,
		id, ownerID, enabled,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ResetShareToken gives the event a new share link. The old one stops
// working straight away.
func (s *EventStore) ResetShareToken(ctx context.Context, id, ownerID string) (string, error) {
	var token string
	err := s.db.QueryRow(ctx,
		`UPDATE events SET share_token = encode(gen_random_bytes(32), 'hex')
		 WHERE id = $1 AND owner_id = $2
		 RETURNING share_token`,
		id, ownerID,
	).Scan(&token)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return token, err
}
