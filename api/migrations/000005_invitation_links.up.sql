-- The personal invitation link, encrypted with the server's
-- INVITATION_LINK_KEY, so organizers can see and copy it again. Lookups
-- still use token_hash. Invitations made before this column existed have no
-- copy and can't be shown again.
ALTER TABLE invitations ADD COLUMN token_encrypted TEXT;
