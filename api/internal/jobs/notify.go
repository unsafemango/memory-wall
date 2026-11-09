package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/unsafemango/memory-wall/internal/email"
	"github.com/unsafemango/memory-wall/internal/store"
)

// previewLimit is how many memories to load for one email.
const previewLimit = 3

// MemoryNotifier emails organizers when new memories arrive on their events.
type MemoryNotifier struct {
	mailer      *email.Client
	submissions *store.SubmissionStore
	siteURL     string
}

func NewMemoryNotifier(mailer *email.Client, submissions *store.SubmissionStore, siteURL string) *MemoryNotifier {
	return &MemoryNotifier{mailer: mailer, submissions: submissions, siteURL: siteURL}
}

func (n *MemoryNotifier) Handle(ctx context.Context, payload []byte) error {
	var input struct {
		EventID string `json:"event_id"`
	}
	if err := json.Unmarshal(payload, &input); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	target, err := n.submissions.NotificationTarget(ctx, input.EventID)
	if errors.Is(err, store.ErrNotFound) {
		return nil // the event was deleted while the job waited
	}
	if err != nil {
		return fmt.Errorf("find organizer: %w", err)
	}

	unseen, err := n.submissions.ListUnseenMemories(ctx, input.EventID, previewLimit)
	if err != nil {
		return fmt.Errorf("list memories: %w", err)
	}
	if unseen.Total == 0 {
		return nil
	}

	if !target.CanNotify || !n.mailer.Enabled() {
		return n.submissions.MarkNotified(ctx, input.EventID, unseen.Through, false)
	}

	previews := make([]email.NewMemory, len(unseen.Previews))
	for i, m := range unseen.Previews {
		previews[i] = email.NewMemory{Sender: m.SenderName, Message: m.Message, Photos: m.Photos, Videos: m.Videos}
	}
	msg, err := email.NewMemories(
		target.OwnerEmail, target.EventTitle, previews, unseen.Total, unseen.NeedsReview,
		n.siteURL+"/dashboard/events/"+input.EventID,
		n.siteURL+"/unsubscribe/"+target.UnsubscribeToken,
	)
	if err != nil {
		return fmt.Errorf("build email: %w", err)
	}
	if err := n.mailer.Send(ctx, msg); err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	return n.submissions.MarkNotified(ctx, input.EventID, unseen.Through, true)
}
