package http

import (
	"encoding/json"
	"errors"
	"net/http"
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
