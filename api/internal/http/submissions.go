package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/unsafemango/memory-wall/internal/store"
)

type submissionInput struct {
	SenderName string `json:"sender_name"`
	Message    string `json:"message"`
}

func (s *Server) createSubmission(w http.ResponseWriter, r *http.Request) {
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

	var input submissionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	input.SenderName = strings.TrimSpace(input.SenderName)
	input.Message = strings.TrimSpace(input.Message)
	if input.SenderName == "" || input.Message == "" {
		http.Error(w, "your name and a message are required", http.StatusBadRequest)
		return
	}
	if len(input.SenderName) > 100 || len(input.Message) > 5000 {
		http.Error(w, "name or message is too long", http.StatusBadRequest)
		return
	}

	id, err := s.submissions.Create(r.Context(), store.Submission{
		EventID:      invitation.EventID,
		InvitationID: invitation.ID,
		SenderName:   input.SenderName,
		Type:         "note",
		Message:      input.Message,
		Status:       "approved",
	})
	if err != nil {
		http.Error(w, "could not save submission", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}
