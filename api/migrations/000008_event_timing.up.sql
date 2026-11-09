-- The exact moment an event happens, for the countdown and the reveal.
-- event_at is worked out from the date, optional time, and timezone. Events
-- that already have a date start at midnight UTC until they're next saved.
ALTER TABLE events
    ADD COLUMN event_time TEXT CHECK (event_time ~ '^[0-9]{2}:[0-9]{2}$'),
    ADD COLUMN timezone TEXT NOT NULL DEFAULT 'UTC',
    ADD COLUMN event_at TIMESTAMPTZ,
    ADD COLUMN hide_until_event BOOLEAN NOT NULL DEFAULT false;

UPDATE events SET event_at = event_date::timestamp AT TIME ZONE 'UTC' WHERE event_date IS NOT NULL;
