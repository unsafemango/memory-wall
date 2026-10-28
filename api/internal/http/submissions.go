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
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	input.SenderName = strings.TrimSpace(input.SenderName)
	input.Message = strings.TrimSpace(input.Message)
	hasMedia := input.StorageKey != ""
	if input.SenderName == "" || (input.Message == "" && !hasMedia) {
		http.Error(w, "your name and a message, photo, or video are required", http.StatusBadRequest)
		return
	}
	if len(input.SenderName) > 100 || len(input.Message) > 5000 {
		http.Error(w, "name or message is too long", http.StatusBadRequest)
		return
	}

	event, err := s.events.GetByID(r.Context(), invitation.EventID)
	if err != nil {
		http.Error(w, "could not look up event", http.StatusInternalServerError)
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
		id, err := s.submissions.Create(r.Context(), sub)
		if err != nil {
			http.Error(w, "could not save submission", http.StatusInternalServerError)
			return
		}
		writeCreated(w, id)
		return
	}

	rule, ok := allowedMimeTypes[strings.ToLower(input.MimeType)]
	if !ok || !strings.HasPrefix(input.StorageKey, mediaKeyPrefix(invitation.EventID)) ||
		!strings.HasSuffix(input.StorageKey, rule.extension) {
		http.Error(w, "invalid upload", http.StatusBadRequest)
		return
	}

	// Don't take the browser's word that the upload finished, check the bucket.
	size, err := s.storage.Size(r.Context(), input.StorageKey)
	if errors.Is(err, storage.ErrNotFound) {
		http.Error(w, "upload not found, try attaching the file again", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "could not check upload", http.StatusInternalServerError)
		return
	}
	if size > rule.maxBytes {
		http.Error(w, "file is too large", http.StatusBadRequest)
		return
	}

	sub.Type = rule.kind
	id, err := s.submissions.CreateWithMedia(r.Context(), sub, store.Media{
		StorageKey: input.StorageKey,
		MimeType:   input.MimeType,
		FileSize:   size,
	})
	if err != nil {
		http.Error(w, "could not save submission", http.StatusInternalServerError)
		return
	}
	writeCreated(w, id)
}

func writeCreated(w http.ResponseWriter, id string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}
