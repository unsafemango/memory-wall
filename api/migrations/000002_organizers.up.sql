CREATE TABLE organizers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    token_hash TEXT PRIMARY KEY,
    organizer_id UUID NOT NULL REFERENCES organizers(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE events
    ADD COLUMN owner_id UUID REFERENCES organizers(id) ON DELETE CASCADE,
    ADD COLUMN closing_message TEXT,
    ADD COLUMN moderation_enabled BOOLEAN NOT NULL DEFAULT true;

CREATE INDEX events_owner_id_idx ON events(owner_id);
