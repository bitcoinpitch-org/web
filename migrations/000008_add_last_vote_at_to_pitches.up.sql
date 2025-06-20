-- Add last_vote_at column to pitches
ALTER TABLE pitches ADD COLUMN IF NOT EXISTS last_vote_at TIMESTAMP WITH TIME ZONE; 