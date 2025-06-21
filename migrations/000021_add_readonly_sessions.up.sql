-- Add read_only column to sessions table
ALTER TABLE sessions ADD COLUMN read_only BOOLEAN NOT NULL DEFAULT FALSE;

-- Add comment explaining the column
COMMENT ON COLUMN sessions.read_only IS 'Indicates if this is a read-only session (e.g., from npub authentication)'; 