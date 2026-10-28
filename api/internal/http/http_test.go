package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/unsafemango/memory-wall/internal/auth"
	"github.com/unsafemango/memory-wall/internal/store"
)

func TestDecodeEventValidation(t *testing.T) {
	tests := []struct {
		name string
		body string
		ok   bool
	}{
		{"valid", `{"title":"Sam turns 30","slug":"sam-30","event_date":"2026-11-14"}`, true},
		{"no date", `{"title":"Sam turns 30","slug":"sam-30"}`, true},
		{"missing title", `{"title":"  ","slug":"sam-30"}`, false},
		{"uppercase slug", `{"title":"T","slug":"Sam-30"}`, false},
		{"slug with spaces", `{"title":"T","slug":"sam 30"}`, false},
		{"trailing dash", `{"title":"T","slug":"sam-"}`, false},
		{"reserved slug", `{"title":"T","slug":"dashboard"}`, false},
		{"bad date", `{"title":"T","slug":"sam","event_date":"14/11/2026"}`, false},
		{"not json", `title=T`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			_, ok := decodeEvent(w, r)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v (response %d %s)", ok, tt.ok, w.Code, w.Body.String())
			}
			if !ok && w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", w.Code)
			}
		})
	}
}

func TestSameOriginWritesBlocksOtherSites(t *testing.T) {
	s := &Server{siteURL: "http://localhost:3000"}
	h := s.sameOriginWrites(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		method, origin string
		want           int
	}{
		{http.MethodPost, "http://localhost:3000", http.StatusOK},
		{http.MethodPost, "https://evil.example", http.StatusForbidden},
		{http.MethodDelete, "https://evil.example", http.StatusForbidden},
		{http.MethodPost, "", http.StatusOK}, // non-browser clients such as curl
		{http.MethodGet, "https://evil.example", http.StatusOK},
	}
	for _, tt := range tests {
		r := httptest.NewRequest(tt.method, "/", nil)
		if tt.origin != "" {
			r.Header.Set("Origin", tt.origin)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tt.want {
			t.Errorf("%s from %q: got %d, want %d", tt.method, tt.origin, w.Code, tt.want)
		}
	}
}

func TestKeyedLimiter(t *testing.T) {
	l := newKeyedLimiter(time.Hour, 3)
	for i := 0; i < 3; i++ {
		if !l.allow("a") {
			t.Fatalf("request %d should be allowed within the burst", i+1)
		}
	}
	if l.allow("a") {
		t.Fatal("request past the burst should be blocked")
	}
	if !l.allow("b") {
		t.Fatal("a different key has its own limit")
	}
}

func TestWriteErrorShape(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusNotFound, "not_found", "invitation not found")

	var body struct {
		Error struct{ Code, Message string } `json:"error"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if w.Code != 404 || body.Error.Code != "not_found" || body.Error.Message != "invitation not found" {
		t.Fatalf("unexpected response %d %+v", w.Code, body)
	}
}

// TestSubmitNoteThroughInvitation runs the guest flow through the real router:
// a note is saved as pending on a moderated event, and the same link then
// stops working. Needs DATABASE_URL.
func TestSubmitNoteThroughInvitation(t *testing.T) {
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

	raw, hash, _ := auth.NewToken()
	organizers := store.NewOrganizerStore(db)
	ownerID, err := organizers.Create(ctx, "http-test-"+hash[:8]+"@test.example", "hash", "Owner")
	if err != nil {
		t.Fatalf("create organizer: %v", err)
	}
	t.Cleanup(func() { db.Exec(context.Background(), `DELETE FROM organizers WHERE id = $1`, ownerID) })

	events := store.NewEventStore(db)
	eventID, err := events.Create(ctx, store.Event{OwnerID: ownerID, Title: "HTTP test", Slug: "http-test-" + hash[:8], ModerationEnabled: true})
	if err != nil {
		t.Fatalf("create event: %v", err)
	}
	invitations := store.NewInvitationStore(db)
	if _, err := invitations.Create(ctx, eventID, hash, "Guest", time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("create invitation: %v", err)
	}

	submissions := store.NewSubmissionStore(db)
	srv := NewServer(Deps{Events: events, Invitations: invitations, Submissions: submissions, SiteURL: "http://localhost:3000"})
	router := srv.Routes()

	post := func(body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "/api/invitations/"+raw+"/submissions", strings.NewReader(body))
		r.RemoteAddr = "192.0.2.1:1234"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}

	if w := post(`{"sender_name":"","message":"hi"}`); w.Code != http.StatusBadRequest {
		t.Fatalf("missing name: got %d, want 400", w.Code)
	}

	w := post(`{"sender_name":"Jo","message":"Happy 30th!"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("submit: got %d %s", w.Code, w.Body.String())
	}
	var created struct{ ID string }
	json.NewDecoder(w.Body).Decode(&created)

	got, err := submissions.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != "pending" || got.Message != "Happy 30th!" {
		t.Fatalf("saved submission = %+v, want a pending note", got)
	}

	if w := post(`{"sender_name":"Jo","message":"again"}`); w.Code != http.StatusGone {
		t.Fatalf("reusing the link: got %d, want 410", w.Code)
	}
}
