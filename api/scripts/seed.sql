-- Dev seed data: one event and one invitation.
-- Safe to run more than once.
-- The invitation link is http://localhost:3000/invite/dev-token-plaintext.
-- Only the SHA-256 hash of the token is stored, same as real invitations.
WITH new_event AS (
  INSERT INTO events (title, slug, moderation_enabled) VALUES ('Sarah''s Event', 'sarah', false)
  ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title
  RETURNING id
)
INSERT INTO invitations (event_id, token_hash, recipient_name)
SELECT id, encode(sha256('dev-token-plaintext'::bytea), 'hex'), 'Emily' FROM new_event
ON CONFLICT (token_hash) DO NOTHING;
