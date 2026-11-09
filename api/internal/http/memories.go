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
		writeError(w, http.StatusNotFound, "not_found", "event not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not load event")
		return
	}

	// Hidden until the event: the memories and closing message aren't sent at
	// all, so nobody can find them in the page's data.
	if event.Hidden(time.Now()) {
		event.ClosingMessage = ""
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"event": event, "memories": []store.Memory{}, "locked": true})
		return
	}

	memories, err := s.submissions.ListApprovedBySlug(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not load memories")
		return
	}

	for i := range memories {
		if err := s.signMediaURLs(r.Context(), &memories[i]); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "could not load memories")
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"event": event, "memories": memories})
}

// signMediaURLs fills in temporary links for a memory's files and thumbnails.
// The bucket is private, so each link is signed and expires after an hour.
func (s *Server) signMediaURLs(ctx context.Context, m *store.Memory) error {
	for i := range m.Files {
		file := &m.Files[i]
		var err error
		if file.URL, err = s.storage.PresignGet(ctx, file.StorageKey, time.Hour); err != nil {
			return err
		}
		if file.ThumbnailKey != "" {
			if file.ThumbnailURL, err = s.storage.PresignGet(ctx, file.ThumbnailKey, time.Hour); err != nil {
				return err
			}
		}
	}
	return nil
}
