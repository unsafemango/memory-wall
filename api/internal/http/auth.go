package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/unsafemango/memory-wall/internal/auth"
	"github.com/unsafemango/memory-wall/internal/email"
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

	s.sendConfirmation(r.Context(), organizerID, email, displayName)
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

// updateProfile changes the logged-in organizer's name and email. Changing
// the email needs their current password, since it's what they log in with.
func (s *Server) updateProfile(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email           string `json:"email"`
		DisplayName     string `json:"display_name"`
		CurrentPassword string `json:"current_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	email := strings.ToLower(strings.TrimSpace(input.Email))
	displayName := strings.TrimSpace(input.DisplayName)
	if _, err := mail.ParseAddress(email); err != nil || displayName == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "a name and a valid email are required")
		return
	}
	if len(displayName) > 100 {
		writeError(w, http.StatusBadRequest, "invalid_request", "the name is too long")
		return
	}

	organizer, err := s.organizers.GetByID(r.Context(), organizerIDFrom(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not load your account")
		return
	}
	if email != organizer.Email && !auth.VerifyPassword(input.CurrentPassword, organizer.PasswordHash) {
		writeError(w, http.StatusForbidden, "wrong_password", "your current password is incorrect")
		return
	}

	err = s.organizers.UpdateProfile(r.Context(), organizer.ID, email, displayName)
	if errors.Is(err, store.ErrDuplicate) {
		writeError(w, http.StatusConflict, "email_taken", "an account with that email already exists")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not update your account")
		return
	}
	if email != organizer.Email {
		s.sendConfirmation(r.Context(), organizer.ID, email, displayName)
	}
	s.me(w, r)
}

// changePassword replaces the logged-in organizer's password and signs them
// out on every other device.
func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	var input struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	if len(input.NewPassword) < 10 || len(input.NewPassword) > 200 {
		writeError(w, http.StatusBadRequest, "invalid_request", "the new password must be at least 10 characters")
		return
	}

	organizer, err := s.organizers.GetByID(r.Context(), organizerIDFrom(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not load your account")
		return
	}
	if !auth.VerifyPassword(input.CurrentPassword, organizer.PasswordHash) {
		writeError(w, http.StatusForbidden, "wrong_password", "your current password is incorrect")
		return
	}

	hash, err := auth.HashPassword(input.NewPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not change your password")
		return
	}
	if err := s.organizers.UpdatePassword(r.Context(), organizer.ID, hash); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not change your password")
		return
	}

	// requireOrganizer already checked this cookie, so it's present.
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	if err := s.sessions.DeleteOthers(r.Context(), organizer.ID, auth.HashToken(cookie.Value)); err != nil {
		s.logger.Error("end other sessions", "organizer_id", organizer.ID, "error", err)
	}
	w.WriteHeader(http.StatusNoContent)
}

const confirmationLifetime = 7 * 24 * time.Hour

// sendConfirmation emails a link that confirms an address. A failure is only
// logged, since the organizer can ask for another from their account page.
func (s *Server) sendConfirmation(ctx context.Context, organizerID, address, name string) bool {
	if !s.mailer.Enabled() {
		return false
	}
	raw, hash, err := auth.NewToken()
	if err == nil {
		err = s.organizers.CreateEmailConfirmation(ctx, hash, organizerID, address, time.Now().Add(confirmationLifetime))
	}
	if err != nil {
		s.logger.Error("create email confirmation", "organizer_id", organizerID, "error", err)
		return false
	}
	msg, err := email.Confirmation(address, name, s.siteURL+"/confirm-email/"+raw)
	if err == nil {
		err = s.mailer.Send(ctx, msg)
	}
	if err != nil {
		s.logger.Error("send confirmation email", "organizer_id", organizerID, "error", err)
		return false
	}
	return true
}

// resendConfirmation sends the logged-in organizer a new confirmation link.
func (s *Server) resendConfirmation(w http.ResponseWriter, r *http.Request) {
	organizer, err := s.organizers.GetByID(r.Context(), organizerIDFrom(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not load your account")
		return
	}
	if organizer.EmailVerified {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if !s.sendConfirmation(r.Context(), organizer.ID, organizer.Email, organizer.DisplayName) {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "we couldn't send the email, try again in a minute")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// confirmEmail uses a link from a confirmation email. It doesn't need a login,
// since people often open the email on another device.
func (s *Server) confirmEmail(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	err := s.organizers.ConfirmEmail(r.Context(), auth.HashToken(input.Token))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusGone, "link_expired", "this link has expired or was already used")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not confirm your email")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// setNotifications turns the logged-in organizer's new-memory emails on or off.
func (s *Server) setNotifications(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Enabled *bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Enabled == nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "send {\"enabled\": true} or false")
		return
	}
	if err := s.organizers.SetNotifications(r.Context(), organizerIDFrom(r), *input.Enabled); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not update your notifications")
		return
	}
	s.me(w, r)
}

// unsubscribe uses the link in a notification email. It takes no login, so
// turning off emails takes one click from any device.
func (s *Server) unsubscribe(w http.ResponseWriter, r *http.Request) {
	err := s.organizers.Unsubscribe(r.Context(), chi.URLParam(r, "token"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "this link isn't valid")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not update your notifications")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
