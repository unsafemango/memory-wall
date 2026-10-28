package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/unsafemango/memory-wall/internal/live"
	"github.com/unsafemango/memory-wall/internal/store"
)

// streamPublicChanges tells an open event page when its approved memories
// change. It only says that something changed; the page then refetches
// /memories. Pending memories aren't announced.
func (s *Server) streamPublicChanges(w http.ResponseWriter, r *http.Request) {
	event, err := s.events.GetBySlug(r.Context(), chi.URLParam(r, "slug"))
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "not_found", "event not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not load event")
		return
	}
	s.streamChanges(w, r, event.ID, func(c live.Change) any {
		if !c.Public() {
			return nil
		}
		return map[string]string{"op": c.Op}
	})
}

// streamOrganizerChanges tells the organizer's manage page about every change
// to the event's memories, including new ones waiting for review.
func (s *Server) streamOrganizerChanges(w http.ResponseWriter, r *http.Request) {
	event, err := s.events.GetOwned(r.Context(), chi.URLParam(r, "eventID"), organizerIDFrom(r))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "event not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not load event")
		return
	}
	s.streamChanges(w, r, event.ID, func(c live.Change) any { return c })
}

// streamChanges holds the request open as a Server-Sent Events stream. For
// each change, describe returns what to send, or nil to skip it. The
// browser's EventSource reconnects by itself if the stream drops.
func (s *Server) streamChanges(w http.ResponseWriter, r *http.Request, eventID string, describe func(live.Change) any) {
	if s.hub == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "live updates aren't available")
		return
	}
	rc := http.NewResponseController(w)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// Tell proxies such as nginx not to buffer the stream.
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	// retry: how long the browser waits before reconnecting.
	fmt.Fprint(w, "retry: 3000\n: connected\n\n")
	if err := rc.Flush(); err != nil {
		return
	}

	changes, unsubscribe := s.hub.Subscribe(eventID)
	defer unsubscribe()

	// A comment line now and then keeps idle connections from being closed
	// by proxies and load balancers.
	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-s.streamsDone:
			return
		case <-heartbeat.C:
			fmt.Fprint(w, ": ping\n\n")
		case c := <-changes:
			msg := describe(c)
			if msg == nil {
				continue
			}
			data, err := json.Marshal(msg)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: change\ndata: %s\n\n", data)
		}
		if err := rc.Flush(); err != nil {
			return
		}
	}
}
