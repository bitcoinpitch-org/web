-- Remove admin management fields from pitches table
DROP INDEX IF EXISTS idx_pitches_hidden;
ALTER TABLE pitches DROP COLUMN IF EXISTS hidden; 