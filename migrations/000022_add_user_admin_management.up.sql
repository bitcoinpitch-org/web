-- Add admin management fields to users table
ALTER TABLE users ADD COLUMN disabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN hidden BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN deleted_at TIMESTAMP NULL;

-- Create index for performance
CREATE INDEX idx_users_disabled ON users(disabled);
CREATE INDEX idx_users_hidden ON users(hidden);
CREATE INDEX idx_users_deleted_at ON users(deleted_at);

-- Add comments for clarity
COMMENT ON COLUMN users.disabled IS 'If true, user cannot login or perform actions';
COMMENT ON COLUMN users.hidden IS 'If true, user is hidden from public lists';
COMMENT ON COLUMN users.deleted_at IS 'Soft delete timestamp - NULL means not deleted'; 