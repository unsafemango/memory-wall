package store

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Needs a running Postgres with the seed data from scripts/seed.sql.
func TestSubmissionRoundTrip(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping database test")
	}

	ctx := context.Background()
	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer db.Close()

	// Find the seeded event. The test leaves InvitationID empty so it doesn't
	// use up the seeded invitation.
	var eventID string
	err = db.QueryRow(ctx, `SELECT id FROM events WHERE slug = 'sarah'`).Scan(&eventID)
	if err != nil {
		t.Fatalf("find seed event (did you run scripts/seed.sql?): %v", err)
	}

	submissions := NewSubmissionStore(db)
	want := Submission{
		EventID:    eventID,
		SenderName: "Round Trip Test",
		Type:       "note",
		Message:    "hello from the test",
		Status:     "approved",
	}

	id, err := submissions.Create(ctx, want, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(context.Background(), `DELETE FROM submissions WHERE id = $1`, id)
	})

	got, err := submissions.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	want.ID = id
	if got != want {
		t.Fatalf("round trip mismatch\n got: %+v\nwant: %+v", got, want)
	}
}
