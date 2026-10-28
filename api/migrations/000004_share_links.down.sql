DROP INDEX IF EXISTS events_share_token_idx;

ALTER TABLE events
    DROP COLUMN IF EXISTS share_enabled,
    DROP COLUMN IF EXISTS share_token;
