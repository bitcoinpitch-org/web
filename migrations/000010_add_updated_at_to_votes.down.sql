-- Remove updated_at column from votes table
ALTER TABLE votes DROP COLUMN IF EXISTS updated_at; 