package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/unsafemango/memory-wall/internal/auth"
	"github.com/unsafemango/memory-wall/internal/store"
)

const (
	sessionCookie   = "session"
	sessionDuration = 7 * 24 * time.Hour
)

func (s *Server) signup(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	// Lowercase so Sam@x.com and sam@x.com can't become two accounts.
	email := strings.ToLower(strings.TrimSpace(input.Email))
	displayName := strings.TrimSpace(input.DisplayName)
	if _, err := mail.ParseAddress(email); err != nil || displayName == "" || len(input.Password) < 10 {
		writeError(w, http.StatusBadRequest, "invalid_request", "name, a valid email, and a password of at least 10 characters are required")
		return
	}
	if len(displayName) > 100 || len(input.Password) > 200 {
		writeError(w, http.StatusBadRequest, "invalid_request", "name or password is too long")
		return
	}

	hash, err := auth.HashPassword(input.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not create account")
		return
	}

	organizerID, err := s.organizers.Create(r.Context(), email, hash, displayName)
	if errors.Is(err, store.ErrDuplicate) {
		writeError(w, http.StatusConflict, "email_taken", "an account with that email already exists")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not create account")
		return
	}

	s.startSession(w, r, organizerID)
}

// startSession issues a new session cookie for organizerID.
func (s *Server) startSession(w http.ResponseWriter, r *http.Request, organizerID string) {
	raw, hash, err := auth.NewToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not start session")
		return
	}

	expires := time.Now().Add(sessionDuration)
	if err := s.sessions.Create(r.Context(), hash, organizerID, expires); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not start session")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:  sessionCookie,
		Value: raw,
		// HttpOnly keeps page JavaScript from reading the cookie.
		HttpOnly: true,
		Secure:   strings.HasPrefix(s.siteURL, "https://"),
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		Expires:  expires,
	})
	w.WriteHeader(http.StatusOK)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	email := strings.ToLower(strings.TrimSpace(input.Email))
	organizer, err := s.organizers.GetByEmail(r.Context(), email)
	// The same message for an unknown email and a wrong password, so nobody can
	// use this endpoint to check who has an account.
	if err != nil || !auth.VerifyPassword(input.Password, organizer.PasswordHash) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "invalid email or password")
		return
	}

	s.startSession(w, r, organizer.ID)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookie); err == nil {
		s.sessions.Delete(r.Context(), auth.HashToken(cookie.Value))
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1})
	w.WriteHeader(http.StatusOK)
}

// me returns the logged-in organizer, so the web app can tell whether to show
// the dashboard or send them to the login page.
func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	organizer, err := s.organizers.GetByID(r.Context(), organizerIDFrom(r))
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(organizer)
}
