package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Event struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Slug      string `json:"slug"`
	EventDate string `json:"event_date,omitempty"` // "2026-11-14", empty when not set
}

type EventStore struct {
	db *pgxpool.Pool
}

func NewEventStore(db *pgxpool.Pool) *EventStore {
	return &EventStore{db: db}
}

func (s *EventStore) GetBySlug(ctx context.Context, slug string) (Event, error) {
	var e Event
	err := s.db.QueryRow(ctx,
		`SELECT id, title, slug, COALESCE(event_date::text, '') FROM events WHERE slug = $1`,
		slug,
	).Scan(&e.ID, &e.Title, &e.Slug, &e.EventDate)
	return e, err
}
