package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/unsafemango/memory-wall/internal/auth"
	"github.com/unsafemango/memory-wall/internal/store"
)

// invitationFromRequest looks up the invitation for the {token} in the URL.
// It hashes the token first, since only hashes are stored. On failure it
// writes the error response and returns false.
func (s *Server) invitationFromRequest(w http.ResponseWriter, r *http.Request) (store.Invitation, bool) {
	token := chi.URLParam(r, "token")

	invitation, err := s.invitations.GetByTokenHash(r.Context(), auth.HashToken(token))
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "not_found", "invitation not found")
		return invitation, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not look up invitation")
		return invitation, false
	}
	// 410 Gone: the link was real, but can't be used any more.
	if invitation.UsedAt != nil {
		writeError(w, http.StatusGone, "invitation_used", "this invitation has already been used")
		return invitation, false
	}
	if !invitation.Usable(time.Now()) {
		writeError(w, http.StatusGone, "invitation_expired", "this invitation has expired")
		return invitation, false
	}
	return invitation, true
}

func (s *Server) getInvitation(w http.ResponseWriter, r *http.Request) {
	invitation, ok := s.invitationFromRequest(w, r)
	if !ok {
		return
	}

	// Guests see which event they're writing for, and whether their memory
	// will need the organizer's approval before it shows up.
	event, err := s.events.GetByID(r.Context(), invitation.EventID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not look up event")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"id":             invitation.ID,
		"recipient_name": invitation.RecipientName,
		"expires_at":     invitation.ExpiresAt,
		"event": map[string]any{
			"title":              event.Title,
			"slug":               event.Slug,
			"event_date":         event.EventDate,
			"moderation_enabled": event.ModerationEnabled,
		},
	})
}

func (s *Server) createInvitation(w http.ResponseWriter, r *http.Request) {
	// Same ownership rule as moderation: you can only invite people to your own events.
	event, err := s.events.GetOwned(r.Context(), chi.URLParam(r, "eventID"), organizerIDFrom(r))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "event not found")
		return
	}

	var input struct {
		RecipientName string `json:"recipient_name"`
		ExpiresInDays int    `json:"expires_in_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	input.RecipientName = strings.TrimSpace(input.RecipientName)
	if len(input.RecipientName) > 100 {
		writeError(w, http.StatusBadRequest, "invalid_request", "the name is too long")
		return
	}
	if input.ExpiresInDays <= 0 {
		input.ExpiresInDays = 30
	}
	if input.ExpiresInDays > 365 {
		input.ExpiresInDays = 365
	}

	raw, hash, err := auth.NewToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not generate token")
		return
	}

	expiresAt := time.Now().AddDate(0, 0, input.ExpiresInDays)
	if err := s.invitations.Create(r.Context(), event.ID, hash, input.RecipientName, expiresAt); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not save invitation")
		return
	}

	// This response is the only place the raw token ever appears.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"invite_url": s.siteURL + "/invite/" + raw,
	})
}

func (s *Server) listInvitations(w http.ResponseWriter, r *http.Request) {
	event, err := s.events.GetOwned(r.Context(), chi.URLParam(r, "eventID"), organizerIDFrom(r))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "event not found")
		return
	}

	invitations, err := s.invitations.ListForEvent(r.Context(), event.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not load invitations")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(invitations)
}
