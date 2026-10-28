package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/unsafemango/memory-wall/internal/store"
)

func (s *Server) listSubmissions(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventID")
	if _, err := s.events.GetOwned(r.Context(), eventID, organizerIDFrom(r)); err != nil {
		// 404 rather than 403 for someone else's event, so IDs can't be probed.
		writeError(w, http.StatusNotFound, "not_found", "event not found")
		return
	}

	items, err := s.submissions.ListForOwner(r.Context(), eventID, organizerIDFrom(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not load submissions")
		return
	}
	for i := range items {
		if err := s.signMediaURLs(r.Context(), &items[i].Memory); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "could not load submissions")
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (s *Server) approveSubmission(w http.ResponseWriter, r *http.Request) {
	s.setSubmissionStatus(w, r, "approved")
}

func (s *Server) rejectSubmission(w http.ResponseWriter, r *http.Request) {
	s.setSubmissionStatus(w, r, "rejected")
}

func (s *Server) setSubmissionStatus(w http.ResponseWriter, r *http.Request, status string) {
	err := s.submissions.UpdateStatus(r.Context(), chi.URLParam(r, "id"), organizerIDFrom(r), status)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "submission not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not update submission")
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) deleteSubmission(w http.ResponseWriter, r *http.Request) {
	keys, err := s.submissions.DeleteForOwner(r.Context(), chi.URLParam(r, "id"), organizerIDFrom(r))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "submission not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not delete submission")
		return
	}

	// The row is already gone, so a failed file delete only leaves an orphaned
	// object in the bucket. Log it rather than failing the request.
	for _, key := range keys {
		if err := s.storage.Delete(r.Context(), key); err != nil {
			s.logger.Error("delete media", "storage_key", key, "error", err)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}
