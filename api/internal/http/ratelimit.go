package http

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/time/rate"
)

// keyedLimiter keeps one token-bucket rate limiter per key (an IP address or
// an invitation token). It lives in memory, which is fine for one API
// instance; running several would need a shared store such as Redis or a
// Postgres table.
type keyedLimiter struct {
	mu       sync.Mutex
	limiters map[string]*limiterEntry
	every    time.Duration
	burst    int
}

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// newKeyedLimiter allows burst requests at once, then one per every.
func newKeyedLimiter(every time.Duration, burst int) *keyedLimiter {
	l := &keyedLimiter{limiters: map[string]*limiterEntry{}, every: every, burst: burst}
	go l.cleanup()
	return l
}

func (l *keyedLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	e, ok := l.limiters[key]
	if !ok {
		e = &limiterEntry{limiter: rate.NewLimiter(rate.Every(l.every), l.burst)}
		l.limiters[key] = e
	}
	e.lastSeen = time.Now()
	return e.limiter.Allow()
}

// cleanup drops limiters that haven't been used for a while, so the map
// doesn't grow forever.
func (l *keyedLimiter) cleanup() {
	for range time.Tick(time.Minute) {
		l.mu.Lock()
		for key, e := range l.limiters {
			if time.Since(e.lastSeen) > 10*time.Minute {
				delete(l.limiters, key)
			}
		}
		l.mu.Unlock()
	}
}

func tooManyRequests(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "60")
	writeError(w, http.StatusTooManyRequests, "rate_limited", "too many requests, try again in a minute")
}

// clientIP is the address the request came from. Behind a reverse proxy, the
// RealIP middleware (enabled with TRUST_PROXY_HEADERS) rewrites RemoteAddr
// from X-Forwarded-For first.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func limitByIP(l *keyedLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !l.allow(clientIP(r)) {
				tooManyRequests(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// limitByToken limits requests per invitation token in the URL, so one link
// can't be used to hammer an endpoint from many IPs.
func limitByToken(l *keyedLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !l.allow(chi.URLParam(r, "token")) {
				tooManyRequests(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
