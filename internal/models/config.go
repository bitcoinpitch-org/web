package models

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// ConfigDataType represents the data type of a configuration value
type ConfigDataType string

const (
	ConfigDataTypeString  ConfigDataType = "string"
	ConfigDataTypeInteger ConfigDataType = "integer"
	ConfigDataTypeBoolean ConfigDataType = "boolean"
	ConfigDataTypeJSON    ConfigDataType = "json"
)

// ConfigSetting represents a configuration setting
type ConfigSetting struct {
	BaseModel
	Key         string         `json:"key" db:"key"`
	Value       string         `json:"value" db:"value"`
	Description *string        `json:"description" db:"description"`
	Category    string         `json:"category" db:"category"`
	DataType    ConfigDataType `json:"data_type" db:"data_type"`
	UpdatedBy   *uuid.UUID     `json:"updated_by" db:"updated_by"`
}

// NewConfigSetting creates a new configuration setting
func NewConfigSetting(key, value, description, category string, dataType ConfigDataType, updatedBy *uuid.UUID) *ConfigSetting {
	now := time.Now()
	return &ConfigSetting{
		BaseModel: BaseModel{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		Key:         key,
		Value:       value,
		Description: &description,
		Category:    category,
		DataType:    dataType,
		UpdatedBy:   updatedBy,
	}
}

// GetStringValue returns the configuration value as a string
func (cs *ConfigSetting) GetStringValue() string {
	return cs.Value
}

// GetIntValue returns the configuration value as an integer
func (cs *ConfigSetting) GetIntValue() (int, error) {
	return strconv.Atoi(cs.Value)
}

// GetBoolValue returns the configuration value as a boolean
func (cs *ConfigSetting) GetBoolValue() (bool, error) {
	return strconv.ParseBool(cs.Value)
}

// GetJSONValue returns the configuration value as parsed JSON
func (cs *ConfigSetting) GetJSONValue(v interface{}) error {
	return json.Unmarshal([]byte(cs.Value), v)
}

// SetValue updates the configuration value and timestamp
func (cs *ConfigSetting) SetValue(value string, updatedBy *uuid.UUID) {
	cs.Value = value
	cs.UpdatedBy = updatedBy
	cs.UpdatedAt = time.Now()
}

// ConfigAuditAction represents the type of audit action
type ConfigAuditAction string

const (
	ConfigAuditActionCreated ConfigAuditAction = "created"
	ConfigAuditActionUpdated ConfigAuditAction = "updated"
	ConfigAuditActionDeleted ConfigAuditAction = "deleted"
)

// ConfigAuditLog represents a configuration change audit log entry
type ConfigAuditLog struct {
	BaseModel
	ConfigKey            string            `json:"config_key" db:"config_key"`
	OldValue             *string           `json:"old_value" db:"old_value"`
	NewValue             *string           `json:"new_value" db:"new_value"`
	ChangedBy            uuid.UUID         `json:"changed_by" db:"changed_by"`
	ChangedAt            time.Time         `json:"changed_at" db:"changed_at"`
	Action               ConfigAuditAction `json:"action" db:"action"`
	ChangedByEmail       *string           `json:"changed_by_email" db:"changed_by_email"`
	ChangedByUsername    *string           `json:"changed_by_username" db:"changed_by_username"`
	ChangedByDisplayName *string           `json:"changed_by_display_name" db:"changed_by_display_name"`
}

// NewConfigAuditLog creates a new configuration audit log entry
func NewConfigAuditLog(configKey string, oldValue, newValue *string, changedBy uuid.UUID, action ConfigAuditAction) *ConfigAuditLog {
	now := time.Now()
	return &ConfigAuditLog{
		BaseModel: BaseModel{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		ConfigKey: configKey,
		OldValue:  oldValue,
		NewValue:  newValue,
		ChangedBy: changedBy,
		ChangedAt: now,
		Action:    action,
	}
}

// ConfigCategory represents different categories of configuration settings
type ConfigCategory struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
}

// GetConfigCategories returns all available configuration categories
func GetConfigCategories() []ConfigCategory {
	return []ConfigCategory{
		{
			Name:        "pitch_limits",
			DisplayName: "Pitch Limits",
			Description: "Character limits for different types of pitches",
		},
		{
			Name:        "security",
			DisplayName: "Security",
			Description: "Rate limiting and security settings",
		},
		{
			Name:        "users",
			DisplayName: "User Settings",
			Description: "User registration and permissions",
		},
		{
			Name:        "moderation",
			DisplayName: "Content Moderation",
			Description: "Content approval and moderation settings",
		},
		{
			Name:        "site",
			DisplayName: "Site Settings",
			Description: "General site configuration",
		},
		{
			Name:        "i18n",
			DisplayName: "Internationalization",
			Description: "Language and translation settings",
		},
		{
			Name:        "footer",
			DisplayName: "Footer Links",
			Description: "Manage footer navigation links and sections",
		},
	}
}
