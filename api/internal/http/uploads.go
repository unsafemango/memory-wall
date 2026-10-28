package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type mediaRule struct {
	kind      string // "photo" or "video"
	extension string
	maxBytes  int64
}

var allowedMimeTypes = map[string]mediaRule{
	"image/jpeg":      {"photo", ".jpg", 15 << 20},
	"image/png":       {"photo", ".png", 15 << 20},
	"image/webp":      {"photo", ".webp", 15 << 20},
	"image/heic":      {"photo", ".heic", 15 << 20},
	"video/mp4":       {"video", ".mp4", 100 << 20},
	"video/quicktime": {"video", ".mov", 100 << 20},
}

// mediaKeyPrefix is the folder an event's uploads live under. Submissions
// may only reference keys inside their own event's folder.
func mediaKeyPrefix(eventID string) string {
	return "events/" + eventID + "/"
}

func (s *Server) createUploadURL(w http.ResponseWriter, r *http.Request) {
	invitation, ok := s.invitationFromRequest(w, r)
	if !ok {
		return
	}

	var input struct {
		MimeType string `json:"mime_type"`
		Size     int64  `json:"size"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	rule, ok := allowedMimeTypes[strings.ToLower(input.MimeType)]
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_request", "unsupported file type, use a JPEG, PNG, WebP or HEIC photo, or an MP4 or MOV video")
		return
	}
	if input.Size <= 0 || input.Size > rule.maxBytes {
		writeError(w, http.StatusBadRequest, "invalid_request", "file is too large")
		return
	}

	// The key is generated here, never taken from the filename, so nobody can
	// overwrite another upload or write outside the event's folder.
	key := mediaKeyPrefix(invitation.EventID) + uuid.NewString() + rule.extension

	url, err := s.storage.PresignPut(r.Context(), key, input.MimeType, input.Size, 5*time.Minute)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not create upload URL")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"upload_url":  url,
		"storage_key": key,
	})
}
