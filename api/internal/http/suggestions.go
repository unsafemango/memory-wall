package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/unsafemango/memory-wall/internal/ai"
)

func (s *Server) suggestMessage(w http.ResponseWriter, r *http.Request) {
	if s.ai == nil {
		http.Error(w, "writing help isn't set up on this server", http.StatusServiceUnavailable)
		return
	}

	token := chi.URLParam(r, "token")

	invitation, err := s.invitations.GetByToken(r.Context(), token)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "invitation not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "could not look up invitation", http.StatusInternalServerError)
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
