-- Create tags table
CREATE TABLE IF NOT EXISTS tags (
    id UUID PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    usage_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Create pitch_tags table for many-to-many relationship
CREATE TABLE IF NOT EXISTS pitch_tags (
    id UUID PRIMARY KEY,
    pitch_id UUID NOT NULL REFERENCES pitches(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    UNIQUE(pitch_id, tag_id)
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_tags_name ON tags(name);
CREATE INDEX IF NOT EXISTS idx_tags_usage_count ON tags(usage_count);
CREATE INDEX IF NOT EXISTS idx_pitch_tags_pitch_id ON pitch_tags(pitch_id);
CREATE INDEX IF NOT EXISTS idx_pitch_tags_tag_id ON pitch_tags(tag_id); 