-- Drop indexes first
DROP INDEX IF EXISTS idx_pitch_tags_tag_id;
DROP INDEX IF EXISTS idx_pitch_tags_pitch_id;
DROP INDEX IF EXISTS idx_tags_usage_count;
DROP INDEX IF EXISTS idx_tags_name;
 
-- Drop tables
DROP TABLE IF EXISTS pitch_tags;
DROP TABLE IF EXISTS tags; 