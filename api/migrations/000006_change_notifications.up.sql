-- Announce every change to a memory on the memory_wall_changes channel, so
-- the API can tell open pages to refresh. Notifications are sent when the
-- transaction commits, so listeners never see uncommitted rows.
CREATE FUNCTION notify_submission_change() RETURNS trigger AS $$
DECLARE
    row_event_id UUID;
    row_id UUID;
    new_status TEXT;
    old_status TEXT;
BEGIN
    IF TG_OP = 'DELETE' THEN
        row_event_id := OLD.event_id;
        row_id := OLD.id;
        old_status := OLD.status;
    ELSE
        row_event_id := NEW.event_id;
        row_id := NEW.id;
        new_status := NEW.status;
        IF TG_OP = 'UPDATE' THEN
            old_status := OLD.status;
        END IF;
    END IF;

    PERFORM pg_notify('memory_wall_changes', json_build_object(
        'event_id', row_event_id,
        'submission_id', row_id,
        'op', TG_OP,
        'status', new_status,
        'old_status', old_status
    )::text);
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER submissions_notify
    AFTER INSERT OR UPDATE OR DELETE ON submissions
    FOR EACH ROW EXECUTE FUNCTION notify_submission_change();

-- A video's thumbnail arrives later from the background job. Announce it too,
-- so a video card picks up its poster without a reload.
CREATE FUNCTION notify_media_change() RETURNS trigger AS $$
DECLARE
    sub RECORD;
BEGIN
    SELECT id, event_id, status INTO sub FROM submissions WHERE id = NEW.submission_id;
    IF FOUND THEN
        PERFORM pg_notify('memory_wall_changes', json_build_object(
            'event_id', sub.event_id,
            'submission_id', sub.id,
            'op', 'MEDIA',
            'status', sub.status,
            'old_status', sub.status
        )::text);
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER media_notify
    AFTER UPDATE ON media
    FOR EACH ROW EXECUTE FUNCTION notify_media_change();
