-- Add footer configuration settings
INSERT INTO config_settings (id, key, value, description, category, data_type, created_at, updated_at) VALUES
(gen_random_uuid(), 'footer_about_section', '{"enabled": true, "title": "About BitcoinPitch.org", "description": "A platform for collecting and sharing Bitcoin-related pitches. Find the perfect way to explain Bitcoin, Lightning, and Cashu to anyone."}', 'About section content in footer', 'footer', 'json', NOW(), NOW()),

(gen_random_uuid(), 'footer_categories_section', '{"enabled": true, "title": "Categories", "links": [{"name": "Bitcoin", "url": "/bitcoin"}, {"name": "Lightning", "url": "/lightning"}, {"name": "Cashu", "url": "/cashu"}]}', 'Category navigation links', 'footer', 'json', NOW(), NOW()),

(gen_random_uuid(), 'footer_resources_section', '{"enabled": true, "title": "Resources", "links": [{"name": "About", "url": "/about"}, {"name": "Privacy Policy", "url": "/privacy"}, {"name": "Terms of Service", "url": "/terms"}]}', 'Resource navigation links', 'footer', 'json', NOW(), NOW()),

(gen_random_uuid(), 'footer_connect_section', '{"enabled": true, "title": "Connect", "links": [{"name": "Twitter", "url": "https://twitter.com/bitcoinpitch", "external": true}, {"name": "GitHub", "url": "https://github.com/bitcoinpitch/bitcoinpitch.org", "external": true}, {"name": "Nostr", "url": "https://nostr.com/npub1bitcoinpitch", "external": true}]}', 'Social media and external links', 'footer', 'json', NOW(), NOW()),

(gen_random_uuid(), 'footer_bottom_text', 'Building a better Bitcoin narrative, one pitch at a time.', 'Footer bottom tagline text', 'footer', 'string', NOW(), NOW()),

(gen_random_uuid(), 'footer_copyright', '&copy; 2025 BitcoinPitch.org. All rights reserved.', 'Copyright text in footer', 'footer', 'string', NOW(), NOW()); 