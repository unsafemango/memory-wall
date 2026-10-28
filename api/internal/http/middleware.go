package http

import (
	"context"
	"net/http"

	"github.com/unsafemango/memory-wall/internal/auth"
)

type ctxKey string

// A custom key type can't collide with context keys set by other packages.
const organizerIDKey ctxKey = "organizerID"

// requireOrganizer rejects requests without a valid session, and puts the
// logged-in organizer's ID on the request context for the handlers behind it.
func (s *Server) requireOrganizer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookie)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
			return
		}

		organizerID, err := s.sessions.OrganizerIDFor(r.Context(), auth.HashToken(cookie.Value))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
			return
		}

		ctx := context.WithValue(r.Context(), organizerIDKey, organizerID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// sameOriginWrites blocks cross-site request forgery: the browser sends the
// session cookie automatically, so state-changing requests must come from our
// own web app. Browsers always set Origin on cross-origin POSTs.
func (s *Server) sameOriginWrites(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			if origin := r.Header.Get("Origin"); origin != "" && origin != s.siteURL {
				writeError(w, http.StatusForbidden, "forbidden", "forbidden")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func organizerIDFrom(r *http.Request) string {
	id, _ := r.Context().Value(organizerIDKey).(string)
	return id
}
