-- Remove upvote_count and downvote_count columns from pitches
ALTER TABLE pitches DROP COLUMN IF EXISTS upvote_count;
ALTER TABLE pitches DROP COLUMN IF EXISTS downvote_count; 