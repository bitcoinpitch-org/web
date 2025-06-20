-- Drop indexes
DROP INDEX IF EXISTS idx_languages_name_search;
DROP INDEX IF EXISTS idx_languages_is_major; 
DROP INDEX IF EXISTS idx_languages_usage_count;

-- Drop trigger
DROP TRIGGER IF EXISTS update_languages_updated_at ON languages;

-- Drop table
DROP TABLE IF EXISTS languages; 