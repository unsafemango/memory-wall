package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/unsafemango/memory-wall/internal/email"
	"github.com/unsafemango/memory-wall/internal/store"
)

// TestNotifierBatchesMemories runs the whole path: submitting memories queues
// one job, and the job sends one email covering all of them.
func TestNotifierBatchesMemories(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping database test")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(db.Close)

	var mu sync.Mutex
	var sent []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		sent = append(sent, body)
		mu.Unlock()
		w.Write([]byte(`{"id":"1"}`))
	}))
	t.Cleanup(srv.Close)

	unique := fmt.Sprintf("notify-%d", time.Now().UnixNano())
	organizers := store.NewOrganizerStore(db)
	ownerID, err := organizers.Create(ctx, unique+"@test.example", "hash", "Owner")
	if err != nil {
		t.Fatalf("create organizer: %v", err)
	}
	t.Cleanup(func() { db.Exec(context.Background(), `DELETE FROM organizers WHERE id = $1`, ownerID) })

	events := store.NewEventStore(db)
	eventID, err := events.Create(ctx, store.Event{OwnerID: ownerID, Title: "Sam's 30th", Slug: unique, ModerationEnabled: true})
	if err != nil {
		t.Fatalf("create event: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(context.Background(), `DELETE FROM jobs WHERE payload->>'event_id' = $1`, eventID)
	})

	submissions := store.NewSubmissionStore(db)
	notifier := NewMemoryNotifier(email.New("re_test", "n@mail.example", srv.URL), submissions, "https://site.example")
	payload := []byte(`{"event_id":"` + eventID + `"}`)

	submit := func(name string) {
		t.Helper()
		_, err := submissions.Create(ctx, store.Submission{EventID: eventID, SenderName: name, Type: "note", Message: "Hi from " + name, Status: "pending"}, nil)
		if err != nil {
			t.Fatalf("submit: %v", err)
		}
	}
	pendingJobs := func() int {
		t.Helper()
		var n int
		err := db.QueryRow(ctx, `SELECT count(*) FROM jobs WHERE type = $1 AND status = 'pending' AND payload->>'event_id' = $2`, store.JobNotifyMemories, eventID).Scan(&n)
		if err != nil {
			t.Fatalf("count jobs: %v", err)
		}
		return n
	}

	// Unconfirmed: nothing is sent, and those memories aren't saved up for later.
	submit("Early")
	if err := notifier.Handle(ctx, payload); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if len(sent) != 0 {
		t.Fatalf("an unconfirmed address got email: %v", sent)
	}
	db.Exec(ctx, `UPDATE jobs SET status = 'done' WHERE payload->>'event_id' = $1`, eventID)
	if _, err := db.Exec(ctx, `UPDATE organizers SET email_verified_at = now() WHERE id = $1`, ownerID); err != nil {
		t.Fatal(err)
	}

	// Three memories share one waiting job and one email.
	submit("Rosa")
	submit("Ben")
	submit("Kai")
	if got := pendingJobs(); got != 1 {
		t.Fatalf("pending jobs = %d, want 1", got)
	}
	if err := notifier.Handle(ctx, payload); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if len(sent) != 1 {
		t.Fatalf("emails = %d, want 1", len(sent))
	}
	if sent[0]["subject"] != "3 new memories for Sam's 30th" {
		t.Errorf("subject = %v", sent[0]["subject"])
	}
	if text := sent[0]["text"].(string); strings.Contains(text, "Early") || !strings.Contains(text, "Rosa") {
		t.Errorf("email should only cover memories since confirming: %s", text)
	}
	db.Exec(ctx, `UPDATE jobs SET status = 'done' WHERE payload->>'event_id' = $1`, eventID)

	// Running it again has nothing new to say.
	if err := notifier.Handle(ctx, payload); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if len(sent) != 1 {
		t.Fatalf("a second run sent another email: %d", len(sent))
	}

	// The next memory waits out the gap instead of emailing straight away.
	submit("Lee")
	var wait time.Duration
	err = db.QueryRow(ctx, `SELECT run_at - now() FROM jobs WHERE type = $1 AND status = 'pending' AND payload->>'event_id' = $2`, store.JobNotifyMemories, eventID).Scan(&wait)
	if err != nil {
		t.Fatalf("read job: %v", err)
	}
	if wait < 9*time.Minute {
		t.Fatalf("next email is due in %v, want about 10 minutes", wait)
	}

	// Turned off: nothing is sent.
	if err := organizers.SetNotifications(ctx, ownerID, false); err != nil {
		t.Fatal(err)
	}
	if err := notifier.Handle(ctx, payload); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if len(sent) != 1 {
		t.Fatalf("got email with notifications off: %d", len(sent))
	}
}
