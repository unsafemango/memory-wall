-- Email confirmation and new-memory notifications.
ALTER TABLE organizers
    ADD COLUMN email_verified_at TIMESTAMPTZ,
    ADD COLUMN notify_new_memories BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN unsubscribe_token TEXT NOT NULL DEFAULT encode(gen_random_bytes(32), 'hex');

CREATE UNIQUE INDEX organizers_unsubscribe_token_idx ON organizers (unsubscribe_token);

-- Only the hash of each confirmation token is stored, like sessions.
CREATE TABLE email_confirmations (
    token_hash TEXT PRIMARY KEY,
    organizer_id UUID NOT NULL REFERENCES organizers(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX email_confirmations_organizer_idx ON email_confirmations (organizer_id);

-- Tracks what the organizer has already been told about, for batching.
-- Starting at now() means nobody gets emailed about older memories.
ALTER TABLE events
    ADD COLUMN last_notified_at TIMESTAMPTZ,
    ADD COLUMN notified_through TIMESTAMPTZ NOT NULL DEFAULT now();
