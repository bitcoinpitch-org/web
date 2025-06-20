-- Drop trigger first
DROP TRIGGER IF EXISTS update_sessions_updated_at ON sessions;

-- Remove updated_at column
ALTER TABLE sessions DROP COLUMN IF EXISTS updated_at; 