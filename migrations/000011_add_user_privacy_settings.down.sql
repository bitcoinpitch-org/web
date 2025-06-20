-- Remove privacy settings columns from users table
ALTER TABLE users 
DROP COLUMN IF EXISTS show_auth_method,
DROP COLUMN IF EXISTS show_username,
DROP COLUMN IF EXISTS show_profile_info; 