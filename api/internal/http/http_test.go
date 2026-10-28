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

	"github.com/go-chi/chi/v5"
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
	if _, err := invitations.Create(ctx, eventID, store.NewInvitation{TokenHash: hash, RecipientName: "Guest"}, time.Now().Add(time.Hour)); err != nil {
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

// TestShareLinkIsReusable checks an event's share link takes many
// submissions, stops working when turned off, and dies when reset.
func TestShareLinkIsReusable(t *testing.T) {
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

	_, hash, _ := auth.NewToken()
	organizers := store.NewOrganizerStore(db)
	ownerID, err := organizers.Create(ctx, "share-test-"+hash[:8]+"@test.example", "hash", "Owner")
	if err != nil {
		t.Fatalf("create organizer: %v", err)
	}
	t.Cleanup(func() { db.Exec(context.Background(), `DELETE FROM organizers WHERE id = $1`, ownerID) })

	events := store.NewEventStore(db)
	eventID, err := events.Create(ctx, store.Event{OwnerID: ownerID, Title: "Share test", Slug: "share-test-" + hash[:8], ModerationEnabled: true})
	if err != nil {
		t.Fatalf("create event: %v", err)
	}
	event, err := events.GetByID(ctx, eventID)
	if err != nil || len(event.ShareToken) != 64 || !event.ShareEnabled {
		t.Fatalf("new event should get an enabled share token: %+v, %v", event, err)
	}

	srv := NewServer(Deps{
		Events:      events,
		Invitations: store.NewInvitationStore(db),
		Submissions: store.NewSubmissionStore(db),
		SiteURL:     "http://localhost:3000",
	})
	router := srv.Routes()
	submit := func(token, name, ip string) int {
		r := httptest.NewRequest(http.MethodPost, "/api/invitations/"+token+"/submissions",
			strings.NewReader(`{"sender_name":"`+name+`","message":"Congratulations!"}`))
		r.RemoteAddr = ip + ":1234"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w.Code
	}

	for i, name := range []string{"Ana", "Ben", "Cy"} {
		if code := submit(event.ShareToken, name, "192.0.2."+string(rune('1'+i))); code != http.StatusCreated {
			t.Fatalf("submission %d through the share link: got %d, want 201", i+1, code)
		}
	}

	if err := events.SetShareEnabled(ctx, eventID, ownerID, false); err != nil {
		t.Fatalf("turn off: %v", err)
	}
	if code := submit(event.ShareToken, "Dee", "192.0.2.9"); code != http.StatusGone {
		t.Fatalf("turned-off share link: got %d, want 410", code)
	}

	if err := events.SetShareEnabled(ctx, eventID, ownerID, true); err != nil {
		t.Fatalf("turn on: %v", err)
	}
	newToken, err := events.ResetShareToken(ctx, eventID, ownerID)
	if err != nil || newToken == event.ShareToken {
		t.Fatalf("reset should give a new token: %q, %v", newToken, err)
	}
	if code := submit(event.ShareToken, "Eve", "192.0.2.9"); code != http.StatusNotFound {
		t.Fatalf("old token after reset: got %d, want 404", code)
	}
	if code := submit(newToken, "Eve", "192.0.2.9"); code != http.StatusCreated {
		t.Fatalf("new token after reset: got %d, want 201", code)
	}

	if _, err := events.ResetShareToken(ctx, eventID, "00000000-0000-0000-0000-000000000000"); err == nil {
		t.Fatal("someone else shouldn't be able to reset the share link")
	}
}

// TestInvitationLinksCanBeShownAgain checks the organizer's invitation list
// gives back a working link for open invitations, and none once it's used.
func TestInvitationLinksCanBeShownAgain(t *testing.T) {
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

	_, suffix, _ := auth.NewToken()
	organizers := store.NewOrganizerStore(db)
	ownerID, err := organizers.Create(ctx, "links-test-"+suffix[:8]+"@test.example", "hash", "Owner")
	if err != nil {
		t.Fatalf("create organizer: %v", err)
	}
	t.Cleanup(func() { db.Exec(context.Background(), `DELETE FROM organizers WHERE id = $1`, ownerID) })
	events := store.NewEventStore(db)
	eventID, err := events.Create(ctx, store.Event{OwnerID: ownerID, Title: "Links test", Slug: "links-test-" + suffix[:8]})
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	sealer, _ := auth.NewSealer("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")
	srv := NewServer(Deps{
		Events:      events,
		Invitations: store.NewInvitationStore(db),
		Submissions: store.NewSubmissionStore(db),
		SiteURL:     "http://localhost:3000",
		LinkSealer:  sealer,
	})
	call := func(method, path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r = r.WithContext(context.WithValue(r.Context(), organizerIDKey, ownerID))
		w := httptest.NewRecorder()
		switch {
		case strings.HasSuffix(path, "/bulk"):
			srv.withEventID(srv.createInvitations, eventID).ServeHTTP(w, r)
		default:
			srv.withEventID(srv.listInvitations, eventID).ServeHTTP(w, r)
		}
		return w
	}

	created := call(http.MethodPost, "/bulk", `{"names":["Emma","Yray"]}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", created.Code, created.Body.String())
	}
	var made []struct {
		ID        string `json:"id"`
		InviteURL string `json:"invite_url"`
	}
	json.NewDecoder(created.Body).Decode(&made)

	var listed []struct {
		ID        string `json:"id"`
		InviteURL string `json:"invite_url"`
	}
	json.NewDecoder(call(http.MethodGet, "/", "").Body).Decode(&listed)
	byID := map[string]string{}
	for _, l := range listed {
		byID[l.ID] = l.InviteURL
	}
	for _, m := range made {
		if byID[m.ID] == "" || byID[m.ID] != m.InviteURL {
			t.Fatalf("list should give back the same link: made %q, listed %q", m.InviteURL, byID[m.ID])
		}
	}

	// Once someone posts with a link, the list stops offering it.
	token := made[0].InviteURL[strings.LastIndex(made[0].InviteURL, "/")+1:]
	r := httptest.NewRequest(http.MethodPost, "/api/invitations/"+token+"/submissions", strings.NewReader(`{"sender_name":"Emma","message":"Hi!"}`))
	r.RemoteAddr = "192.0.2.50:1"
	w := httptest.NewRecorder()
	srv.Routes().ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("submit: %d %s", w.Code, w.Body.String())
	}
	listed = nil
	json.NewDecoder(call(http.MethodGet, "/", "").Body).Decode(&listed)
	for _, l := range listed {
		if l.ID == made[0].ID && l.InviteURL != "" {
			t.Fatal("a used link shouldn't be offered again")
		}
	}
}

// withEventID runs h as if the route had matched {eventID}.
func (s *Server) withEventID(h http.HandlerFunc, eventID string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("eventID", eventID)
		h(w, r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx)))
	})
}
