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
		http.Error(w, "writing help isn't set up on this server", http.StatusServiceUnavailable)
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
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	input.SenderName = strings.TrimSpace(input.SenderName)
	input.Detail = strings.TrimSpace(input.Detail)
	if len(input.SenderName) > 100 || len(input.Detail) > 300 {
		http.Error(w, "name or detail is too long", http.StatusBadRequest)
		return
	}

	suggestions, err := s.ai.SuggestMessages(r.Context(), invitation.RecipientName, input.SenderName, input.Tone, input.Detail)
	if errors.Is(err, ai.ErrDeclined) {
		http.Error(w, "couldn't write suggestions for that, try different wording", http.StatusUnprocessableEntity)
		return
	}
	if err != nil {
		http.Error(w, "could not generate suggestions", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]string{"suggestions": suggestions})
}
