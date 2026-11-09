package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/unsafemango/memory-wall/internal/ai"
)

func (s *Server) suggestMessage(w http.ResponseWriter, r *http.Request) {
	if s.ai == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "writing help isn't set up on this server")
		return
	}

	invitation, ok := s.invitationFromRequest(w, r)
	if !ok {
		return
	}

	var input struct {
		SenderName string `json:"sender_name"`
		Tone       string `json:"tone"`
		Detail     string `json:"detail"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	input.SenderName = strings.TrimSpace(input.SenderName)
	input.Detail = strings.TrimSpace(input.Detail)
	if len(input.SenderName) > 100 || len(input.Detail) > 300 {
		writeError(w, http.StatusBadRequest, "invalid_request", "name or detail is too long")
		return
	}

	event, err := s.events.GetByID(r.Context(), invitation.EventID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not look up event")
		return
	}

	// The invitation's name is the guest, so it's only a fallback for the sender.
	sender := input.SenderName
	if sender == "" {
		sender = invitation.RecipientName
	}

	suggestions, err := s.ai.SuggestMessages(r.Context(), ai.Request{
		EventTitle:  event.Title,
		Occasion:    event.Occasion,
		HonoreeName: event.HonoreeName,
		SenderName:  sender,
		Tone:        input.Tone,
		Detail:      input.Detail,
	})
	if errors.Is(err, ai.ErrDeclined) {
		writeError(w, http.StatusUnprocessableEntity, "declined", "couldn't write suggestions for that, try different wording")
		return
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, "upstream_error", "could not generate suggestions")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]string{"suggestions": suggestions})
}
