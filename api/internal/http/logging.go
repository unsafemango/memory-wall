package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// logRequests writes one structured log line per request.
func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		// The route pattern (/api/invitations/{token}) instead of the raw path
		// keeps invitation and session tokens out of the logs.
		route := chi.RouteContext(r.Context()).RoutePattern()
		if route == "" {
			route = "unmatched"
		}

		level := slog.LevelInfo
		if ww.Status() >= 500 {
			level = slog.LevelError
		}
		s.logger.LogAttrs(r.Context(), level, "request",
			slog.String("request_id", middleware.GetReqID(r.Context())),
			slog.String("method", r.Method),
			slog.String("route", route),
			slog.Int("status", ww.Status()),
			slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		)
	})
}
