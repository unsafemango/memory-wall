package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/unsafemango/memory-wall/internal/auth"
	mail "github.com/unsafemango/memory-wall/internal/email"
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

// TestAccountSettings covers changing a name, email, and password through the
// router, and that a password change signs out other devices. Needs DATABASE_URL.
func TestAccountSettings(t *testing.T) {
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
	email := "account-" + suffix[:8] + "@test.example"
	other := "taken-" + suffix[:8] + "@test.example"
	organizers := store.NewOrganizerStore(db)
	if _, err := organizers.Create(ctx, other, "hash", "Other"); err != nil {
		t.Fatalf("create other: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(context.Background(), `DELETE FROM organizers WHERE email LIKE $1`, "%"+suffix[:8]+"@test.example")
	})

	router := NewServer(Deps{Organizers: organizers, Sessions: store.NewSessionStore(db), SiteURL: "http://localhost:3000"}).Routes()
	ip := 0
	call := func(method, path, body, cookie string) *httptest.ResponseRecorder {
		ip++ // a new address each call, so rate limits stay out of the way
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.RemoteAddr = "192.0.2." + strconv.Itoa(100+ip) + ":1"
		if cookie != "" {
			r.AddCookie(&http.Cookie{Name: sessionCookie, Value: cookie})
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	login := func(mail, password string) string {
		w := call(http.MethodPost, "/api/auth/login", `{"email":"`+mail+`","password":"`+password+`"}`, "")
		for _, c := range w.Result().Cookies() {
			if c.Name == sessionCookie {
				return c.Value
			}
		}
		return ""
	}

	w := call(http.MethodPost, "/api/auth/signup", `{"email":"`+email+`","password":"first password","display_name":"First"}`, "")
	if w.Code != http.StatusOK {
		t.Fatalf("signup: %d %s", w.Code, w.Body.String())
	}
	laptop, phone := login(email, "first password"), login(email, "first password")
	if laptop == "" || phone == "" {
		t.Fatal("login should give session cookies")
	}

	const me = "/api/organizer/me"
	tests := []struct {
		name, method, path, body string
		want                     int
	}{
		{"rename without a password", http.MethodPatch, me, `{"email":"` + email + `","display_name":"Renamed"}`, 200},
		{"blank name", http.MethodPatch, me, `{"email":"` + email + `","display_name":" "}`, 400},
		{"new email without the password", http.MethodPatch, me, `{"email":"new-` + suffix[:8] + `@test.example","display_name":"Renamed"}`, 403},
		{"email already taken", http.MethodPatch, me, `{"email":"` + other + `","display_name":"Renamed","current_password":"first password"}`, 409},
		{"wrong current password", http.MethodPost, me + "/password", `{"current_password":"nope","new_password":"second password"}`, 403},
		{"new password too short", http.MethodPost, me + "/password", `{"current_password":"first password","new_password":"short"}`, 400},
		{"change password", http.MethodPost, me + "/password", `{"current_password":"first password","new_password":"second password"}`, 204},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if w := call(tt.method, tt.path, tt.body, laptop); w.Code != tt.want {
				t.Fatalf("got %d, want %d: %s", w.Code, tt.want, w.Body.String())
			}
		})
	}

	if w := call(http.MethodGet, me, "", laptop); w.Code != 200 || !strings.Contains(w.Body.String(), "Renamed") || strings.Contains(w.Body.String(), "argon2") {
		t.Fatalf("this device should stay signed in with the new name and no hash: %d %s", w.Code, w.Body.String())
	}
	if w := call(http.MethodGet, me, "", phone); w.Code != http.StatusUnauthorized {
		t.Fatalf("the other device should be signed out: got %d", w.Code)
	}
	if login(email, "first password") != "" {
		t.Fatal("the old password should stop working")
	}
	if login(email, "second password") == "" {
		t.Fatal("the new password should work")
	}
}

func TestEventMoment(t *testing.T) {
	at, err := eventMoment("2026-12-14", "18:30", "America/Toronto")
	if err != nil {
		t.Fatalf("eventMoment: %v", err)
	}
	if got := at.UTC().Format(time.RFC3339); got != "2026-12-14T23:30:00Z" {
		t.Fatalf("6:30pm in Toronto = %s, want 23:30 UTC", got)
	}
	midnight, _ := eventMoment("2026-12-14", "", "America/Toronto")
	if got := midnight.UTC().Format(time.RFC3339); got != "2026-12-14T05:00:00Z" {
		t.Fatalf("with no time it should be the start of the day there, got %s", got)
	}
	if none, err := eventMoment("", "18:30", "UTC"); none != nil || err != nil {
		t.Fatalf("no date should give no moment: %v, %v", none, err)
	}
	for _, bad := range [][2]string{{"18:30", "Not/AZone"}, {"6pm", "UTC"}} {
		if _, err := eventMoment("2026-12-14", bad[0], bad[1]); err == nil {
			t.Errorf("%v should be rejected", bad)
		}
	}
}

// TestHiddenEventKeepsMemoriesPrivate checks memories aren't sent before the
// reveal, even approved ones, and are afterwards. Needs DATABASE_URL.
func TestHiddenEventKeepsMemoriesPrivate(t *testing.T) {
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
	ownerID, err := store.NewOrganizerStore(db).Create(ctx, "hide-"+suffix[:8]+"@test.example", "hash", "Owner")
	if err != nil {
		t.Fatalf("create organizer: %v", err)
	}
	t.Cleanup(func() { db.Exec(context.Background(), `DELETE FROM organizers WHERE id = $1`, ownerID) })

	future := time.Now().Add(time.Hour)
	slug := "hide-" + suffix[:8]
	events := store.NewEventStore(db)
	event := store.Event{OwnerID: ownerID, Title: "Surprise", Slug: slug, EventDate: "2030-01-01", EventAt: &future, Timezone: "UTC", HideUntilEvent: true, ClosingMessage: "the closing surprise"}
	if event.ID, err = events.Create(ctx, event); err != nil {
		t.Fatalf("create event: %v", err)
	}
	const secret = "the surprise message"
	if _, err := store.NewSubmissionStore(db).Create(ctx, store.Submission{EventID: event.ID, SenderName: "Ana", Type: "note", Message: secret, Status: "approved"}, nil); err != nil {
		t.Fatalf("create submission: %v", err)
	}

	router := NewServer(Deps{Events: events, Submissions: store.NewSubmissionStore(db), SiteURL: "http://localhost:3000"}).Routes()
	get := func() string {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/events/"+slug+"/memories", nil))
		return w.Body.String()
	}
	if body := get(); strings.Contains(body, secret) || strings.Contains(body, "closing surprise") || !strings.Contains(body, `"locked":true`) {
		t.Fatalf("before the event, memories must not be sent: %s", body)
	}

	past := time.Now().Add(-time.Minute)
	event.EventAt = &past
	if err := events.Update(ctx, event); err != nil {
		t.Fatalf("update event: %v", err)
	}
	if body := get(); !strings.Contains(body, secret) || !strings.Contains(body, "closing surprise") {
		t.Fatalf("after the event, memories should be sent: %s", body)
	}
}

// TestEmailConfirmation follows the links an organizer gets by email.
func TestEmailConfirmation(t *testing.T) {
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

	var lastText string
	resend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Text string `json:"text"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		lastText = body.Text
		w.Write([]byte(`{"id":"1"}`))
	}))
	t.Cleanup(resend.Close)

	_, suffix, _ := auth.NewToken()
	address := "confirm-" + suffix[:8] + "@test.example"
	organizers := store.NewOrganizerStore(db)
	t.Cleanup(func() {
		db.Exec(context.Background(), `DELETE FROM organizers WHERE email LIKE $1`, "%"+suffix[:8]+"@test.example")
	})

	router := NewServer(Deps{
		Organizers: organizers,
		Sessions:   store.NewSessionStore(db),
		SiteURL:    "http://site.example",
		Mailer:     mail.New("re_test", "n@mail.example", resend.URL),
	}).Routes()
	ip := 0
	call := func(method, path, body, cookie string) *httptest.ResponseRecorder {
		ip++
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.RemoteAddr = "192.0.2." + strconv.Itoa(150+ip) + ":1"
		if cookie != "" {
			r.AddCookie(&http.Cookie{Name: sessionCookie, Value: cookie})
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	tokenFromEmail := func() string {
		t.Helper()
		_, rest, ok := strings.Cut(lastText, "/confirm-email/")
		if !ok {
			t.Fatalf("no link in email: %q", lastText)
		}
		return strings.Fields(rest)[0]
	}
	organizer := func() store.Organizer {
		t.Helper()
		o, err := organizers.GetByEmail(ctx, address)
		if err != nil {
			t.Fatalf("load organizer: %v", err)
		}
		return o
	}

	w := call(http.MethodPost, "/api/auth/signup", `{"email":"`+address+`","password":"long-enough-password","display_name":"Sam"}`, "")
	if w.Code >= 300 {
		t.Fatalf("signup: %d %s", w.Code, w.Body.String())
	}
	var cookie string
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookie {
			cookie = c.Value
		}
	}
	if organizer().EmailVerified {
		t.Fatal("a new account shouldn't start confirmed")
	}

	// Asking again replaces the first link.
	first := tokenFromEmail()
	if w := call(http.MethodPost, "/api/organizer/me/send-confirmation", "", cookie); w.Code != http.StatusNoContent {
		t.Fatalf("resend: %d %s", w.Code, w.Body.String())
	}
	if w := call(http.MethodPost, "/api/auth/confirm-email", `{"token":"`+first+`"}`, ""); w.Code != http.StatusGone {
		t.Fatalf("old link: got %d, want 410", w.Code)
	}

	token := tokenFromEmail()
	if w := call(http.MethodPost, "/api/auth/confirm-email", `{"token":"`+token+`"}`, ""); w.Code != http.StatusNoContent {
		t.Fatalf("confirm: %d %s", w.Code, w.Body.String())
	}
	if !organizer().EmailVerified {
		t.Fatal("should be confirmed")
	}
	if w := call(http.MethodPost, "/api/auth/confirm-email", `{"token":"`+token+`"}`, ""); w.Code != http.StatusGone {
		t.Fatalf("reused link: got %d, want 410", w.Code)
	}

	// Saving the same email keeps it confirmed; a new one has to be confirmed again.
	if w := call(http.MethodPatch, "/api/organizer/me", `{"email":"`+address+`","display_name":"Samira"}`, cookie); w.Code != http.StatusOK {
		t.Fatalf("rename: %d %s", w.Code, w.Body.String())
	}
	if !organizer().EmailVerified {
		t.Fatal("renaming shouldn't undo the confirmation")
	}
	oldLink := tokenFromEmail()
	address = "new-" + suffix[:8] + "@test.example"
	if w := call(http.MethodPatch, "/api/organizer/me", `{"email":"`+address+`","display_name":"Samira","current_password":"long-enough-password"}`, cookie); w.Code != http.StatusOK {
		t.Fatalf("change email: %d %s", w.Code, w.Body.String())
	}
	if organizer().EmailVerified {
		t.Fatal("a new email has to be confirmed")
	}
	if tokenFromEmail() == oldLink {
		t.Fatal("a new email should get its own link")
	}

	// Notifications: the setting, and the one-click link with no login.
	w = call(http.MethodPost, "/api/organizer/me/notifications", `{"enabled":false}`, cookie)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"notify_new_memories":false`) {
		t.Fatalf("turn off: %d %s", w.Code, w.Body.String())
	}
	call(http.MethodPost, "/api/organizer/me/notifications", `{"enabled":true}`, cookie)
	var unsubscribe string
	if err := db.QueryRow(ctx, `SELECT unsubscribe_token FROM organizers WHERE email = $1`, address).Scan(&unsubscribe); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(call(http.MethodGet, "/api/organizer/me", "", cookie).Body.String(), unsubscribe) {
		t.Fatal("the unsubscribe token shouldn't be in account data")
	}
	if w := call(http.MethodPost, "/api/unsubscribe/"+unsubscribe, "", ""); w.Code != http.StatusNoContent {
		t.Fatalf("unsubscribe: %d %s", w.Code, w.Body.String())
	}
	if organizer().NotifyNewMemories {
		t.Fatal("unsubscribing should turn notifications off")
	}
	if w := call(http.MethodPost, "/api/unsubscribe/nope", "", ""); w.Code != http.StatusNotFound {
		t.Fatalf("bad token: got %d, want 404", w.Code)
	}
}

// TestMemoryWithSeveralFiles checks a memory keeps its files in order, gets a
// job per video, and takes all of them with it when deleted.
func TestMemoryWithSeveralFiles(t *testing.T) {
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

	raw, suffix, _ := auth.NewToken()
	organizers := store.NewOrganizerStore(db)
	ownerID, err := organizers.Create(ctx, "files-"+suffix[:8]+"@test.example", "hash", "Owner")
	if err != nil {
		t.Fatalf("create organizer: %v", err)
	}
	t.Cleanup(func() { db.Exec(context.Background(), `DELETE FROM organizers WHERE id = $1`, ownerID) })

	events := store.NewEventStore(db)
	eventID, err := events.Create(ctx, store.Event{OwnerID: ownerID, Title: "Files", Slug: "files-" + suffix[:8], ModerationEnabled: true})
	if err != nil {
		t.Fatalf("create event: %v", err)
	}
	invitations := store.NewInvitationStore(db)
	if _, err := invitations.Create(ctx, eventID, store.NewInvitation{TokenHash: suffix, RecipientName: "Guest"}, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("create invitation: %v", err)
	}

	// Requests that fail before reaching the bucket.
	submissions := store.NewSubmissionStore(db)
	router := NewServer(Deps{Events: events, Invitations: invitations, Submissions: submissions, SiteURL: "http://localhost:3000"}).Routes()
	post := func(files []uploadedFile) int {
		body, _ := json.Marshal(submissionInput{SenderName: "Jo", Files: files})
		r := httptest.NewRequest(http.MethodPost, "/api/invitations/"+raw+"/submissions", strings.NewReader(string(body)))
		r.RemoteAddr = "192.0.2.77:1"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w.Code
	}
	key := func(name string) string { return "events/" + eventID + "/" + name }
	photo := func(name string) uploadedFile { return uploadedFile{key(name + ".jpg"), "image/jpeg"} }
	tooMany := []uploadedFile{photo("a"), photo("b"), photo("c"), photo("d"), photo("e"), photo("f")}
	for name, files := range map[string][]uploadedFile{
		"six files":     tooMany,
		"same file":     {photo("a"), photo("a")},
		"another event": {{"events/other/a.jpg", "image/jpeg"}},
		"wrong type":    {{key("a.exe"), "application/octet-stream"}},
	} {
		if code := post(files); code != http.StatusBadRequest {
			t.Errorf("%s: got %d, want 400", name, code)
		}
	}

	files := []store.Media{
		{StorageKey: key("first.jpg"), MimeType: "image/jpeg", FileSize: 1},
		{StorageKey: key("second.mp4"), MimeType: "video/mp4", FileSize: 2},
		{StorageKey: key("third.mov"), MimeType: "video/quicktime", FileSize: 3},
	}
	id, err := submissions.Create(ctx, store.Submission{EventID: eventID, SenderName: "Jo", Type: "photo", Status: "approved"}, files)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(context.Background(), `DELETE FROM jobs WHERE payload->>'media_id' IS NOT NULL AND NOT EXISTS (SELECT 1 FROM media m WHERE m.id::text = payload->>'media_id')`)
	})

	memories, err := submissions.ListApprovedBySlug(ctx, "files-"+suffix[:8])
	if err != nil || len(memories) != 1 {
		t.Fatalf("list: %v %v", memories, err)
	}
	var got []string
	for _, f := range memories[0].Files {
		got = append(got, f.StorageKey)
	}
	if len(got) != 3 || got[0] != key("first.jpg") || got[2] != key("third.mov") {
		t.Fatalf("files out of order: %v", got)
	}

	var videoJobs int
	err = db.QueryRow(ctx,
		`SELECT count(*) FROM jobs j JOIN media m ON m.id::text = j.payload->>'media_id' WHERE m.submission_id = $1 AND j.type = $2`,
		id, store.JobProcessVideo,
	).Scan(&videoJobs)
	if err != nil || videoJobs != 2 {
		t.Fatalf("video jobs = %d, %v, want 2", videoJobs, err)
	}

	// A note still lists an empty array, and someone else can't delete this.
	noteID, err := submissions.Create(ctx, store.Submission{EventID: eventID, SenderName: "Al", Type: "note", Message: "hi", Status: "approved"}, nil)
	if err != nil {
		t.Fatalf("create note: %v", err)
	}
	items, err := submissions.ListForOwner(ctx, eventID, ownerID)
	if err != nil || len(items) != 2 || items[0].ID != noteID || items[0].Files == nil || len(items[1].Files) != 3 {
		t.Fatalf("moderation list: %+v %v", items, err)
	}
	strangerID, err := organizers.Create(ctx, "stranger-"+suffix[:8]+"@test.example", "hash", "Other")
	if err != nil {
		t.Fatalf("create stranger: %v", err)
	}
	t.Cleanup(func() { db.Exec(context.Background(), `DELETE FROM organizers WHERE id = $1`, strangerID) })
	if _, err := submissions.DeleteForOwner(ctx, id, strangerID); err != store.ErrNotFound {
		t.Fatalf("stranger delete: %v", err)
	}

	if _, err := db.Exec(ctx, `UPDATE media SET thumbnail_key = 'thumb.jpg' WHERE storage_key = $1`, key("second.mp4")); err != nil {
		t.Fatal(err)
	}
	keys, err := submissions.DeleteForOwner(ctx, id, ownerID)
	if err != nil || len(keys) != 4 {
		t.Fatalf("delete should return 3 files and a thumbnail: %v %v", keys, err)
	}
}
