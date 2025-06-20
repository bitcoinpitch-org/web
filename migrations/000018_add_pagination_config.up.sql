-- Add pagination configuration settings
INSERT INTO config_settings (key, value, description, category, data_type) VALUES
    ('pagination.default_page_size', '10', 'Default number of pitches per page', 'site', 'integer'),
    ('pagination.page_size_options', '["10", "25", "50", "100"]', 'Available page size options for users', 'site', 'json'),
    ('pagination.max_page_size', '100', 'Maximum allowed page size', 'site', 'integer'),
    ('pagination.show_total_count', 'true', 'Show total pitch count in pagination', 'site', 'boolean'),
    ('pagination.show_page_info', 'true', 'Show current page information', 'site', 'boolean'); 