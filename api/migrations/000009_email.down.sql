ALTER TABLE events
    DROP COLUMN IF EXISTS notified_through,
    DROP COLUMN IF EXISTS last_notified_at;

DROP TABLE IF EXISTS email_confirmations;

DROP INDEX IF EXISTS organizers_unsubscribe_token_idx;

ALTER TABLE organizers
    DROP COLUMN IF EXISTS unsubscribe_token,
    DROP COLUMN IF EXISTS notify_new_memories,
    DROP COLUMN IF EXISTS email_verified_at;
