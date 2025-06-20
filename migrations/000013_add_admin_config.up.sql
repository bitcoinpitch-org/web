-- Add configuration settings table for admin-configurable values
CREATE TABLE config_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key VARCHAR(255) NOT NULL UNIQUE,
    value TEXT NOT NULL,
    description TEXT,
    category VARCHAR(100) NOT NULL DEFAULT 'general',
    data_type VARCHAR(50) NOT NULL DEFAULT 'string', -- string, integer, boolean, json
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    updated_by UUID REFERENCES users(id)
);

-- Create index for faster lookups
CREATE INDEX idx_config_settings_key ON config_settings(key);
CREATE INDEX idx_config_settings_category ON config_settings(category);

-- Insert default configuration values
INSERT INTO config_settings (key, value, description, category, data_type) VALUES
    -- Pitch length limits
    ('pitch.one_liner.min_length', '3', 'Minimum characters for one-liner pitches', 'pitch_limits', 'integer'),
    ('pitch.one_liner.max_length', '30', 'Maximum characters for one-liner pitches', 'pitch_limits', 'integer'),
    ('pitch.sms.max_length', '80', 'Maximum characters for SMS pitches', 'pitch_limits', 'integer'),
    ('pitch.tweet.max_length', '280', 'Maximum characters for tweet pitches', 'pitch_limits', 'integer'),
    ('pitch.elevator.max_length', '1024', 'Maximum characters for elevator pitches', 'pitch_limits', 'integer'),
    
    -- Rate limiting
    ('rate_limit.max_requests', '100', 'Maximum requests per time window', 'security', 'integer'),
    ('rate_limit.window_seconds', '60', 'Rate limit time window in seconds', 'security', 'integer'),
    
    -- User permissions
    ('users.allow_registration', 'true', 'Allow new user registration', 'users', 'boolean'),
    ('users.require_email_verification', 'true', 'Require email verification for new users', 'users', 'boolean'),
    ('users.max_pitches_per_day', '10', 'Maximum pitches a user can create per day', 'users', 'integer'),
    
    -- Content moderation
    ('moderation.auto_approve_pitches', 'true', 'Automatically approve new pitches', 'moderation', 'boolean'),
    ('moderation.min_score_for_visibility', '-5', 'Minimum score before pitch is hidden', 'moderation', 'integer'),
    
    -- Site settings
    ('site.maintenance_mode', 'false', 'Enable maintenance mode', 'site', 'boolean'),
    ('site.announcement_banner', '', 'Site-wide announcement banner text', 'site', 'string'),
    ('site.max_tags_per_pitch', '5', 'Maximum tags allowed per pitch', 'site', 'integer'),
    
    -- Translation settings
    ('i18n.default_language', 'en', 'Default site language', 'i18n', 'string'),
    ('i18n.enabled_languages', '["en", "cs"]', 'List of enabled languages', 'i18n', 'json');

-- Add audit log for configuration changes
CREATE TABLE config_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_key VARCHAR(255) NOT NULL,
    old_value TEXT,
    new_value TEXT,
    changed_by UUID NOT NULL REFERENCES users(id),
    changed_at TIMESTAMP DEFAULT NOW(),
    action VARCHAR(50) NOT NULL -- 'created', 'updated', 'deleted'
);

CREATE INDEX idx_config_audit_key ON config_audit_log(config_key);
CREATE INDEX idx_config_audit_changed_by ON config_audit_log(changed_by);
CREATE INDEX idx_config_audit_changed_at ON config_audit_log(changed_at); 