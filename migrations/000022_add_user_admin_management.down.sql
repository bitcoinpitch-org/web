-- Remove admin management fields from users table
DROP INDEX IF EXISTS idx_users_deleted_at;
DROP INDEX IF EXISTS idx_users_hidden;
DROP INDEX IF EXISTS idx_users_disabled;

ALTER TABLE users DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE users DROP COLUMN IF EXISTS hidden;
ALTER TABLE users DROP COLUMN IF EXISTS disabled; 