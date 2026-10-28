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
		http.Error(w, "invitation not found", http.StatusNotFound)
		return invitation, false
	}
	if err != nil {
		http.Error(w, "could not look up invitation", http.StatusInternalServerError)
		return invitation, false
	}
	// 410 Gone: the link was real, but can't be used any more.
	if invitation.UsedAt != nil {
		http.Error(w, "this invitation has already been used", http.StatusGone)
		return invitation, false
	}
	if !invitation.Usable(time.Now()) {
		http.Error(w, "this invitation has expired", http.StatusGone)
		return invitation, false
	}
	return invitation, true
}

func (s *Server) getInvitation(w http.ResponseWriter, r *http.Request) {
	invitation, ok := s.invitationFromRequest(w, r)
	if !ok {
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(invitation)
}

func (s *Server) createInvitation(w http.ResponseWriter, r *http.Request) {
	// Same ownership rule as moderation: you can only invite people to your own events.
	event, err := s.events.GetOwned(r.Context(), chi.URLParam(r, "eventID"), organizerIDFrom(r))
	if err != nil {
		http.Error(w, "event not found", http.StatusNotFound)
		return
	}

	var input struct {
		RecipientName string `json:"recipient_name"`
		ExpiresInDays int    `json:"expires_in_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	input.RecipientName = strings.TrimSpace(input.RecipientName)
	if len(input.RecipientName) > 100 {
		http.Error(w, "the name is too long", http.StatusBadRequest)
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
		http.Error(w, "could not generate token", http.StatusInternalServerError)
		return
	}

	expiresAt := time.Now().AddDate(0, 0, input.ExpiresInDays)
	if err := s.invitations.Create(r.Context(), event.ID, hash, input.RecipientName, expiresAt); err != nil {
		http.Error(w, "could not save invitation", http.StatusInternalServerError)
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
		http.Error(w, "event not found", http.StatusNotFound)
		return
	}

	invitations, err := s.invitations.ListForEvent(r.Context(), event.ID)
	if err != nil {
		http.Error(w, "could not load invitations", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(invitations)
}
