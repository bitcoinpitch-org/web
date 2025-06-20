package config

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"bitcoinpitch.org/internal/models"
	"github.com/google/uuid"
)

// ConfigRepository interface for configuration database operations
type ConfigRepository interface {
	GetConfigSetting(ctx context.Context, key string) (*models.ConfigSetting, error)
	GetConfigSettingsByCategory(ctx context.Context, category string) ([]*models.ConfigSetting, error)
	GetAllConfigSettings(ctx context.Context) ([]*models.ConfigSetting, error)
	CreateConfigSetting(ctx context.Context, setting *models.ConfigSetting) error
	UpdateConfigSetting(ctx context.Context, setting *models.ConfigSetting) error
	DeleteConfigSetting(ctx context.Context, key string) error
	CreateConfigAuditLog(ctx context.Context, log *models.ConfigAuditLog) error
	GetConfigAuditLogs(ctx context.Context, configKey string, limit, offset int) ([]*models.ConfigAuditLog, error)
}

// Service manages configuration settings with caching
type Service struct {
	repo  ConfigRepository
	cache map[string]*models.ConfigSetting
	mutex sync.RWMutex
}

// NewService creates a new configuration service
func NewService(repo ConfigRepository) *Service {
	return &Service{
		repo:  repo,
		cache: make(map[string]*models.ConfigSetting),
	}
}

// RefreshCache loads all configuration settings into memory cache
func (s *Service) RefreshCache(ctx context.Context) error {
	settings, err := s.repo.GetAllConfigSettings(ctx)
	if err != nil {
		return fmt.Errorf("failed to load config settings: %w", err)
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Clear existing cache
	s.cache = make(map[string]*models.ConfigSetting)

	// Load settings into cache
	for _, setting := range settings {
		s.cache[setting.Key] = setting
	}

	return nil
}

// GetString returns a configuration value as a string
func (s *Service) GetString(ctx context.Context, key string, defaultValue string) string {
	s.mutex.RLock()
	setting, exists := s.cache[key]
	s.mutex.RUnlock()

	if !exists {
		// Try to load from database
		if dbSetting, err := s.repo.GetConfigSetting(ctx, key); err == nil {
			s.mutex.Lock()
			s.cache[key] = dbSetting
			s.mutex.Unlock()
			return dbSetting.GetStringValue()
		}
		return defaultValue
	}

	return setting.GetStringValue()
}

// GetInt returns a configuration value as an integer
func (s *Service) GetInt(ctx context.Context, key string, defaultValue int) int {
	s.mutex.RLock()
	setting, exists := s.cache[key]
	s.mutex.RUnlock()

	if !exists {
		// Try to load from database
		if dbSetting, err := s.repo.GetConfigSetting(ctx, key); err == nil {
			s.mutex.Lock()
			s.cache[key] = dbSetting
			s.mutex.Unlock()
			if value, err := dbSetting.GetIntValue(); err == nil {
				return value
			}
		}
		return defaultValue
	}

	if value, err := setting.GetIntValue(); err == nil {
		return value
	}
	return defaultValue
}

// GetBool returns a configuration value as a boolean
func (s *Service) GetBool(ctx context.Context, key string, defaultValue bool) bool {
	s.mutex.RLock()
	setting, exists := s.cache[key]
	s.mutex.RUnlock()

	if !exists {
		// Try to load from database
		if dbSetting, err := s.repo.GetConfigSetting(ctx, key); err == nil {
			s.mutex.Lock()
			s.cache[key] = dbSetting
			s.mutex.Unlock()
			if value, err := dbSetting.GetBoolValue(); err == nil {
				return value
			}
		}
		return defaultValue
	}

	if value, err := setting.GetBoolValue(); err == nil {
		return value
	}
	return defaultValue
}

// GetJSON returns a configuration value as parsed JSON
func (s *Service) GetJSON(ctx context.Context, key string, target interface{}, defaultValue interface{}) error {
	s.mutex.RLock()
	setting, exists := s.cache[key]
	s.mutex.RUnlock()

	if !exists {
		// Try to load from database
		if dbSetting, err := s.repo.GetConfigSetting(ctx, key); err == nil {
			s.mutex.Lock()
			s.cache[key] = dbSetting
			s.mutex.Unlock()
			return dbSetting.GetJSONValue(target)
		}
		// Set default value
		if defaultValue != nil {
			if data, err := json.Marshal(defaultValue); err == nil {
				return json.Unmarshal(data, target)
			}
		}
		return fmt.Errorf("config key %s not found and no default provided", key)
	}

	return setting.GetJSONValue(target)
}

// SetString sets a configuration value as a string
func (s *Service) SetString(ctx context.Context, key, value string, updatedBy uuid.UUID) error {
	return s.setValue(ctx, key, value, updatedBy)
}

// SetInt sets a configuration value as an integer
func (s *Service) SetInt(ctx context.Context, key string, value int, updatedBy uuid.UUID) error {
	return s.setValue(ctx, key, strconv.Itoa(value), updatedBy)
}

// SetBool sets a configuration value as a boolean
func (s *Service) SetBool(ctx context.Context, key string, value bool, updatedBy uuid.UUID) error {
	return s.setValue(ctx, key, strconv.FormatBool(value), updatedBy)
}

// SetJSON sets a configuration value as JSON
func (s *Service) SetJSON(ctx context.Context, key string, value interface{}, updatedBy uuid.UUID) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return s.setValue(ctx, key, string(data), updatedBy)
}

// setValue is the internal method to update configuration values
func (s *Service) setValue(ctx context.Context, key, value string, updatedBy uuid.UUID) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var oldValue *string

	// Get existing setting for audit log
	if existing, exists := s.cache[key]; exists {
		oldVal := existing.GetStringValue()
		oldValue = &oldVal

		// Update existing setting
		existing.SetValue(value, &updatedBy)
		if err := s.repo.UpdateConfigSetting(ctx, existing); err != nil {
			return fmt.Errorf("failed to update config setting: %w", err)
		}
	} else {
		// Create new setting - we need to determine the category and data type
		// For now, we'll use defaults - in practice, this should be handled by admin UI
		setting := models.NewConfigSetting(key, value, "", "general", models.ConfigDataTypeString, &updatedBy)
		if err := s.repo.CreateConfigSetting(ctx, setting); err != nil {
			return fmt.Errorf("failed to create config setting: %w", err)
		}
		s.cache[key] = setting
	}

	// Create audit log
	auditLog := models.NewConfigAuditLog(key, oldValue, &value, updatedBy, models.ConfigAuditActionUpdated)
	if err := s.repo.CreateConfigAuditLog(ctx, auditLog); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to create audit log: %v\n", err)
	}

	return nil
}

// GetSettingsByCategory returns all settings in a category
func (s *Service) GetSettingsByCategory(ctx context.Context, category string) ([]*models.ConfigSetting, error) {
	return s.repo.GetConfigSettingsByCategory(ctx, category)
}

// GetAllSettings returns all configuration settings
func (s *Service) GetAllSettings(ctx context.Context) ([]*models.ConfigSetting, error) {
	return s.repo.GetAllConfigSettings(ctx)
}

// DeleteSetting deletes a configuration setting
func (s *Service) DeleteSetting(ctx context.Context, key string, deletedBy uuid.UUID) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var oldValue *string
	if existing, exists := s.cache[key]; exists {
		oldVal := existing.GetStringValue()
		oldValue = &oldVal
		delete(s.cache, key)
	}

	if err := s.repo.DeleteConfigSetting(ctx, key); err != nil {
		return fmt.Errorf("failed to delete config setting: %w", err)
	}

	// Create audit log
	auditLog := models.NewConfigAuditLog(key, oldValue, nil, deletedBy, models.ConfigAuditActionDeleted)
	if err := s.repo.CreateConfigAuditLog(ctx, auditLog); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to create audit log: %v\n", err)
	}

	return nil
}

// GetAuditLogs returns audit logs for a configuration key
func (s *Service) GetAuditLogs(ctx context.Context, configKey string, limit, offset int) ([]*models.ConfigAuditLog, error) {
	return s.repo.GetConfigAuditLogs(ctx, configKey, limit, offset)
}

// PitchLimits returns current pitch length limits
func (s *Service) PitchLimits(ctx context.Context) PitchLimits {
	return PitchLimits{
		OneLinerMin: s.GetInt(ctx, "pitch.one_liner.min_length", 3),
		OneLinerMax: s.GetInt(ctx, "pitch.one_liner.max_length", 30),
		SMSMax:      s.GetInt(ctx, "pitch.sms.max_length", 80),
		TweetMax:    s.GetInt(ctx, "pitch.tweet.max_length", 280),
		ElevatorMax: s.GetInt(ctx, "pitch.elevator.max_length", 1024),
	}
}

// PaginationConfig returns current pagination configuration
func (s *Service) PaginationConfig(ctx context.Context) PaginationConfig {
	var pageSizeOptions []string
	err := s.GetJSON(ctx, "pagination.page_size_options", &pageSizeOptions, []string{"10", "25", "50", "100"})

	// Convert string array to int array
	var options []int
	if err == nil {
		for _, opt := range pageSizeOptions {
			if val, err := strconv.Atoi(opt); err == nil {
				options = append(options, val)
			}
		}
	}

	// Fallback if conversion fails
	if len(options) == 0 {
		options = []int{10, 25, 50, 100}
	}

	return PaginationConfig{
		DefaultPageSize:      s.GetInt(ctx, "pagination.default_page_size", 10),
		PageSizeOptions:      options,
		MaxPageSize:          s.GetInt(ctx, "pagination.max_page_size", 100),
		ShowTotalCount:       s.GetBool(ctx, "pagination.show_total_count", true),
		ShowPageInfo:         s.GetBool(ctx, "pagination.show_page_info", true),
		ShowPageSizeSelector: s.GetBool(ctx, "pagination.show_page_size_selector", true),
	}
}

// PitchLimits holds the current pitch length configuration
type PitchLimits struct {
	OneLinerMin int `json:"one_liner_min"`
	OneLinerMax int `json:"one_liner_max"`
	SMSMax      int `json:"sms_max"`
	TweetMax    int `json:"tweet_max"`
	ElevatorMax int `json:"elevator_max"`
}

// PaginationConfig holds the current pagination configuration
type PaginationConfig struct {
	DefaultPageSize      int   `json:"default_page_size"`
	PageSizeOptions      []int `json:"page_size_options"`
	MaxPageSize          int   `json:"max_page_size"`
	ShowTotalCount       bool  `json:"show_total_count"`
	ShowPageInfo         bool  `json:"show_page_info"`
	ShowPageSizeSelector bool  `json:"show_page_size_selector"`
}

// GetFloat64 returns a configuration value as a float64
func (s *Service) GetFloat64(ctx context.Context, key string, defaultValue float64) float64 {
	s.mutex.RLock()
	setting, exists := s.cache[key]
	s.mutex.RUnlock()

	if !exists {
		// Try to load from database
		if dbSetting, err := s.repo.GetConfigSetting(ctx, key); err == nil {
			s.mutex.Lock()
			s.cache[key] = dbSetting
			s.mutex.Unlock()
			if value, err := strconv.ParseFloat(dbSetting.GetStringValue(), 64); err == nil {
				return value
			}
		}
		return defaultValue
	}

	if value, err := strconv.ParseFloat(setting.GetStringValue(), 64); err == nil {
		return value
	}
	return defaultValue
}

// GetStringSlice returns a configuration value as a slice of strings (from JSON array)
func (s *Service) GetStringSlice(ctx context.Context, key string) []string {
	var result []string
	err := s.GetJSON(ctx, key, &result, []string{})
	if err != nil {
		return []string{}
	}
	return result
}

// FooterSection represents a footer section with title and links
type FooterSection struct {
	Enabled     bool         `json:"enabled"`
	Title       string       `json:"title"`
	Description string       `json:"description,omitempty"`
	Links       []FooterLink `json:"links,omitempty"`
}

// FooterLink represents a single link in footer
type FooterLink struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	External bool   `json:"external,omitempty"`
}

// FooterConfigData holds the complete footer configuration
type FooterConfigData struct {
	AboutSection      FooterSection `json:"about_section"`
	CategoriesSection FooterSection `json:"categories_section"`
	ResourcesSection  FooterSection `json:"resources_section"`
	ConnectSection    FooterSection `json:"connect_section"`
	BottomText        string        `json:"bottom_text"`
	Copyright         string        `json:"copyright"`
}

// FooterConfig returns the complete footer configuration
func (s *Service) FooterConfig(ctx context.Context) *FooterConfigData {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[ERROR] FooterConfig panic: %v\n", r)
		}
	}()

	config := &FooterConfigData{}

	// Load about section
	if err := s.GetJSON(ctx, "footer_about_section", &config.AboutSection, FooterSection{
		Enabled:     true,
		Title:       "About BitcoinPitch.org",
		Description: "A platform for collecting and sharing Bitcoin-related pitches. Find the perfect way to explain Bitcoin, Lightning, and Cashu to anyone.",
	}); err != nil {
		fmt.Printf("[DEBUG] FooterConfig: Error loading about section: %v\n", err)
		config.AboutSection = FooterSection{
			Enabled:     true,
			Title:       "About BitcoinPitch.org",
			Description: "A platform for collecting and sharing Bitcoin-related pitches. Find the perfect way to explain Bitcoin, Lightning, and Cashu to anyone.",
		}
	}
	fmt.Printf("[DEBUG] FooterConfig: AboutSection loaded - Enabled: %v, Title: %s\n", config.AboutSection.Enabled, config.AboutSection.Title)

	// Load categories section
	if err := s.GetJSON(ctx, "footer_categories_section", &config.CategoriesSection, FooterSection{
		Enabled: true,
		Title:   "Categories",
		Links: []FooterLink{
			{Name: "Bitcoin", URL: "/bitcoin"},
			{Name: "Lightning", URL: "/lightning"},
			{Name: "Cashu", URL: "/cashu"},
		},
	}); err != nil {
		fmt.Printf("[DEBUG] FooterConfig: Error loading categories section: %v\n", err)
		config.CategoriesSection = FooterSection{
			Enabled: true,
			Title:   "Categories",
			Links: []FooterLink{
				{Name: "Bitcoin", URL: "/bitcoin"},
				{Name: "Lightning", URL: "/lightning"},
				{Name: "Cashu", URL: "/cashu"},
			},
		}
	}
	fmt.Printf("[DEBUG] FooterConfig: CategoriesSection loaded - Enabled: %v, Title: %s, Links count: %d\n",
		config.CategoriesSection.Enabled, config.CategoriesSection.Title, len(config.CategoriesSection.Links))
	for i, link := range config.CategoriesSection.Links {
		fmt.Printf("[DEBUG] FooterConfig: Categories Link %d - Name: %s, URL: %s, External: %v\n", i, link.Name, link.URL, link.External)
	}

	// Load resources section
	if err := s.GetJSON(ctx, "footer_resources_section", &config.ResourcesSection, FooterSection{
		Enabled: true,
		Title:   "Resources",
		Links: []FooterLink{
			{Name: "About", URL: "/about"},
			{Name: "Privacy Policy", URL: "/privacy"},
			{Name: "Terms of Service", URL: "/terms"},
		},
	}); err != nil {
		fmt.Printf("[DEBUG] FooterConfig: Error loading resources section: %v\n", err)
		config.ResourcesSection = FooterSection{
			Enabled: true,
			Title:   "Resources",
			Links: []FooterLink{
				{Name: "About", URL: "/about"},
				{Name: "Privacy Policy", URL: "/privacy"},
				{Name: "Terms of Service", URL: "/terms"},
			},
		}
	}
	fmt.Printf("[DEBUG] FooterConfig: ResourcesSection loaded - Enabled: %v, Title: %s, Links count: %d\n",
		config.ResourcesSection.Enabled, config.ResourcesSection.Title, len(config.ResourcesSection.Links))

	// Load connect section
	if err := s.GetJSON(ctx, "footer_connect_section", &config.ConnectSection, FooterSection{
		Enabled: true,
		Title:   "Connect",
		Links: []FooterLink{
			{Name: "Twitter", URL: "https://twitter.com/bitcoinpitch", External: true},
			{Name: "GitHub", URL: "https://github.com/bitcoinpitch/bitcoinpitch.org", External: true},
			{Name: "Nostr", URL: "https://nostr.com/npub1bitcoinpitch", External: true},
		},
	}); err != nil {
		fmt.Printf("[DEBUG] FooterConfig: Error loading connect section: %v\n", err)
		config.ConnectSection = FooterSection{
			Enabled: true,
			Title:   "Connect",
			Links: []FooterLink{
				{Name: "Twitter", URL: "https://twitter.com/bitcoinpitch", External: true},
				{Name: "GitHub", URL: "https://github.com/bitcoinpitch/bitcoinpitch.org", External: true},
				{Name: "Nostr", URL: "https://nostr.com/npub1bitcoinpitch", External: true},
			},
		}
	}
	fmt.Printf("[DEBUG] FooterConfig: ConnectSection loaded - Enabled: %v, Title: %s, Links count: %d\n",
		config.ConnectSection.Enabled, config.ConnectSection.Title, len(config.ConnectSection.Links))

	// Load bottom text and copyright
	config.BottomText = s.GetString(ctx, "footer_bottom_text", "Building a better Bitcoin narrative, one pitch at a time.")
	config.Copyright = s.GetString(ctx, "footer_copyright", "&copy; 2025 BitcoinPitch.org. All rights reserved.")

	fmt.Printf("[DEBUG] FooterConfig: BottomText: %s\n", config.BottomText)
	fmt.Printf("[DEBUG] FooterConfig: Copyright: %s\n", config.Copyright)
	fmt.Printf("[DEBUG] FooterConfig: Successfully loaded footer config with 4 sections\n")
	return config
}
