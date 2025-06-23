-- Add admin management fields to pitches table
ALTER TABLE pitches ADD COLUMN hidden BOOLEAN NOT NULL DEFAULT FALSE;

-- Create index for performance
CREATE INDEX idx_pitches_hidden ON pitches(hidden);

-- Add comment for clarity
COMMENT ON COLUMN pitches.hidden IS 'If true, pitch is hidden from public lists (admin/moderator only)'; 