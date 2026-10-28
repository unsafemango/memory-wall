package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/unsafemango/memory-wall/internal/ai"
	"github.com/unsafemango/memory-wall/internal/storage"
	"github.com/unsafemango/memory-wall/internal/store"
)

// Server holds everything the handlers need. Each handler is a method on it,
// so inside a handler you reach the database through s.invitations, s.submissions, etc.
type Server struct {
	organizers  *store.OrganizerStore
	sessions    *store.SessionStore
	events      *store.EventStore
	invitations *store.InvitationStore
	submissions *store.SubmissionStore
	ai          *ai.Client
	storage     *storage.Storage
	siteURL     string
}

// Deps lists what NewServer needs. main.go fills it in once at startup.
type Deps struct {
	Organizers  *store.OrganizerStore
	Sessions    *store.SessionStore
	Events      *store.EventStore
	Invitations *store.InvitationStore
	Submissions *store.SubmissionStore
	// AI is nil when no ANTHROPIC_API_KEY is configured; the writing helper
	// endpoint then answers 503 instead of failing every request.
	AI      *ai.Client
	Storage *storage.Storage
	// SiteURL is the web app's origin, e.g. http://localhost:3000. It's the
	// only origin the browser is allowed to call the API from.
	SiteURL string
}

func NewServer(d Deps) *Server {
	return &Server{
		organizers:  d.Organizers,
		sessions:    d.Sessions,
		events:      d.Events,
		invitations: d.Invitations,
		submissions: d.Submissions,
		ai:          d.AI,
		storage:     d.Storage,
		siteURL:     d.SiteURL,
	}
}

// Routes builds the router with every endpoint registered.
func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(s.corsMiddleware)

	r.Get("/api/healthz", healthz)
	r.Post("/api/auth/signup", s.signup)
	r.Get("/api/invitations/{token}", s.getInvitation)
	r.Post("/api/invitations/{token}/submissions", s.createSubmission)
	r.Post("/api/invitations/{token}/suggest-message", s.suggestMessage)
	r.Post("/api/invitations/{token}/uploads", s.createUploadURL)
	r.Get("/api/events/{slug}/memories", s.getMemories)

	return r
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// corsMiddleware lets the Next.js app at siteURL call this API from the browser.
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", s.siteURL)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		// Without this the browser drops the session cookie on requests from the web app.
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
