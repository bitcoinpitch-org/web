package models

import (
	"fmt"
	"time"
)

// Language represents a language in the system
type Language struct {
	BaseModel
	Code         string `json:"code" db:"code"`                   // ISO language code
	NameEnglish  string `json:"name_english" db:"name_english"`   // English name
	NameNative   string `json:"name_native" db:"name_native"`     // Native name
	FlagEmoji    string `json:"flag_emoji" db:"flag_emoji"`       // Flag emoji
	UsageCount   int    `json:"usage_count" db:"usage_count"`     // Number of pitches
	IsMajor      bool   `json:"is_major" db:"is_major"`           // Major language flag
	DisplayOrder int    `json:"display_order" db:"display_order"` // Custom ordering
}

// NewLanguage creates a new language
func NewLanguage(code, nameEnglish, nameNative, flagEmoji string, isMajor bool, displayOrder int) *Language {
	now := time.Now()
	return &Language{
		BaseModel: BaseModel{
			CreatedAt: now,
			UpdatedAt: now,
		},
		Code:         code,
		NameEnglish:  nameEnglish,
		NameNative:   nameNative,
		FlagEmoji:    flagEmoji,
		IsMajor:      isMajor,
		DisplayOrder: displayOrder,
		UsageCount:   0,
	}
}

// IncrementUsage increments the usage count
func (l *Language) IncrementUsage() {
	l.UsageCount++
	l.UpdatedAt = time.Now()
}

// DecrementUsage decrements the usage count
func (l *Language) DecrementUsage() {
	if l.UsageCount > 0 {
		l.UsageCount--
		l.UpdatedAt = time.Now()
	}
}

// GetDisplayName returns the display name with flag emoji
func (l *Language) GetDisplayName() string {
	if l.FlagEmoji != "" {
		return l.FlagEmoji + " " + l.NameNative
	}
	return l.NameNative
}

// GetDisplayNameWithUsage returns display name with usage count for dropdown
func (l *Language) GetDisplayNameWithUsage() string {
	if l.UsageCount > 0 {
		return fmt.Sprintf("%s (%d pitches)", l.GetDisplayName(), l.UsageCount)
	}
	return l.GetDisplayName()
}
