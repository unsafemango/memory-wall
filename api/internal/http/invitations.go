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
		// Not a personal invitation, so it may be an event's share link.
		return s.shareLinkFromToken(w, r, token)
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

func (s *Server) shareLinkFromToken(w http.ResponseWriter, r *http.Request, token string) (store.Invitation, bool) {
	invitation, err := s.invitations.GetShareLink(r.Context(), token)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "not_found", "invitation not found")
		return invitation, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not look up invitation")
		return invitation, false
	}
	if !invitation.ShareEnabled {
		writeError(w, http.StatusGone, "share_link_off", "this link has been turned off")
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
		"shared":         invitation.Shared,
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
	id, err := s.invitations.Create(r.Context(), event.ID, hash, input.RecipientName, expiresAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not save invitation")
		return
	}

	// This response is the only place the raw token ever appears.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"id":         id,
		"invite_url": s.siteURL + "/invite/" + raw,
	})
}

// maxBulkInvitations caps one bulk request. Bigger lists can be sent in parts.
const maxBulkInvitations = 200

// createInvitations makes a personal link for each name in a list.
func (s *Server) createInvitations(w http.ResponseWriter, r *http.Request) {
	event, err := s.events.GetOwned(r.Context(), chi.URLParam(r, "eventID"), organizerIDFrom(r))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "event not found")
		return
	}

	var input struct {
		Names         []string `json:"names"`
		ExpiresInDays int      `json:"expires_in_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	// Trim, drop blank lines, and skip repeats of the same name.
	var names []string
	seen := map[string]bool{}
	for _, n := range input.Names {
		n = strings.TrimSpace(n)
		if n == "" || seen[strings.ToLower(n)] {
			continue
		}
		if len(n) > 100 {
			writeError(w, http.StatusBadRequest, "invalid_request", "each name can be up to 100 characters")
			return
		}
		seen[strings.ToLower(n)] = true
		names = append(names, n)
	}
	if len(names) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "add at least one name")
		return
	}
	if len(names) > maxBulkInvitations {
		writeError(w, http.StatusBadRequest, "invalid_request", "you can make up to 200 links at a time")
		return
	}
	if input.ExpiresInDays <= 0 {
		input.ExpiresInDays = 30
	}
	if input.ExpiresInDays > 365 {
		input.ExpiresInDays = 365
	}

	raws := make([]string, len(names))
	news := make([]store.NewInvitation, len(names))
	for i, name := range names {
		raw, hash, err := auth.NewToken()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "could not generate tokens")
			return
		}
		raws[i] = raw
		news[i] = store.NewInvitation{TokenHash: hash, RecipientName: name}
	}

	expiresAt := time.Now().AddDate(0, 0, input.ExpiresInDays)
	ids, err := s.invitations.CreateMany(r.Context(), event.ID, news, expiresAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not save invitations")
		return
	}

	type created struct {
		ID            string `json:"id"`
		RecipientName string `json:"recipient_name"`
		InviteURL     string `json:"invite_url"`
	}
	out := make([]created, len(names))
	for i := range names {
		out[i] = created{ID: ids[i], RecipientName: names[i], InviteURL: s.siteURL + "/invite/" + raws[i]}
	}

	// Like single invitations, this response is the only place the links appear.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(out)
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
