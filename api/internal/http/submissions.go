package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/unsafemango/memory-wall/internal/storage"
	"github.com/unsafemango/memory-wall/internal/store"
)

// maxFilesPerSubmission is how many photos and videos one memory can hold.
const maxFilesPerSubmission = 5

type uploadedFile struct {
	StorageKey string `json:"storage_key"`
	MimeType   string `json:"mime_type"`
}

type submissionInput struct {
	SenderName string `json:"sender_name"`
	Message    string `json:"message"`
	// Files the sender already uploaded with presigned URLs.
	Files []uploadedFile `json:"files,omitempty"`
	// One file sent the old way, by a page opened before multiple files existed.
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
	if input.StorageKey != "" {
		input.Files = append(input.Files, uploadedFile{input.StorageKey, input.MimeType})
	}
	if input.SenderName == "" || (input.Message == "" && len(input.Files) == 0) {
		writeError(w, http.StatusBadRequest, "invalid_request", "your name and a message, photo, or video are required")
		return
	}
	if len(input.SenderName) > 100 || len(input.Message) > 5000 {
		writeError(w, http.StatusBadRequest, "invalid_request", "name or message is too long")
		return
	}
	if len(input.Files) > maxFilesPerSubmission {
		writeError(w, http.StatusBadRequest, "invalid_request", "a memory can have up to 5 photos and videos")
		return
	}

	// Check every file's details before asking the bucket about any of them.
	rules := make([]mediaRule, len(input.Files))
	seen := make(map[string]bool, len(input.Files))
	for i, file := range input.Files {
		rule, ok := allowedMimeTypes[strings.ToLower(file.MimeType)]
		if !ok || seen[file.StorageKey] ||
			!strings.HasPrefix(file.StorageKey, mediaKeyPrefix(invitation.EventID)) ||
			!strings.HasSuffix(file.StorageKey, rule.extension) {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid upload")
			return
		}
		seen[file.StorageKey] = true
		rules[i] = rule
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

	files := make([]store.Media, len(input.Files))
	for i, file := range input.Files {
		// Don't take the browser's word that the upload finished, check the bucket.
		size, err := s.storage.Size(r.Context(), file.StorageKey)
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusBadRequest, "invalid_request", "upload not found, try attaching the file again")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "could not check upload")
			return
		}
		if size > rules[i].maxBytes {
			writeError(w, http.StatusBadRequest, "invalid_request", "file is too large")
			return
		}
		files[i] = store.Media{StorageKey: file.StorageKey, MimeType: file.MimeType, FileSize: size}
	}
	if len(rules) > 0 {
		sub.Type = rules[0].kind
	}

	s.saveSubmission(w, r, sub, files)
}

func (s *Server) saveSubmission(w http.ResponseWriter, r *http.Request, sub store.Submission, files []store.Media) {
	id, err := s.submissions.Create(r.Context(), sub, files)
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
