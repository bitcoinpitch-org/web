-- Drop antispam configuration settings
DELETE FROM config_settings WHERE category = 'antispam';

-- Drop content similarity tracking table
DROP TABLE IF EXISTS content_hashes;

-- Drop user penalties table
DROP TABLE IF EXISTS user_penalties;

-- Drop user activity tracking table
DROP TABLE IF EXISTS user_activities; 