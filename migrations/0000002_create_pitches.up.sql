-- Create pitches table
CREATE TABLE IF NOT EXISTS pitches (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    content TEXT NOT NULL,
    language TEXT NOT NULL,
    main_category TEXT NOT NULL CHECK (main_category IN ('bitcoin', 'lightning', 'cashu')),
    length_category TEXT NOT NULL CHECK (length_category IN ('one-liner', 'sms', 'tweet', 'elevator')),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE,
    vote_count INTEGER DEFAULT 0,
    last_edit_at TIMESTAMP WITH TIME ZONE,
    posted_by UUID,
    author_type TEXT NOT NULL CHECK (author_type IN ('same', 'unknown', 'custom', 'twitter', 'nostr')),
    author_name TEXT,
    author_handle TEXT
); 