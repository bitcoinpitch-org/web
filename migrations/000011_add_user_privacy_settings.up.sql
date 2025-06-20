-- Add privacy settings columns to users table
ALTER TABLE users 
ADD COLUMN show_auth_method BOOLEAN NOT NULL DEFAULT false,
ADD COLUMN show_username BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN show_profile_info BOOLEAN NOT NULL DEFAULT false; 