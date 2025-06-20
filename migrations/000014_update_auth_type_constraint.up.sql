-- Update auth_type constraint to include 'email'
ALTER TABLE users DROP CONSTRAINT users_auth_type_check;
ALTER TABLE users ADD CONSTRAINT users_auth_type_check CHECK (auth_type IN ('trezor', 'nostr', 'twitter', 'password', 'email')); 