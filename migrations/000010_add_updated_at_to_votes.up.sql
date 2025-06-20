-- Add updated_at column to votes table
ALTER TABLE votes ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(); 