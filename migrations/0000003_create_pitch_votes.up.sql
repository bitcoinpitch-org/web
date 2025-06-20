-- Create votes table
CREATE TABLE IF NOT EXISTS votes (
    id UUID PRIMARY KEY,
    pitch_id UUID NOT NULL REFERENCES pitches(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    vote_type TEXT NOT NULL CHECK (vote_type IN ('up', 'down')),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    UNIQUE(pitch_id, user_id)
); 