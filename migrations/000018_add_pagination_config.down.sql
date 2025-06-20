-- Remove pagination configuration settings
DELETE FROM config_settings WHERE key IN (
    'pagination.default_page_size',
    'pagination.page_size_options', 
    'pagination.max_page_size',
    'pagination.show_total_count',
    'pagination.show_page_info'
); 