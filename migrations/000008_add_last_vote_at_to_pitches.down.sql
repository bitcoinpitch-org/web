-- Remove last_vote_at column from pitches
ALTER TABLE pitches DROP COLUMN IF EXISTS last_vote_at; 