package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/unsafemango/memory-wall/internal/store"
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

	for i := range memories {
		if err := s.signMediaURLs(r.Context(), &memories[i]); err != nil {
			http.Error(w, "could not load memories", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"event": event, "memories": memories})
}

// signMediaURLs fills in temporary links for a memory's file and thumbnail.
// The bucket is private, so each link is signed and expires after an hour.
func (s *Server) signMediaURLs(ctx context.Context, m *store.Memory) error {
	var err error
	if m.StorageKey != "" {
		if m.MediaURL, err = s.storage.PresignGet(ctx, m.StorageKey, time.Hour); err != nil {
			return err
		}
	}
	if m.ThumbnailKey != "" {
		if m.ThumbnailURL, err = s.storage.PresignGet(ctx, m.ThumbnailKey, time.Hour); err != nil {
			return err
		}
	}
	return nil
}
