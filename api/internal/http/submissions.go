package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/unsafemango/memory-wall/internal/storage"
	"github.com/unsafemango/memory-wall/internal/store"
)

type submissionInput struct {
	SenderName string `json:"sender_name"`
	Message    string `json:"message"`
	// Set when the sender attached a photo or video they already uploaded
	// with a presigned URL.
	StorageKey string `json:"storage_key,omitempty"`
	MimeType   string `json:"mime_type,omitempty"`
}

func (s *Server) createSubmission(w http.ResponseWriter, r *http.Request) {
	invitation, ok := s.invitationFromRequest(w, r)
	if !ok {
		return
	}

	var input submissionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	input.SenderName = strings.TrimSpace(input.SenderName)
	input.Message = strings.TrimSpace(input.Message)
	hasMedia := input.StorageKey != ""
	if input.SenderName == "" || (input.Message == "" && !hasMedia) {
		writeError(w, http.StatusBadRequest, "invalid_request", "your name and a message, photo, or video are required")
		return
	}
	if len(input.SenderName) > 100 || len(input.Message) > 5000 {
		writeError(w, http.StatusBadRequest, "invalid_request", "name or message is too long")
		return
	}

	event, err := s.events.GetByID(r.Context(), invitation.EventID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not look up event")
		return
	}

	// With moderation on, nothing is public until the organizer approves it.
	status := "approved"
	if event.ModerationEnabled {
		status = "pending"
	}

	sub := store.Submission{
		EventID:      invitation.EventID,
		InvitationID: invitation.ID,
		SenderName:   input.SenderName,
		Type:         "note",
		Message:      input.Message,
		Status:       status,
	}

	if !hasMedia {
		s.saveSubmission(w, r, sub, nil)
		return
	}

	rule, ok := allowedMimeTypes[strings.ToLower(input.MimeType)]
	if !ok || !strings.HasPrefix(input.StorageKey, mediaKeyPrefix(invitation.EventID)) ||
		!strings.HasSuffix(input.StorageKey, rule.extension) {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid upload")
		return
	}

	// Don't take the browser's word that the upload finished, check the bucket.
	size, err := s.storage.Size(r.Context(), input.StorageKey)
	if errors.Is(err, storage.ErrNotFound) {
		writeError(w, http.StatusBadRequest, "invalid_request", "upload not found, try attaching the file again")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not check upload")
		return
	}
	if size > rule.maxBytes {
		writeError(w, http.StatusBadRequest, "invalid_request", "file is too large")
		return
	}

	sub.Type = rule.kind
	s.saveSubmission(w, r, sub, &store.Media{
		StorageKey: input.StorageKey,
		MimeType:   input.MimeType,
		FileSize:   size,
	})
}

func (s *Server) saveSubmission(w http.ResponseWriter, r *http.Request, sub store.Submission, media *store.Media) {
	id, err := s.submissions.Create(r.Context(), sub, media)
	if errors.Is(err, store.ErrInvitationUsed) {
		writeError(w, http.StatusGone, "invitation_used", "this invitation has already been used")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not save submission")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}
