DROP INDEX IF EXISTS events_owner_id_idx;

ALTER TABLE events
    DROP COLUMN IF EXISTS moderation_enabled,
    DROP COLUMN IF EXISTS closing_message,
    DROP COLUMN IF EXISTS owner_id;

DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS organizers;
