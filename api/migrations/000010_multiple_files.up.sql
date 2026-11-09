-- A memory can have several files, shown in the order the guest added them.
ALTER TABLE media ADD COLUMN position INT NOT NULL DEFAULT 0;

CREATE INDEX media_submission_idx ON media (submission_id, position);

-- Video jobs now point at one file rather than at the whole memory. Every
-- memory has at most one file so far, so unfinished jobs can be rewritten.
UPDATE jobs j
SET payload = jsonb_build_object('media_id', m.id)
FROM media m
WHERE j.type = 'process_video'
  AND j.status IN ('pending', 'running')
  AND j.payload->>'submission_id' = m.submission_id::text;
