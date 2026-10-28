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
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Lowercase so Sam@x.com and sam@x.com can't become two accounts.
	email := strings.ToLower(strings.TrimSpace(input.Email))
	displayName := strings.TrimSpace(input.DisplayName)
	if _, err := mail.ParseAddress(email); err != nil || displayName == "" || len(input.Password) < 10 {
		http.Error(w, "name, a valid email, and a password of at least 10 characters are required", http.StatusBadRequest)
		return
	}
	if len(displayName) > 100 || len(input.Password) > 200 {
		http.Error(w, "name or password is too long", http.StatusBadRequest)
		return
	}

	hash, err := auth.HashPassword(input.Password)
	if err != nil {
		http.Error(w, "could not create account", http.StatusInternalServerError)
		return
	}

	organizerID, err := s.organizers.Create(r.Context(), email, hash, displayName)
	if errors.Is(err, store.ErrDuplicate) {
		http.Error(w, "an account with that email already exists", http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, "could not create account", http.StatusInternalServerError)
		return
	}

	s.startSession(w, r, organizerID)
}

// startSession issues a new session cookie for organizerID.
func (s *Server) startSession(w http.ResponseWriter, r *http.Request, organizerID string) {
	raw, hash, err := auth.NewToken()
	if err != nil {
		http.Error(w, "could not start session", http.StatusInternalServerError)
		return
	}

	expires := time.Now().Add(sessionDuration)
	if err := s.sessions.Create(r.Context(), hash, organizerID, expires); err != nil {
		http.Error(w, "could not start session", http.StatusInternalServerError)
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
