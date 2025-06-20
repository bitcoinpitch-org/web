-- Add test user for development voting
INSERT INTO users (id, auth_type, auth_id, username, display_name, created_at, updated_at)
VALUES (
  '00000000-0000-0000-0000-000000000001',
  'password',
  'dev-user',
  'devuser',
  'Development User',
  NOW(),
  NOW()
) ON CONFLICT (id) DO NOTHING; 