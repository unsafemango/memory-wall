package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

func noteFrom(f fixture, status string) Submission {
	return Submission{
		EventID:      f.event.ID,
		InvitationID: f.inviteID,
		SenderName:   "Guest",
		Type:         "note",
		Message:      "Congratulations!",
		Status:       status,
	}
}

func TestPendingSubmissionOnlyAppearsAfterApproval(t *testing.T) {
	db := testDB(t)
	f := newFixture(t, db)
	ctx := context.Background()
	submissions := NewSubmissionStore(db)

	id, err := submissions.Create(ctx, noteFrom(f, "pending"), nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	public, err := submissions.ListApprovedBySlug(ctx, f.event.Slug)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(public) != 0 {
		t.Fatalf("pending submission is public: %+v", public)
	}

	if err := submissions.UpdateStatus(ctx, id, f.ownerID, "approved"); err != nil {
		t.Fatalf("approve: %v", err)
	}

	public, err = submissions.ListApprovedBySlug(ctx, f.event.Slug)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(public) != 1 || public[0].ID != id {
		t.Fatalf("approved submission should be public, got %+v", public)
	}
}

// The ownership rule: another organizer can't see, moderate, or delete
// submissions on an event they don't own, even with the right IDs.
func TestOtherOrganizerCannotModerate(t *testing.T) {
	db := testDB(t)
	f := newFixture(t, db)
	ctx := context.Background()
	submissions := NewSubmissionStore(db)

	id, err := submissions.Create(ctx, noteFrom(f, "pending"), nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := submissions.UpdateStatus(ctx, id, f.otherID, "approved"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("approve by other organizer: got %v, want ErrNotFound", err)
	}
	got, err := submissions.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != "pending" {
		t.Fatalf("status changed to %q by another organizer", got.Status)
	}

	items, err := submissions.ListForOwner(ctx, f.event.ID, f.otherID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("other organizer can list submissions: %+v", items)
	}

	if _, err := submissions.DeleteForOwner(ctx, id, f.otherID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete by other organizer: got %v, want ErrNotFound", err)
	}

	events := NewEventStore(db)
	if _, err := events.GetOwned(ctx, f.event.ID, f.otherID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other organizer can load the event: %v", err)
	}
	hijack := f.event
	hijack.OwnerID = f.otherID
	hijack.Title = "Hijacked"
	if err := events.Update(ctx, hijack); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other organizer can edit the event: %v", err)
	}
}

func TestInvitationIsSingleUse(t *testing.T) {
	db := testDB(t)
	f := newFixture(t, db)
	ctx := context.Background()
	submissions := NewSubmissionStore(db)

	if _, err := submissions.Create(ctx, noteFrom(f, "pending"), nil); err != nil {
		t.Fatalf("first submission: %v", err)
	}
	if _, err := submissions.Create(ctx, noteFrom(f, "pending"), nil); !errors.Is(err, ErrInvitationUsed) {
		t.Fatalf("second submission: got %v, want ErrInvitationUsed", err)
	}

	inv, err := NewInvitationStore(db).GetByTokenHash(ctx, sha256Hex(f.inviteRaw))
	if err != nil {
		t.Fatalf("look up invitation: %v", err)
	}
	if inv.UsedAt == nil || inv.Usable(time.Now()) {
		t.Fatalf("invitation should be marked used: %+v", inv)
	}
}

func TestExpiredInvitationCannotSubmit(t *testing.T) {
	db := testDB(t)
	f := newFixture(t, db)
	ctx := context.Background()

	f.inviteID = createInvitation(t, db, f.event.ID, f.inviteRaw+"-old", time.Now().Add(-time.Minute))
	if _, err := NewSubmissionStore(db).Create(ctx, noteFrom(f, "pending"), nil); !errors.Is(err, ErrInvitationUsed) {
		t.Fatalf("got %v, want ErrInvitationUsed", err)
	}
}

func TestDuplicateSlugIsRejected(t *testing.T) {
	db := testDB(t)
	f := newFixture(t, db)

	dup := Event{OwnerID: f.otherID, Title: "Copy", Slug: f.event.Slug}
	if _, err := NewEventStore(db).Create(context.Background(), dup); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("got %v, want ErrDuplicate", err)
	}
}
