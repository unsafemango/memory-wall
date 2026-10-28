// Package live tells open pages when an event's memories change. Postgres
// announces each change (see migration 000006), the Listener receives it,
// and the Hub passes it on to every page subscribed to that event.
package live

import (
	"sync"
)

// Change is one announced change to a memory.
type Change struct {
	EventID      string `json:"event_id"`
	SubmissionID string `json:"submission_id"`
	// Op is INSERT, UPDATE, DELETE, or MEDIA (a video's thumbnail arrived).
	// RESYNC means notifications may have been missed and pages should
	// reload everything.
	Op        string `json:"op"`
	Status    string `json:"status,omitempty"`
	OldStatus string `json:"old_status,omitempty"`
}

// Public reports whether the change affects what visitors to the event page
// can see. Pending memories stay private, so adding one isn't announced.
func (c Change) Public() bool {
	return c.Op == "RESYNC" || c.Status == "approved" || c.OldStatus == "approved"
}

// Hub fans changes out to subscribers, grouped by event.
type Hub struct {
	mu   sync.Mutex
	subs map[string]map[chan Change]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: map[string]map[chan Change]struct{}{}}
}

// Subscribe returns a channel of changes for one event, and a function to
// call when the subscriber goes away.
func (h *Hub) Subscribe(eventID string) (<-chan Change, func()) {
	// A small buffer absorbs bursts. If a page falls behind, extra changes
	// are dropped, which is fine because each one only means "refetch".
	ch := make(chan Change, 8)

	h.mu.Lock()
	if h.subs[eventID] == nil {
		h.subs[eventID] = map[chan Change]struct{}{}
	}
	h.subs[eventID][ch] = struct{}{}
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		delete(h.subs[eventID], ch)
		if len(h.subs[eventID]) == 0 {
			delete(h.subs, eventID)
		}
		h.mu.Unlock()
	}
}

// Publish sends a change to everyone watching its event. It never blocks.
func (h *Hub) Publish(c Change) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs[c.EventID] {
		select {
		case ch <- c:
		default:
		}
	}
}

// publishAll sends a change to every subscriber of every event.
func (h *Hub) publishAll(op string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for eventID, chans := range h.subs {
		for ch := range chans {
			select {
			case ch <- Change{EventID: eventID, Op: op}:
			default:
			}
		}
	}
}
