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

	// Find the seeded event and invitation.
	var eventID, invitationID string
	err = db.QueryRow(ctx,
		`SELECT i.event_id, i.id FROM invitations i WHERE i.token_hash = 'dev-token-plaintext'`,
	).Scan(&eventID, &invitationID)
	if err != nil {
		t.Fatalf("find seed invitation (did you run scripts/seed.sql?): %v", err)
	}

	submissions := NewSubmissionStore(db)
	want := Submission{
		EventID:      eventID,
		InvitationID: invitationID,
		SenderName:   "Round Trip Test",
		Type:         "note",
		Message:      "hello from the test",
		Status:       "approved",
	}

	id, err := submissions.Create(ctx, want)
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
