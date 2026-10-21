package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

func (s *Server) getInvitation(w http.ResponseWriter, r *http.Request) {
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(invitation)
}
