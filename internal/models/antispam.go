package models

import (
	"encoding/json"
	"net"
	"time"

	"github.com/google/uuid"
)

// ActivityType represents the type of user activity for tracking
type ActivityType string

const (
	ActivityTypePitchCreate ActivityType = "pitch_create"
	ActivityTypePitchEdit   ActivityType = "pitch_edit"
	ActivityTypePitchDelete ActivityType = "pitch_delete"
	ActivityTypeVote        ActivityType = "vote"
	ActivityTypeLogin       ActivityType = "login"
	ActivityTypeRegister    ActivityType = "register"
)

// PenaltyType represents the type of penalty applied to a user
type PenaltyType string

const (
	PenaltyTypeRateLimit          PenaltyType = "rate_limit"
	PenaltyTypeCooldown           PenaltyType = "cooldown"
	PenaltyTypeContentRestriction PenaltyType = "content_restriction"
)

// UserActivity tracks user actions for antispam monitoring
type UserActivity struct {
	BaseModel
	UserID     *uuid.UUID             `json:"user_id" db:"user_id"`
	ActionType ActivityType           `json:"action_type" db:"action_type"`
	TargetID   *uuid.UUID             `json:"target_id" db:"target_id"`
	IPAddress  *net.IP                `json:"ip_address" db:"ip_address"`
	UserAgent  *string                `json:"user_agent" db:"user_agent"`
	Metadata   map[string]interface{} `json:"metadata" db:"metadata"`
}

// UserPenalty represents a temporary penalty applied to a user
type UserPenalty struct {
	BaseModel
	UserID      uuid.UUID   `json:"user_id" db:"user_id"`
	PenaltyType PenaltyType `json:"penalty_type" db:"penalty_type"`
	Reason      string      `json:"reason" db:"reason"`
	Multiplier  float64     `json:"multiplier" db:"multiplier"`
	ExpiresAt   time.Time   `json:"expires_at" db:"expires_at"`
	CreatedBy   *uuid.UUID  `json:"created_by" db:"created_by"`
	IsActive    bool        `json:"is_active" db:"is_active"`
}

// ContentHash tracks content for duplicate detection
type ContentHash struct {
	BaseModel
	UserID          uuid.UUID `json:"user_id" db:"user_id"`
	ContentHash     string    `json:"content_hash" db:"content_hash"`
	OriginalContent string    `json:"original_content" db:"original_content"`
	PitchID         uuid.UUID `json:"pitch_id" db:"pitch_id"`
}

// NewUserActivity creates a new user activity record
func NewUserActivity(userID *uuid.UUID, actionType ActivityType, targetID *uuid.UUID, ipAddress *net.IP, userAgent *string) *UserActivity {
	return &UserActivity{
		BaseModel: BaseModel{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		UserID:     userID,
		ActionType: actionType,
		TargetID:   targetID,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Metadata:   make(map[string]interface{}),
	}
}

// SetMetadata sets a metadata field
func (ua *UserActivity) SetMetadata(key string, value interface{}) {
	if ua.Metadata == nil {
		ua.Metadata = make(map[string]interface{})
	}
	ua.Metadata[key] = value
}

// GetMetadata gets a metadata field
func (ua *UserActivity) GetMetadata(key string) interface{} {
	if ua.Metadata == nil {
		return nil
	}
	return ua.Metadata[key]
}

// NewUserPenalty creates a new user penalty
func NewUserPenalty(userID uuid.UUID, penaltyType PenaltyType, reason string, multiplier float64, duration time.Duration, createdBy *uuid.UUID) *UserPenalty {
	return &UserPenalty{
		BaseModel: BaseModel{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		UserID:      userID,
		PenaltyType: penaltyType,
		Reason:      reason,
		Multiplier:  multiplier,
		ExpiresAt:   time.Now().Add(duration),
		CreatedBy:   createdBy,
		IsActive:    true,
	}
}

// IsExpired checks if the penalty has expired
func (up *UserPenalty) IsExpired() bool {
	return time.Now().After(up.ExpiresAt)
}

// Deactivate marks the penalty as inactive
func (up *UserPenalty) Deactivate() {
	up.IsActive = false
	up.UpdatedAt = time.Now()
}

// NewContentHash creates a new content hash record
func NewContentHash(userID uuid.UUID, contentHash string, originalContent string, pitchID uuid.UUID) *ContentHash {
	return &ContentHash{
		BaseModel: BaseModel{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		UserID:          userID,
		ContentHash:     contentHash,
		OriginalContent: originalContent,
		PitchID:         pitchID,
	}
}

// AntiSpamCheck represents the result of an antispam check
type AntiSpamCheck struct {
	Allowed    bool                   `json:"allowed"`
	Reason     string                 `json:"reason,omitempty"`
	RetryAfter *time.Duration         `json:"retry_after,omitempty"`
	Penalties  []*UserPenalty         `json:"penalties,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// NewAntiSpamCheck creates a new antispam check result
func NewAntiSpamCheck(allowed bool) *AntiSpamCheck {
	return &AntiSpamCheck{
		Allowed:   allowed,
		Metadata:  make(map[string]interface{}),
		Penalties: make([]*UserPenalty, 0),
	}
}

// SetRetryAfter sets the retry after duration
func (asc *AntiSpamCheck) SetRetryAfter(duration time.Duration) *AntiSpamCheck {
	asc.RetryAfter = &duration
	return asc
}

// SetReason sets the reason for blocking
func (asc *AntiSpamCheck) SetReason(reason string) *AntiSpamCheck {
	asc.Reason = reason
	return asc
}

// AddPenalty adds a penalty to the check result
func (asc *AntiSpamCheck) AddPenalty(penalty *UserPenalty) *AntiSpamCheck {
	asc.Penalties = append(asc.Penalties, penalty)
	return asc
}

// SetMetadata sets a metadata field
func (asc *AntiSpamCheck) SetMetadata(key string, value interface{}) *AntiSpamCheck {
	if asc.Metadata == nil {
		asc.Metadata = make(map[string]interface{})
	}
	asc.Metadata[key] = value
	return asc
}

// Scan implements the sql.Scanner interface for JSONB metadata
func (ua *UserActivity) Scan(value interface{}) error {
	if value == nil {
		ua.Metadata = make(map[string]interface{})
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		ua.Metadata = make(map[string]interface{})
		return nil
	}

	return json.Unmarshal(bytes, &ua.Metadata)
}
