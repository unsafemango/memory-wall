package store

import (
	"context"

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
}

type EventStore struct {
	db *pgxpool.Pool
}

func NewEventStore(db *pgxpool.Pool) *EventStore {
	return &EventStore{db: db}
}

const eventColumns = `id, COALESCE(owner_id::text, ''), title, slug, COALESCE(event_date::text, ''),
	COALESCE(closing_message, ''), moderation_enabled`

func scanEvent(row interface{ Scan(...any) error }) (Event, error) {
	var e Event
	err := row.Scan(&e.ID, &e.OwnerID, &e.Title, &e.Slug, &e.EventDate, &e.ClosingMessage, &e.ModerationEnabled)
	return e, err
}

func (s *EventStore) GetBySlug(ctx context.Context, slug string) (Event, error) {
	return scanEvent(s.db.QueryRow(ctx, `SELECT `+eventColumns+` FROM events WHERE slug = $1`, slug))
}

func (s *EventStore) GetByID(ctx context.Context, id string) (Event, error) {
	return scanEvent(s.db.QueryRow(ctx, `SELECT `+eventColumns+` FROM events WHERE id = $1`, id))
}
