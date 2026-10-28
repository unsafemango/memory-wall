package live

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestHubSendsOnlyToTheRightEvent(t *testing.T) {
	hub := NewHub()
	a, stopA := hub.Subscribe("event-a")
	b, stopB := hub.Subscribe("event-b")
	defer stopB()

	hub.Publish(Change{EventID: "event-a", Op: "INSERT"})
	select {
	case c := <-a:
		if c.Op != "INSERT" {
			t.Fatalf("got %+v", c)
		}
	case <-time.After(time.Second):
		t.Fatal("subscriber to event-a got nothing")
	}
	select {
	case c := <-b:
		t.Fatalf("subscriber to event-b shouldn't get %+v", c)
	default:
	}

	stopA()
	hub.Publish(Change{EventID: "event-a", Op: "UPDATE"})
	if len(hub.subs["event-a"]) != 0 {
		t.Fatal("unsubscribing should remove the channel")
	}
}

func TestPublishNeverBlocks(t *testing.T) {
	hub := NewHub()
	_, stop := hub.Subscribe("event")
	defer stop()
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			hub.Publish(Change{EventID: "event"})
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("publishing to a full subscriber blocked")
	}
}

func TestChangePublic(t *testing.T) {
	tests := []struct {
		c    Change
		want bool
	}{
		{Change{Op: "INSERT", Status: "pending"}, false},
		{Change{Op: "INSERT", Status: "approved"}, true},
		{Change{Op: "UPDATE", Status: "approved", OldStatus: "pending"}, true},
		{Change{Op: "UPDATE", Status: "rejected", OldStatus: "approved"}, true},
		{Change{Op: "UPDATE", Status: "rejected", OldStatus: "pending"}, false},
		{Change{Op: "DELETE", OldStatus: "approved"}, true},
		{Change{Op: "DELETE", OldStatus: "pending"}, false},
		{Change{Op: "RESYNC"}, true},
	}
	for _, tt := range tests {
		if got := tt.c.Public(); got != tt.want {
			t.Errorf("%+v: Public() = %v, want %v", tt.c, got, tt.want)
		}
	}
}

// TestListenerDeliversDatabaseChanges checks a real change in Postgres
// reaches a subscriber. Needs DATABASE_URL with migrations applied.
func TestListenerDeliversDatabaseChanges(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping database test")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(db.Close)

	var eventID string
	if err := db.QueryRow(ctx, `INSERT INTO events (title, slug) VALUES ('Live test', 'live-test-'||substr(md5(random()::text), 1, 8)) RETURNING id`).Scan(&eventID); err != nil {
		t.Fatalf("create event: %v", err)
	}
	t.Cleanup(func() { db.Exec(context.Background(), `DELETE FROM events WHERE id = $1`, eventID) })

	hub := NewHub()
	changes, stop := hub.Subscribe(eventID)
	defer stop()
	go Listen(ctx, db, hub, slog.New(slog.NewTextHandler(io.Discard, nil)))
	time.Sleep(300 * time.Millisecond) // let LISTEN start

	var subID string
	if err := db.QueryRow(ctx, `INSERT INTO submissions (event_id, sender_name, type, message, status) VALUES ($1, 'Live', 'note', 'hi', 'pending') RETURNING id`, eventID).Scan(&subID); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if _, err := db.Exec(ctx, `UPDATE submissions SET status = 'approved' WHERE id = $1`, subID); err != nil {
		t.Fatalf("approve: %v", err)
	}

	want := []Change{
		{EventID: eventID, SubmissionID: subID, Op: "INSERT", Status: "pending"},
		{EventID: eventID, SubmissionID: subID, Op: "UPDATE", Status: "approved", OldStatus: "pending"},
	}
	for _, w := range want {
		select {
		case got := <-changes:
			if got != w {
				t.Fatalf("got %+v, want %+v", got, w)
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("timed out waiting for %+v", w)
		}
	}
}
