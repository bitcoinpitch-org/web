-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create enum types
CREATE TYPE auth_type AS ENUM ('trezor', 'nostr', 'twitter', 'password');
CREATE TYPE main_category AS ENUM ('bitcoin', 'lightning', 'cashu');
CREATE TYPE length_category AS ENUM ('one-liner', 'sms', 'tweet', 'elevator');
CREATE TYPE author_type AS ENUM ('same', 'unknown', 'custom', 'twitter', 'nostr');

-- Create users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    auth_type auth_type NOT NULL,
    auth_id TEXT NOT NULL,
    username TEXT,
    display_name TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    show_auth_method BOOLEAN NOT NULL DEFAULT false,
    show_username BOOLEAN NOT NULL DEFAULT true,
    show_profile_info BOOLEAN NOT NULL DEFAULT false,
    UNIQUE(auth_type, auth_id)
);

-- Create sessions table
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(token)
);

-- Create tags table
CREATE TABLE tags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL UNIQUE,
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create pitches table
CREATE TABLE pitches (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    language TEXT NOT NULL,
    main_category main_category NOT NULL,
    length_category length_category NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    vote_count INTEGER DEFAULT 0,
    upvote_count INTEGER DEFAULT 0,
    downvote_count INTEGER DEFAULT 0,
    score INTEGER DEFAULT 0,
    last_vote_at TIMESTAMP WITH TIME ZONE,
    last_edit_at TIMESTAMP WITH TIME ZONE,
    posted_by UUID NOT NULL REFERENCES users(id),
    author_type author_type NOT NULL,
    author_name TEXT,
    author_handle TEXT,
    CONSTRAINT valid_author CHECK (
        (author_type = 'same') OR
        (author_type = 'unknown') OR
        (author_type = 'custom' AND author_name IS NOT NULL) OR
        (author_type = 'twitter' AND author_handle ~ '^@[A-Za-z0-9_]{1,15}$') OR
        (author_type = 'nostr' AND author_handle ~ '^npub1[a-zA-Z0-9]{58}$')
    )
);

-- Create pitch_tags table
CREATE TABLE pitch_tags (
    pitch_id UUID NOT NULL REFERENCES pitches(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (pitch_id, tag_id)
);

-- Create votes table
CREATE TABLE votes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pitch_id UUID NOT NULL REFERENCES pitches(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    vote_type TEXT NOT NULL CHECK (vote_type IN ('up', 'down')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(pitch_id, user_id)
);

-- Create indexes
CREATE INDEX idx_pitches_main_category ON pitches(main_category);
CREATE INDEX idx_pitches_length_category ON pitches(length_category);
CREATE INDEX idx_pitches_language ON pitches(language);
CREATE INDEX idx_pitches_score ON pitches(score DESC);
CREATE INDEX idx_pitches_created_at ON pitches(created_at DESC);
CREATE INDEX idx_pitches_user_id ON pitches(user_id);
CREATE INDEX idx_votes_pitch_id ON votes(pitch_id);
CREATE INDEX idx_votes_user_id ON votes(user_id);
CREATE INDEX idx_tags_name ON tags(name);
CREATE INDEX idx_pitch_tags_pitch_id ON pitch_tags(pitch_id);
CREATE INDEX idx_pitch_tags_tag_id ON pitch_tags(tag_id);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_pitches_updated_at
    BEFORE UPDATE ON pitches
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create function to update tag usage count
CREATE OR REPLACE FUNCTION update_tag_usage_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE tags SET usage_count = usage_count + 1 WHERE id = NEW.tag_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE tags SET usage_count = usage_count - 1 WHERE id = OLD.tag_id;
    END IF;
    RETURN NULL;
END;
$$ language 'plpgsql';

-- Create trigger for tag usage count
CREATE TRIGGER update_tag_usage_count
    AFTER INSERT OR DELETE ON pitch_tags
    FOR EACH ROW
    EXECUTE FUNCTION update_tag_usage_count(); 