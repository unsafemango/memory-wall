UPDATE jobs j
SET payload = jsonb_build_object('submission_id', m.submission_id)
FROM media m
WHERE j.type = 'process_video'
  AND j.status IN ('pending', 'running')
  AND j.payload->>'media_id' = m.id::text;

-- The old schema shows one file per memory, so only the first one can stay.
DELETE FROM media WHERE position > 0;

DROP INDEX IF EXISTS media_submission_idx;

ALTER TABLE media DROP COLUMN position;
