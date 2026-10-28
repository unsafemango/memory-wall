-- One reusable link per event that anyone can use to leave a memory.
-- Unlike personal invitations the raw token is stored, because the organizer
-- needs to see and copy the link again. Resetting it replaces the token.
ALTER TABLE events
    ADD COLUMN share_token TEXT,
    ADD COLUMN share_enabled BOOLEAN NOT NULL DEFAULT true;

UPDATE events SET share_token = encode(gen_random_bytes(32), 'hex');

ALTER TABLE events
    ALTER COLUMN share_token SET DEFAULT encode(gen_random_bytes(32), 'hex'),
    ALTER COLUMN share_token SET NOT NULL;

CREATE UNIQUE INDEX events_share_token_idx ON events (share_token);
