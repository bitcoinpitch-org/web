-- Migration: Add page_size preference to users table
ALTER TABLE users ADD COLUMN page_size INTEGER;

-- Add comment explaining the column
COMMENT ON COLUMN users.page_size IS 'User preferred page size for pitch lists. NULL means use system default.'; 