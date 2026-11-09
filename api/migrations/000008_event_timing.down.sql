ALTER TABLE events
    DROP COLUMN IF EXISTS hide_until_event,
    DROP COLUMN IF EXISTS event_at,
    DROP COLUMN IF EXISTS timezone,
    DROP COLUMN IF EXISTS event_time;
