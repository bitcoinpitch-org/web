-- Create user activity tracking table for antispam monitoring
CREATE TABLE user_activities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    action_type VARCHAR(50) NOT NULL, -- 'pitch_create', 'pitch_edit', 'pitch_delete', 'vote', 'login', 'register'
    target_id UUID, -- ID of the target object (pitch_id for votes/edits, null for general actions)
    ip_address INET,
    user_agent TEXT,
    metadata JSONB, -- Additional context (e.g., content length, similarity score)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for efficient queries
CREATE INDEX idx_user_activities_user_id_created_at ON user_activities(user_id, created_at DESC);
CREATE INDEX idx_user_activities_action_type_created_at ON user_activities(action_type, created_at DESC);
CREATE INDEX idx_user_activities_ip_created_at ON user_activities(ip_address, created_at DESC);

-- Create temporary penalties table for progressive antispam enforcement
CREATE TABLE user_penalties (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    penalty_type VARCHAR(50) NOT NULL, -- 'rate_limit', 'cooldown', 'content_restriction'
    reason TEXT NOT NULL,
    multiplier DECIMAL(3,2) DEFAULT 1.0, -- Rate limit multiplier (e.g., 2.0 = double the normal limits)
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by UUID REFERENCES users(id), -- Admin who applied the penalty (null for automatic)
    is_active BOOLEAN DEFAULT true
);

-- Create indexes for penalty lookups
CREATE INDEX idx_user_penalties_user_id_active ON user_penalties(user_id, is_active, expires_at);
CREATE INDEX idx_user_penalties_expires_at ON user_penalties(expires_at) WHERE is_active = true;

-- Create content similarity tracking for duplicate detection
CREATE TABLE content_hashes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    content_hash VARCHAR(64) NOT NULL, -- SHA256 hash of normalized content
    original_content TEXT NOT NULL, -- Store original for admin review
    pitch_id UUID REFERENCES pitches(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for content similarity detection
CREATE INDEX idx_content_hashes_hash ON content_hashes(content_hash);
CREATE INDEX idx_content_hashes_user_id_created_at ON content_hashes(user_id, created_at DESC);

-- Add configuration for antispam settings
INSERT INTO config_settings (key, value, description, category, data_type) VALUES
    -- Daily limits
    ('antispam.pitch_create_cooldown_seconds', '60', 'Minimum seconds between pitch creations', 'antispam', 'integer'),
    ('antispam.pitch_edit_cooldown_seconds', '30', 'Minimum seconds between pitch edits', 'antispam', 'integer'),
    ('antispam.vote_cooldown_seconds', '2', 'Minimum seconds between votes', 'antispam', 'integer'),
    
    -- Content similarity
    ('antispam.content_similarity_threshold', '0.8', 'Similarity threshold for duplicate content detection (0.0-1.0)', 'antispam', 'number'),
    ('antispam.min_time_between_similar_hours', '24', 'Minimum hours between similar content from same user', 'antispam', 'integer'),
    
    -- Progressive penalties
    ('antispam.rapid_action_threshold', '5', 'Number of rapid actions before penalty', 'antispam', 'integer'),
    ('antispam.rapid_action_window_minutes', '5', 'Time window for rapid action detection (minutes)', 'antispam', 'integer'),
    ('antispam.penalty_multiplier', '2.0', 'Rate limit multiplier for penalties', 'antispam', 'number'),
    ('antispam.penalty_duration_hours', '24', 'Default penalty duration in hours', 'antispam', 'integer'),
    
    -- Content restrictions
    ('antispam.min_pitch_length', '3', 'Minimum pitch length to prevent spam', 'antispam', 'integer'),
    ('antispam.max_pitch_length', '2048', 'Maximum pitch length to prevent abuse', 'antispam', 'integer'),
    ('antispam.blacklisted_phrases', '[]', 'JSON array of blacklisted phrases/patterns', 'antispam', 'json'),
    
    -- IP-based limits
    ('antispam.max_accounts_per_ip', '5', 'Maximum user accounts per IP address', 'antispam', 'integer'),
    ('antispam.max_pitches_per_ip_per_hour', '20', 'Maximum pitches from single IP per hour', 'antispam', 'integer'); 