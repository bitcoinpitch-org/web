-- Add upvote_count and downvote_count columns to pitches
ALTER TABLE pitches ADD COLUMN IF NOT EXISTS upvote_count INTEGER DEFAULT 0;
ALTER TABLE pitches ADD COLUMN IF NOT EXISTS downvote_count INTEGER DEFAULT 0; 