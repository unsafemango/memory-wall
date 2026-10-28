package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testDB connects to the database in DATABASE_URL, or skips the test.
// Point it at the Docker Postgres with the migrations applied.
func testDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping database test")
	}
	db, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(db.Close)
	return db
}

func randomSuffix(t *testing.T) string {
	t.Helper()
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(b)
}

type fixture struct {
	ownerID   string
	otherID   string
	event     Event
	inviteID  string
	inviteRaw string
}

// newFixture creates two organizers, an event owned by the first with
// moderation on, and one invitation. Everything is deleted when the test ends
// (deleting an organizer cascades to its events and their rows).
func newFixture(t *testing.T, db *pgxpool.Pool) fixture {
	t.Helper()
	ctx := context.Background()
	suffix := randomSuffix(t)
	organizers := NewOrganizerStore(db)

	ownerID, err := organizers.Create(ctx, "owner-"+suffix+"@test.example", "hash", "Owner")
	if err != nil {
		t.Fatalf("create owner: %v", err)
	}
	otherID, err := organizers.Create(ctx, "other-"+suffix+"@test.example", "hash", "Other")
	if err != nil {
		t.Fatalf("create other organizer: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(context.Background(), `DELETE FROM organizers WHERE id = ANY($1)`, []string{ownerID, otherID})
	})

	events := NewEventStore(db)
	event := Event{OwnerID: ownerID, Title: "Test event", Slug: "test-" + suffix, ModerationEnabled: true}
	event.ID, err = events.Create(ctx, event)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	f := fixture{ownerID: ownerID, otherID: otherID, event: event, inviteRaw: "token-" + suffix}
	f.inviteID = createInvitation(t, db, event.ID, f.inviteRaw, time.Now().Add(time.Hour))
	return f
}

func createInvitation(t *testing.T, db *pgxpool.Pool, eventID, raw string, expiresAt time.Time) string {
	t.Helper()
	ctx := context.Background()
	invitations := NewInvitationStore(db)
	// Hashing here mirrors auth.HashToken; store tests don't import auth.
	hash := sha256Hex(raw)
	id, err := invitations.Create(ctx, eventID, hash, "Guest", expiresAt)
	if err != nil {
		t.Fatalf("create invitation: %v", err)
	}
	return id
}
