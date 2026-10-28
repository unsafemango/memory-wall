package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

func (s *Server) getMemories(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	event, err := s.events.GetBySlug(r.Context(), slug)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "event not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "could not load event", http.StatusInternalServerError)
		return
	}

	memories, err := s.submissions.ListApprovedBySlug(r.Context(), slug)
	if err != nil {
		http.Error(w, "could not load memories", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"event": event, "memories": memories})
}
