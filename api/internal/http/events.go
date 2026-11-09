package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/unsafemango/memory-wall/internal/ai"
	"github.com/unsafemango/memory-wall/internal/store"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// Slugs that would clash with the web app's own pages, since /[slug] is a
// catch-all route.
var reservedSlugs = map[string]bool{
	"api": true, "login": true, "signup": true, "dashboard": true, "invite": true,
	"confirm-email": true, "unsubscribe": true,
}

type eventInput struct {
	Title             string `json:"title"`
	Slug              string `json:"slug"`
	EventDate         string `json:"event_date"` // "2026-11-14", optional
	ClosingMessage    string `json:"closing_message"`
	ModerationEnabled bool   `json:"moderation_enabled"`
	Occasion          string `json:"occasion"`     // one of ai.Occasions, optional
	HonoreeName       string `json:"honoree_name"` // who the event is for, optional
	EventTime         string `json:"event_time"`   // "18:30", optional
	Timezone          string `json:"timezone"`     // IANA name, from the organizer's browser
	HideUntilEvent    bool   `json:"hide_until_event"`

	// eventAt is worked out from the date, time, and timezone.
	eventAt *time.Time
}

// eventMoment works out the exact moment of an event. With no time, it's the
// start of that day in the event's timezone.
func eventMoment(date, clock, timezone string) (*time.Time, error) {
	if date == "" {
		return nil, nil
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, err
	}
	if clock == "" {
		clock = "00:00"
	}
	at, err := time.ParseInLocation("2006-01-02 15:04", date+" "+clock, loc)
	if err != nil {
		return nil, err
	}
	return &at, nil
}

// decodeEvent reads and validates an event from the request body. On failure
// it writes the error response and returns false.
func decodeEvent(w http.ResponseWriter, r *http.Request) (eventInput, bool) {
	var input eventInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return input, false
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Slug = strings.TrimSpace(input.Slug)
	input.ClosingMessage = strings.TrimSpace(input.ClosingMessage)
	input.HonoreeName = strings.TrimSpace(input.HonoreeName)

	if input.Title == "" || len(input.Title) > 200 {
		writeError(w, http.StatusBadRequest, "invalid_request", "a title of up to 200 characters is required")
		return input, false
	}
	if !slugPattern.MatchString(input.Slug) || len(input.Slug) > 60 || reservedSlugs[input.Slug] {
		writeError(w, http.StatusBadRequest, "invalid_request", "the link can only use lowercase letters, numbers, and dashes")
		return input, false
	}
	if input.EventDate != "" {
		if _, err := time.Parse("2006-01-02", input.EventDate); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "the date must look like 2026-11-14")
			return input, false
		}
	}
	if len(input.ClosingMessage) > 5000 {
		writeError(w, http.StatusBadRequest, "invalid_request", "the closing message is too long")
		return input, false
	}
	if input.Occasion == "" {
		input.Occasion = ai.OccasionOther
	}
	if !ai.IsOccasion(input.Occasion) {
		writeError(w, http.StatusBadRequest, "invalid_request", "pick one of the listed occasions")
		return input, false
	}
	if len(input.HonoreeName) > 200 {
		writeError(w, http.StatusBadRequest, "invalid_request", "the name of who it's for is too long")
		return input, false
	}

	if input.Timezone == "" {
		input.Timezone = "UTC"
	}
	at, err := eventMoment(input.EventDate, input.EventTime, input.Timezone)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "the time must look like 18:30")
		return input, false
	}
	input.eventAt = at
	if input.EventDate == "" {
		input.EventTime = ""
	}
	// A memorial has nothing to count down to, so it never hides its memories.
	if input.Occasion == "memorial" {
		input.HideUntilEvent = false
	}
	if input.HideUntilEvent && input.EventDate == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "add a date to keep memories hidden until the event")
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
		Occasion:          input.Occasion,
		HonoreeName:       input.HonoreeName,
		EventTime:         input.EventTime,
		Timezone:          input.Timezone,
		EventAt:           input.eventAt,
		HideUntilEvent:    input.HideUntilEvent,
	})
	if errors.Is(err, store.ErrDuplicate) {
		writeError(w, http.StatusConflict, "slug_taken", "that link is already taken, try another")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not create event")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": id, "slug": input.Slug})
}

// shareInfo is the share link as the event's owner sees it. Only owner
// endpoints include it; public responses never show the token.
type shareInfo struct {
	ShareURL     string `json:"share_url"`
	ShareEnabled bool   `json:"share_enabled"`
}

func (s *Server) shareInfoFor(e store.Event) shareInfo {
	return shareInfo{ShareURL: s.siteURL + "/invite/" + e.ShareToken, ShareEnabled: e.ShareEnabled}
}

func (s *Server) listMyEvents(w http.ResponseWriter, r *http.Request) {
	events, err := s.events.ListByOwner(r.Context(), organizerIDFrom(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not load events")
		return
	}

	type listedEvent struct {
		store.EventSummary
		shareInfo
	}
	out := make([]listedEvent, len(events))
	for i, e := range events {
		out[i] = listedEvent{e, s.shareInfoFor(e.Event)}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (s *Server) getMyEvent(w http.ResponseWriter, r *http.Request) {
	event, err := s.events.GetOwned(r.Context(), chi.URLParam(r, "eventID"), organizerIDFrom(r))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "event not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not load event")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		store.Event
		shareInfo
	}{event, s.shareInfoFor(event)})
}

// setShareLink turns the event's share link on or off.
func (s *Server) setShareLink(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Enabled *bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Enabled == nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "send {\"enabled\": true} or false")
		return
	}

	eventID, ownerID := chi.URLParam(r, "eventID"), organizerIDFrom(r)
	err := s.events.SetShareEnabled(r.Context(), eventID, ownerID, *input.Enabled)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "event not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not update the share link")
		return
	}
	s.writeShareInfo(w, r, eventID, ownerID)
}

// resetShareLink replaces the event's share link, so the old one stops working.
func (s *Server) resetShareLink(w http.ResponseWriter, r *http.Request) {
	eventID, ownerID := chi.URLParam(r, "eventID"), organizerIDFrom(r)
	_, err := s.events.ResetShareToken(r.Context(), eventID, ownerID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "event not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not reset the share link")
		return
	}
	s.writeShareInfo(w, r, eventID, ownerID)
}

func (s *Server) writeShareInfo(w http.ResponseWriter, r *http.Request, eventID, ownerID string) {
	event, err := s.events.GetOwned(r.Context(), eventID, ownerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not load the share link")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.shareInfoFor(event))
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
		Occasion:          input.Occasion,
		HonoreeName:       input.HonoreeName,
		EventTime:         input.EventTime,
		Timezone:          input.Timezone,
		EventAt:           input.eventAt,
		HideUntilEvent:    input.HideUntilEvent,
	})
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "event not found")
		return
	}
	if errors.Is(err, store.ErrDuplicate) {
		writeError(w, http.StatusConflict, "slug_taken", "that link is already taken, try another")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not update event")
		return
	}
	w.WriteHeader(http.StatusOK)
}
