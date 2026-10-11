-- Dev seed data: one event and one invitation.
-- Safe to run more than once.
WITH new_event AS (
  INSERT INTO events (title, slug) VALUES ('Sarah''s Event', 'sarah')
  ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title
  RETURNING id
)
INSERT INTO invitations (event_id, token_hash, recipient_name)
SELECT id, 'dev-token-plaintext', 'Emily' FROM new_event
ON CONFLICT (token_hash) DO NOTHING;