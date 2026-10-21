package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/unsafemango/memory-wall/internal/store"
)

// Server holds everything the handlers need. Each handler is a method on it,
// so inside a handler you reach the database through s.invitations, s.submissions, etc.
type Server struct {
	invitations *store.InvitationStore
	submissions *store.SubmissionStore
}

func NewServer(invitations *store.InvitationStore, submissions *store.SubmissionStore) *Server {
	return &Server{
		invitations: invitations,
		submissions: submissions,
	}
}

// Routes builds the router with every endpoint registered.
func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(corsMiddleware)

	r.Get("/api/healthz", healthz)
	r.Get("/api/invitations/{token}", s.getInvitation)

	return r
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// corsMiddleware lets the Next.js app on localhost:3000 call this API from the browser.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
