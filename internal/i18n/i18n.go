package i18n

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"
)

// TranslationMeta contains metadata about a language
type TranslationMeta struct {
	Name       string `json:"name"`
	NativeName string `json:"nativeName"`
	Code       string `json:"code"`
	Flag       string `json:"flag"`
}

// Translation contains all translations for a language
type Translation struct {
	Meta     TranslationMeta        `json:"meta"`
	UI       map[string]interface{} `json:"ui"`
	Errors   map[string]string      `json:"errors"`
	Register map[string]interface{} `json:"register"`
	Verify   map[string]interface{} `json:"verify"`
	// Store the raw JSON data to handle any additional top-level keys
	Raw map[string]interface{} `json:"-"`
}

// Manager handles translation loading and retrieval
type Manager struct {
	translations map[string]*Translation
	defaultLang  string
	mu           sync.RWMutex
}

// NewManager creates a new translation manager
func NewManager(defaultLang string) *Manager {
	return &Manager{
		translations: make(map[string]*Translation),
		defaultLang:  defaultLang,
	}
}

// LoadTranslations loads all translation files from the i18n directory
func (m *Manager) LoadTranslations(i18nDir string) error {
	files, err := ioutil.ReadDir(i18nDir)
	if err != nil {
		return fmt.Errorf("failed to read i18n directory: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		langCode := strings.TrimSuffix(file.Name(), ".json")
		filePath := filepath.Join(i18nDir, file.Name())

		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read translation file %s: %w", file.Name(), err)
		}

		var translation Translation
		if err := json.Unmarshal(data, &translation); err != nil {
			return fmt.Errorf("failed to parse translation file %s: %w", file.Name(), err)
		}

		// Also parse raw JSON data for flexible key access
		var rawData map[string]interface{}
		if err := json.Unmarshal(data, &rawData); err != nil {
			return fmt.Errorf("failed to parse raw JSON for %s: %w", file.Name(), err)
		}
		translation.Raw = rawData

		m.translations[langCode] = &translation
		fmt.Printf("[I18N] Loaded translation for %s (%s)\n", translation.Meta.NativeName, langCode)
	}

	return nil
}

// GetTranslation retrieves a translation for a given language and key
// Key can be nested using dot notation, e.g., "ui.header.tagline" or "register.title"
func (m *Manager) GetTranslation(lang, key string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	translation, exists := m.translations[lang]
	if !exists {
		// Fallback to default language
		translation, exists = m.translations[m.defaultLang]
		if !exists {
			return key // Return key if no translation found
		}
	}

	// Split key by dots to navigate nested structure
	keys := strings.Split(key, ".")
	var current interface{} = translation.Raw // Use Raw data instead of just UI

	for _, k := range keys {
		switch v := current.(type) {
		case map[string]interface{}:
			if val, ok := v[k]; ok {
				current = val
			} else {
				return key // Key not found, return original key
			}
		default:
			return key // Not a map, can't navigate further
		}
	}

	// Convert final result to string
	if str, ok := current.(string); ok {
		return str
	}

	return key // Return key if conversion fails
}

// GetAvailableLanguages returns a list of all available language codes
func (m *Manager) GetAvailableLanguages() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	languages := make([]string, 0, len(m.translations))
	for langCode := range m.translations {
		languages = append(languages, langCode)
	}

	return languages
}

// GetLanguageMeta returns metadata for all available languages
func (m *Manager) GetLanguageMeta() map[string]TranslationMeta {
	m.mu.RLock()
	defer m.mu.RUnlock()

	meta := make(map[string]TranslationMeta)
	for langCode, translation := range m.translations {
		meta[langCode] = translation.Meta
	}

	return meta
}

// T returns a translated string using dot notation (e.g., "ui.header.tagline", "register.title", "errors.notFound")
func (m *Manager) T(langCode, key string) string {
	return m.GetTranslation(langCode, key)
}

// DetectLanguageFromAccept parses Accept-Language header and returns best match
func (m *Manager) DetectLanguageFromAccept(acceptLanguage string) string {
	if acceptLanguage == "" {
		return m.defaultLang
	}

	// Simple language detection - look for exact matches or language prefix
	languages := strings.Split(acceptLanguage, ",")
	availableLanguages := m.GetAvailableLanguages()

	for _, lang := range languages {
		// Clean up the language tag (remove quality values, etc.)
		lang = strings.TrimSpace(strings.Split(lang, ";")[0])
		lang = strings.ToLower(lang)

		// Check for exact match
		for _, available := range availableLanguages {
			if lang == available {
				return available
			}
		}

		// Check for language prefix match (e.g., "cs" matches "cs-CZ")
		langPrefix := strings.Split(lang, "-")[0]
		for _, available := range availableLanguages {
			if langPrefix == available {
				return available
			}
		}
	}

	return m.defaultLang
}
