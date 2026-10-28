package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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
	trustProxy  bool
	logger      *slog.Logger
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
	// TrustProxy reads the client IP from X-Forwarded-For. Only turn it on
	// behind a proxy you control, since clients can set that header.
	TrustProxy bool
	Logger     *slog.Logger
}

func NewServer(d Deps) *Server {
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	return &Server{
		organizers:  d.Organizers,
		sessions:    d.Sessions,
		events:      d.Events,
		invitations: d.Invitations,
		submissions: d.Submissions,
		ai:          d.AI,
		storage:     d.Storage,
		siteURL:     d.SiteURL,
		trustProxy:  d.TrustProxy,
		logger:      d.Logger,
	}
}

// Routes builds the router with every endpoint registered.
func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	if s.trustProxy {
		r.Use(middleware.RealIP)
	}
	r.Use(middleware.RequestID, s.logRequests, middleware.Recoverer, s.corsMiddleware)

	// Login and signup are what attackers hammer to guess passwords.
	authLimit := limitByIP(newKeyedLimiter(12*time.Second, 5))
	// Guests: a generous per-IP limit, plus a tighter one per invitation link.
	guestIPLimit := limitByIP(newKeyedLimiter(2*time.Second, 20))
	guestTokenLimit := limitByToken(newKeyedLimiter(6*time.Second, 10))
	// Each writing suggestion is a paid model call.
	suggestLimit := limitByToken(newKeyedLimiter(20*time.Second, 5))

	r.Get("/api/healthz", healthz)
	r.Get("/api/events/{slug}/memories", s.getMemories)

	r.Route("/api/invitations/{token}", func(r chi.Router) {
		r.Use(guestIPLimit, guestTokenLimit)
		r.Get("/", s.getInvitation)
		r.Post("/submissions", s.createSubmission)
		r.Post("/uploads", s.createUploadURL)
		r.With(suggestLimit).Post("/suggest-message", s.suggestMessage)
	})

	r.Group(func(r chi.Router) {
		r.Use(s.sameOriginWrites)
		r.With(authLimit).Post("/api/auth/signup", s.signup)
		r.With(authLimit).Post("/api/auth/login", s.login)
		r.Post("/api/auth/logout", s.logout)

		r.Route("/api/organizer", func(r chi.Router) {
			r.Use(s.requireOrganizer)
			r.Get("/me", s.me)
			r.Get("/events", s.listMyEvents)
			r.Post("/events", s.createEvent)
			r.Get("/events/{eventID}", s.getMyEvent)
			r.Patch("/events/{eventID}", s.updateEvent)
			r.Get("/events/{eventID}/submissions", s.listSubmissions)
			r.Get("/events/{eventID}/invitations", s.listInvitations)
			r.Post("/events/{eventID}/invitations", s.createInvitation)
			r.Post("/events/{eventID}/share-link", s.setShareLink)
			r.Post("/events/{eventID}/share-link/reset", s.resetShareLink)
			r.Post("/submissions/{id}/approve", s.approveSubmission)
			r.Post("/submissions/{id}/reject", s.rejectSubmission)
			r.Delete("/submissions/{id}", s.deleteSubmission)
		})
	})

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
