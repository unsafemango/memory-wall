package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/unsafemango/memory-wall/internal/store"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// Slugs that would clash with the web app's own pages, since /[slug] is a
// catch-all route.
var reservedSlugs = map[string]bool{
	"api": true, "login": true, "signup": true, "dashboard": true, "invite": true,
}

type eventInput struct {
	Title             string `json:"title"`
	Slug              string `json:"slug"`
	EventDate         string `json:"event_date"` // "2026-11-14", optional
	ClosingMessage    string `json:"closing_message"`
	ModerationEnabled bool   `json:"moderation_enabled"`
}

// decodeEvent reads and validates an event from the request body. On failure
// it writes the error response and returns false.
func decodeEvent(w http.ResponseWriter, r *http.Request) (eventInput, bool) {
	var input eventInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return input, false
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Slug = strings.TrimSpace(input.Slug)
	input.ClosingMessage = strings.TrimSpace(input.ClosingMessage)

	if input.Title == "" || len(input.Title) > 200 {
		http.Error(w, "a title of up to 200 characters is required", http.StatusBadRequest)
		return input, false
	}
	if !slugPattern.MatchString(input.Slug) || len(input.Slug) > 60 || reservedSlugs[input.Slug] {
		http.Error(w, "the link can only use lowercase letters, numbers, and dashes", http.StatusBadRequest)
		return input, false
	}
	if input.EventDate != "" {
		if _, err := time.Parse("2006-01-02", input.EventDate); err != nil {
			http.Error(w, "the date must look like 2026-11-14", http.StatusBadRequest)
			return input, false
		}
	}
	if len(input.ClosingMessage) > 5000 {
		http.Error(w, "the closing message is too long", http.StatusBadRequest)
		return input, false
	}
	return input, true
}

func (s *Server) createEvent(w http.ResponseWriter, r *http.Request) {
	input, ok := decodeEvent(w, r)
	if !ok {
		return
	}

	// The owner always comes from the session, never the request body.
	id, err := s.events.Create(r.Context(), store.Event{
		OwnerID:           organizerIDFrom(r),
		Title:             input.Title,
		Slug:              input.Slug,
		EventDate:         input.EventDate,
		ClosingMessage:    input.ClosingMessage,
		ModerationEnabled: input.ModerationEnabled,
	})
	if errors.Is(err, store.ErrDuplicate) {
		http.Error(w, "that link is already taken, try another", http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, "could not create event", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": id, "slug": input.Slug})
}

func (s *Server) listMyEvents(w http.ResponseWriter, r *http.Request) {
	events, err := s.events.ListByOwner(r.Context(), organizerIDFrom(r))
	if err != nil {
		http.Error(w, "could not load events", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

func (s *Server) getMyEvent(w http.ResponseWriter, r *http.Request) {
	event, err := s.events.GetOwned(r.Context(), chi.URLParam(r, "eventID"), organizerIDFrom(r))
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "event not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "could not load event", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

func (s *Server) updateEvent(w http.ResponseWriter, r *http.Request) {
	input, ok := decodeEvent(w, r)
	if !ok {
		return
	}

	err := s.events.Update(r.Context(), store.Event{
		ID:                chi.URLParam(r, "eventID"),
		OwnerID:           organizerIDFrom(r),
		Title:             input.Title,
		Slug:              input.Slug,
		EventDate:         input.EventDate,
		ClosingMessage:    input.ClosingMessage,
		ModerationEnabled: input.ModerationEnabled,
	})
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "event not found", http.StatusNotFound)
		return
	}
	if errors.Is(err, store.ErrDuplicate) {
		http.Error(w, "that link is already taken, try another", http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, "could not update event", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
