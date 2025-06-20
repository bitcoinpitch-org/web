-- Create languages table for enhanced language support
CREATE TABLE languages (
    code VARCHAR(5) PRIMARY KEY,          -- ISO 639-1/639-3 code
    name_english VARCHAR(100) NOT NULL,   -- English name
    name_native VARCHAR(100) NOT NULL,    -- Native name
    flag_emoji VARCHAR(10),               -- Flag emoji
    usage_count INTEGER DEFAULT 0,        -- Number of pitches using this language
    is_major BOOLEAN DEFAULT false,       -- Whether it's a major language (top of list)
    display_order INTEGER DEFAULT 999,    -- Custom ordering
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create trigger to auto-update updated_at
CREATE TRIGGER update_languages_updated_at
    BEFORE UPDATE ON languages
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Insert major world languages and EU official languages
INSERT INTO languages (code, name_english, name_native, flag_emoji, is_major, display_order) VALUES
-- Tier 1: Major World Languages
('en', 'English', 'English', '🇬🇧', true, 1),
('es', 'Spanish', 'Español', '🇪🇸', true, 2),
('fr', 'French', 'Français', '🇫🇷', true, 3),
('de', 'German', 'Deutsch', '🇩🇪', true, 4),
('it', 'Italian', 'Italiano', '🇮🇹', true, 5),
('pt', 'Portuguese', 'Português', '🇵🇹', true, 6),
('ru', 'Russian', 'Русский', '🇷🇺', true, 7),
('zh', 'Chinese (Mandarin)', '中文', '🇨🇳', true, 8),
('ja', 'Japanese', '日本語', '🇯🇵', true, 9),
('ko', 'Korean', '한국어', '🇰🇷', true, 10),
('ar', 'Arabic', 'العربية', '🇸🇦', true, 11),
('hi', 'Hindi', 'हिन्दी', '🇮🇳', true, 12),
('nl', 'Dutch', 'Nederlands', '🇳🇱', true, 13),
('sv', 'Swedish', 'Svenska', '🇸🇪', true, 14),
('no', 'Norwegian', 'Norsk', '🇳🇴', true, 15),
('da', 'Danish', 'Dansk', '🇩🇰', true, 16),
('fi', 'Finnish', 'Suomi', '🇫🇮', true, 17),
('pl', 'Polish', 'Polski', '🇵🇱', true, 18),
('cs', 'Czech', 'Čeština', '🇨🇿', true, 19),
('hu', 'Hungarian', 'Magyar', '🇭🇺', true, 20),
('ro', 'Romanian', 'Română', '🇷🇴', true, 21),
('bg', 'Bulgarian', 'Български', '🇧🇬', true, 22),

-- Tier 2: Additional EU Official Languages
('hr', 'Croatian', 'Hrvatski', '🇭🇷', false, 100),
('et', 'Estonian', 'Eesti', '🇪🇪', false, 101),
('el', 'Greek', 'Ελληνικά', '🇬🇷', false, 102),
('ga', 'Irish', 'Gaeilge', '🇮🇪', false, 103),
('lv', 'Latvian', 'Latviešu', '🇱🇻', false, 104),
('lt', 'Lithuanian', 'Lietuvių', '🇱🇹', false, 105),
('mt', 'Maltese', 'Malti', '🇲🇹', false, 106),
('sk', 'Slovak', 'Slovenčina', '🇸🇰', false, 107),
('sl', 'Slovene', 'Slovenščina', '🇸🇮', false, 108),

-- Tier 3: Other Important Languages
('tr', 'Turkish', 'Türkçe', '🇹🇷', false, 200),
('uk', 'Ukrainian', 'Українська', '🇺🇦', false, 201),
('sr', 'Serbian', 'Српски', '🇷🇸', false, 202),
('sq', 'Albanian', 'Shqip', '🇦🇱', false, 203),
('mk', 'Macedonian', 'Македонски', '🇲🇰', false, 204),
('he', 'Hebrew', 'עברית', '🇮🇱', false, 205),
('fa', 'Persian/Farsi', 'فارسی', '🇮🇷', false, 206),
('id', 'Indonesian', 'Bahasa Indonesia', '🇮🇩', false, 207),
('vi', 'Vietnamese', 'Tiếng Việt', '🇻🇳', false, 208),
('th', 'Thai', 'ไทย', '🇹🇭', false, 209),
('ca', 'Catalan', 'Català', '🏴󠁥󠁳󠁣󠁴󠁿', false, 210),
('eu', 'Basque', 'Euskera', '🏴󠁥󠁳󠁰󠁶󠁿', false, 211),
('is', 'Icelandic', 'Íslenska', '🇮🇸', false, 212),
('nb', 'Norwegian Bokmål', 'Norsk Bokmål', '🇳🇴', false, 213),
('nn', 'Norwegian Nynorsk', 'Norsk Nynorsk', '🇳🇴', false, 214);

-- Update usage counts for existing pitches
UPDATE languages SET usage_count = (
    SELECT COUNT(*) FROM pitches WHERE language = languages.code
);

-- Create index for performance
CREATE INDEX idx_languages_usage_count ON languages(usage_count DESC);
CREATE INDEX idx_languages_is_major ON languages(is_major, display_order);
CREATE INDEX idx_languages_name_search ON languages(name_english, name_native); 